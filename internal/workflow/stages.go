package workflow

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func (e *Engine) runPlanning(ctx context.Context, taskID int64) error {
	task, err := e.queries.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}

	repoPath, err := e.getRepoPath(taskID)
	if err != nil {
		return fmt.Errorf("get repo path: %w", err)
	}

	// Check for prior rejection feedback
	var feedbackSection string
	messages, _ := e.queries.GetChatMessages(taskID)
	for _, m := range messages {
		if m.Role == "user" && strings.HasPrefix(m.Content, "Plan rejected.") {
			feedbackSection = fmt.Sprintf("\n\nPrevious plan was rejected. User feedback: %s", m.Content)
		}
	}

	prompt := fmt.Sprintf(`You are working in %s. The user wants: %s%s

Research the codebase thoroughly and create a detailed implementation plan.
Include what files to modify, what tests to write, and your approach.
Start your plan with a brief title and type classification (feature/fix/refactor/chore).
Format: TITLE: <title>
TYPE: <type>
Then provide the detailed plan.`, repoPath, task.Description, feedbackSection)

	output, err := e.runClaude(ctx, taskID, "planning", repoPath, prompt)
	if err != nil {
		return err
	}

	// Extract title and type from output
	title, taskType := extractTitleAndType(output)
	if title != "" {
		_ = e.queries.UpdateTaskTitle(taskID, title, taskType)
	}

	// Save plan text and generate branch name
	_ = e.queries.UpdateTaskPlan(taskID, output)

	branchName := e.generateBranchName(task)
	_ = e.queries.UpdateTaskBranch(taskID, branchName)

	// Transition to plan_review
	_ = e.queries.UpdateTaskStatus(taskID, "plan_review")
	e.broadcastStatus(taskID, "plan_review")

	return nil
}

func (e *Engine) runDeveloping(ctx context.Context, taskID int64) error {
	task, err := e.queries.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}

	repoPath, err := e.getRepoPath(taskID)
	if err != nil {
		return fmt.Errorf("get repo path: %w", err)
	}

	prompt := fmt.Sprintf(`You are working in %s. Here is the approved plan:

%s

Implement this plan. Requirements:
- Create and checkout a new branch named: %s
- Write unit tests for all changes
- After implementing, commit your changes with a conventional commit message
- After committing, push to remote
- After pushing, create a PR using: gh pr create --fill

The branch name is: %s`, repoPath, task.PlanText, task.BranchName, task.BranchName)

	output, err := e.runClaude(ctx, taskID, "developing", repoPath, prompt)
	if err != nil {
		return err
	}

	// Try to extract PR number from output
	prNum := extractPRNumber(output)
	if prNum > 0 {
		_ = e.queries.UpdateTaskPR(taskID, prNum)
	}

	// Transition to reviewing
	_ = e.queries.UpdateTaskStatus(taskID, "reviewing")
	e.broadcastStatus(taskID, "reviewing")

	// Start review polling
	return e.runReviewing(ctx, taskID)
}

const maxReviewRounds = 5

func (e *Engine) runReviewing(ctx context.Context, taskID int64) error {
	task, err := e.queries.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}

	if task.PRNumber == 0 {
		// No PR to review, go straight to merging
		_ = e.queries.UpdateTaskStatus(taskID, "merging")
		e.broadcastStatus(taskID, "merging")
		return e.runMerging(ctx, taskID)
	}

	repoPath, err := e.getRepoPath(taskID)
	if err != nil {
		return fmt.Errorf("get repo path: %w", err)
	}

	for round := 0; round < maxReviewRounds; round++ {
		// Check for review comments using the GitHub CLI
		hasComments, err := e.checkReviewComments(ctx, taskID, repoPath, task.PRNumber)
		if err != nil {
			return fmt.Errorf("check review: %w", err)
		}

		if !hasComments {
			// No comments, proceed to merge
			_ = e.queries.UpdateTaskStatus(taskID, "merging")
			e.broadcastStatus(taskID, "merging")
			return e.runMerging(ctx, taskID)
		}

		// Address review comments
		prompt := fmt.Sprintf(`You are working in %s on branch %s.
There are review comments on PR #%d. Read and address them:
1. Run: gh pr view %d --comments
2. For each comment, either fix the issue or rebut with technical reasoning.
3. If you made fixes, commit with a conventional commit message, push, and update the PR.`,
			repoPath, task.BranchName, task.PRNumber, task.PRNumber)

		_, err = e.runClaude(ctx, taskID, "reviewing", repoPath, prompt)
		if err != nil {
			return err
		}
	}

	return fmt.Errorf("exceeded maximum review rounds (%d)", maxReviewRounds)
}

func (e *Engine) runMerging(ctx context.Context, taskID int64) error {
	task, err := e.queries.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}

	repo, err := e.getRepo(taskID)
	if err != nil {
		return fmt.Errorf("get repo: %w", err)
	}

	if task.PRNumber == 0 {
		// No PR to merge — check for deploy or complete
		return e.transitionAfterMerge(taskID, repo)
	}

	prompt := fmt.Sprintf(`Merge PR #%d in %s using:
gh pr merge %d --squash --delete-branch`, task.PRNumber, repo.Path, task.PRNumber)

	_, err = e.runClaude(ctx, taskID, "merging", repo.Path, prompt)
	if err != nil {
		return err
	}

	return e.transitionAfterMerge(taskID, repo)
}

// transitionAfterMerge moves to deploy_review if a deploy script is configured,
// otherwise completes the task.
func (e *Engine) transitionAfterMerge(taskID int64, repo *db.Repo) error {
	if repo.DeployScript != "" {
		_ = e.queries.UpdateTaskStatus(taskID, "deploy_review")
		e.broadcastStatus(taskID, "deploy_review")
		return nil
	}
	_ = e.queries.UpdateTaskStatus(taskID, "completed")
	e.broadcastStatus(taskID, "completed")
	return nil
}

func (e *Engine) runDeploying(ctx context.Context, taskID int64) error {
	repo, err := e.getRepo(taskID)
	if err != nil {
		return fmt.Errorf("get repo: %w", err)
	}

	if repo.DeployScript == "" {
		_ = e.queries.UpdateTaskStatus(taskID, "completed")
		e.broadcastStatus(taskID, "completed")
		return nil
	}

	prompt := fmt.Sprintf(`You are working in %s. Run the deploy script: %s

Execute it and report the results. If there are errors, describe them clearly.`,
		repo.Path, repo.DeployScript)

	_, err = e.runClaude(ctx, taskID, "deploying", repo.Path, prompt)
	if err != nil {
		return err
	}

	_ = e.queries.UpdateTaskStatus(taskID, "completed")
	e.broadcastStatus(taskID, "completed")
	return nil
}

func extractTitleAndType(output string) (string, string) {
	lines := strings.Split(output, "\n")
	var title, taskType string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(line), "TITLE:") {
			title = strings.TrimSpace(line[6:])
		}
		if strings.HasPrefix(strings.ToUpper(line), "TYPE:") {
			taskType = strings.TrimSpace(line[5:])
		}
	}
	return title, taskType
}

func extractPRNumber(output string) int {
	// Look for patterns like "pull/123" or "#123" in PR creation output
	re := regexp.MustCompile(`(?:pull/|#)(\d+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		num, _ := strconv.Atoi(matches[1])
		return num
	}
	return 0
}
