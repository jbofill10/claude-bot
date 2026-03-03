package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"claude-bot/internal/claude"
	"claude-bot/internal/db"
	"claude-bot/internal/ws"
)

type Engine struct {
	queries *db.Queries
	runner  *claude.Runner
	hub     *ws.Hub

	mu      sync.Mutex
	cancels map[int64]context.CancelFunc
}

func NewEngine(q *db.Queries, r *claude.Runner, h *ws.Hub) *Engine {
	return &Engine{
		queries: q,
		runner:  r,
		hub:     h,
		cancels: make(map[int64]context.CancelFunc),
	}
}

func (e *Engine) Start(taskID int64) {
	task, err := e.queries.GetTask(taskID)
	if err != nil {
		log.Printf("engine: failed to get task %d: %v", taskID, err)
		return
	}
	if task.Status != "pending" && task.Status != "failed" {
		log.Printf("engine: task %d has status %s, cannot start", taskID, task.Status)
		return
	}
	e.runStage(taskID, "planning")
}

func (e *Engine) Approve(taskID int64) {
	task, err := e.queries.GetTask(taskID)
	if err != nil {
		log.Printf("engine: failed to get task %d: %v", taskID, err)
		return
	}
	if task.Status != "plan_review" {
		log.Printf("engine: task %d has status %s, cannot approve", taskID, task.Status)
		return
	}
	e.runStage(taskID, "developing")
}

func (e *Engine) Reject(taskID int64, feedback string) {
	task, err := e.queries.GetTask(taskID)
	if err != nil {
		log.Printf("engine: failed to get task %d: %v", taskID, err)
		return
	}
	if task.Status != "plan_review" {
		log.Printf("engine: task %d has status %s, cannot reject", taskID, task.Status)
		return
	}

	// Append feedback as chat message for context
	_ = e.queries.CreateChatMessage(taskID, "user", "Plan rejected. Feedback: "+feedback)
	e.runStage(taskID, "planning")
}

func (e *Engine) ApproveDeploy(taskID int64) {
	task, err := e.queries.GetTask(taskID)
	if err != nil {
		log.Printf("engine: failed to get task %d: %v", taskID, err)
		return
	}
	if task.Status != "deploy_review" {
		log.Printf("engine: task %d has status %s, cannot approve deploy", taskID, task.Status)
		return
	}
	e.runStage(taskID, "deploying")
}

func (e *Engine) SkipDeploy(taskID int64) {
	task, err := e.queries.GetTask(taskID)
	if err != nil {
		log.Printf("engine: failed to get task %d: %v", taskID, err)
		return
	}
	if task.Status != "deploy_review" {
		log.Printf("engine: task %d has status %s, cannot skip deploy", taskID, task.Status)
		return
	}
	_ = e.queries.UpdateTaskStatus(taskID, "completed")
	e.broadcastStatus(taskID, "completed")
}

func (e *Engine) Cancel(taskID int64) error {
	task, err := e.queries.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}

	// Only allow cancelling non-terminal tasks
	switch task.Status {
	case "completed", "failed", "cancelled":
		return fmt.Errorf("task %d has status %s, cannot cancel", taskID, task.Status)
	}

	// Cancel the context (stops the goroutine in runStage)
	e.mu.Lock()
	cancelFn, hasCancel := e.cancels[taskID]
	e.mu.Unlock()

	if hasCancel {
		cancelFn()
	}

	// Also kill the process directly for immediate effect
	e.runner.Kill(taskID)

	// Update DB status
	_ = e.queries.UpdateTaskStatus(taskID, "cancelled")
	e.broadcastStatus(taskID, "cancelled")
	return nil
}

func (e *Engine) Retry(taskID int64) {
	task, err := e.queries.GetTask(taskID)
	if err != nil {
		log.Printf("engine: failed to get task %d: %v", taskID, err)
		return
	}
	if task.Status != "failed" && task.Status != "cancelled" {
		log.Printf("engine: task %d has status %s, cannot retry", taskID, task.Status)
		return
	}

	// Determine which stage to retry based on error context
	logs, _ := e.queries.GetTaskLogs(taskID)
	lastStage := "planning"
	if len(logs) > 0 {
		lastStage = logs[len(logs)-1].Stage
	}
	e.runStage(taskID, lastStage)
}

func (e *Engine) runStage(taskID int64, stage string) {
	if err := e.queries.UpdateTaskStatus(taskID, stage); err != nil {
		log.Printf("engine: failed to update task %d status to %s: %v", taskID, stage, err)
		return
	}

	e.broadcastStatus(taskID, stage)

	ctx, cancel := context.WithCancel(context.Background())
	e.mu.Lock()
	e.cancels[taskID] = cancel
	e.mu.Unlock()

	go func() {
		defer func() {
			e.mu.Lock()
			delete(e.cancels, taskID)
			e.mu.Unlock()
			cancel()
		}()

		var err error
		switch stage {
		case "planning":
			err = e.runPlanning(ctx, taskID)
		case "developing":
			err = e.runDeveloping(ctx, taskID)
		case "reviewing":
			err = e.runReviewing(ctx, taskID)
		case "merging":
			err = e.runMerging(ctx, taskID)
		case "deploying":
			err = e.runDeploying(ctx, taskID)
		default:
			err = fmt.Errorf("unknown stage: %s", stage)
		}

		if err != nil {
			// Check if task was cancelled (don't overwrite cancelled with failed)
			if t, e2 := e.queries.GetTask(taskID); e2 == nil && t.Status == "cancelled" {
				log.Printf("engine: task %d stage %s was cancelled", taskID, stage)
				return
			}
			log.Printf("engine: task %d stage %s failed: %v", taskID, stage, err)
			_ = e.queries.UpdateTaskError(taskID, err.Error())
			e.broadcastStatus(taskID, "failed")
		}
	}()
}

func (e *Engine) runClaude(ctx context.Context, taskID int64, stage, repoPath, prompt string) (string, error) {
	if e.runner == nil {
		return "", fmt.Errorf("claude runner is not configured")
	}

	onEvent := func(ev claude.Event) {
		// Store log asynchronously — don't block the broadcast.
		go func() {
			_ = e.queries.CreateTaskLog(taskID, stage, ev.Content)
		}()

		// Broadcast to WebSocket clients
		msg, _ := json.Marshal(map[string]interface{}{
			"type":    "output",
			"stage":   stage,
			"content": ev.Content,
			"raw":     json.RawMessage(ev.Raw),
		})
		e.hub.Broadcast(taskID, msg)
	}

	result := e.runner.Run(ctx, taskID, repoPath, prompt, onEvent)
	if result.Err != nil {
		return "", fmt.Errorf("claude process error: %w", result.Err)
	}
	if result.ExitCode != 0 {
		stderr := strings.TrimSpace(result.Stderr)
		if len(stderr) > 2000 {
			stderr = stderr[:2000]
		}
		if stderr != "" {
			return result.Output, fmt.Errorf("claude exited with code %d: %s", result.ExitCode, stderr)
		}
		return result.Output, fmt.Errorf("claude exited with code %d", result.ExitCode)
	}
	return result.Output, nil
}

func (e *Engine) broadcastStatus(taskID int64, status string) {
	msg, _ := json.Marshal(map[string]string{
		"type":   "status",
		"status": status,
	})
	e.hub.Broadcast(taskID, msg)
}

func (e *Engine) getRepoPath(taskID int64) (string, error) {
	task, err := e.queries.GetTask(taskID)
	if err != nil {
		return "", err
	}
	repo, err := e.queries.GetRepo(task.RepoID)
	if err != nil {
		return "", err
	}
	return repo.Path, nil
}

func (e *Engine) getRepo(taskID int64) (*db.Repo, error) {
	task, err := e.queries.GetTask(taskID)
	if err != nil {
		return nil, err
	}
	return e.queries.GetRepo(task.RepoID)
}

func (e *Engine) generateBranchName(task *db.Task) string {
	// Generate a branch name from the description
	name := strings.ToLower(task.Description)
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		if r == ' ' || r == '-' || r == '_' {
			return '-'
		}
		return -1
	}, name)
	// Trim and truncate
	name = strings.Trim(name, "-")
	if len(name) > 50 {
		name = name[:50]
	}
	return fmt.Sprintf("claude-bot/task-%d-%s", task.ID, name)
}
