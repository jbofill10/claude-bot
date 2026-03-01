package workflow

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// checkReviewComments uses the GitHub CLI to check if there are unresolved
// review comments on a PR.
func (e *Engine) checkReviewComments(ctx context.Context, taskID int64, repoPath string, prNumber int) (bool, error) {
	// Use gh CLI to get PR review comments
	cmd := exec.CommandContext(ctx, "gh", "pr", "view", fmt.Sprintf("%d", prNumber), "--json", "reviewRequests,reviews,comments", "--jq", ".reviews[].body, .comments[].body")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		// If the gh command fails, assume no comments (might not have gh installed or no PR)
		return false, nil
	}

	// If there's any output, there are comments
	trimmed := strings.TrimSpace(string(output))
	return trimmed != "", nil
}
