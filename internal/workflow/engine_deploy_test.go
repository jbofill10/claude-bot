package workflow

import (
	"database/sql"
	"testing"
	"time"

	"claude-bot/internal/db"
	"claude-bot/internal/ws"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestEngine creates an in-memory DB, runs migrations, and builds an
// Engine with a real Hub (running in background) and a nil Runner (stages that
// call runClaude aren't exercised in these tests).
func setupTestEngine(t *testing.T) (*db.Queries, *Engine) {
	t.Helper()
	database, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	// Run migrations via Open-like approach
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		)`,
		`CREATE TABLE IF NOT EXISTS repos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id),
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			added_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id),
			repo_id INTEGER NOT NULL REFERENCES repos(id),
			title TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			branch_name TEXT NOT NULL DEFAULT '',
			pr_number INTEGER NOT NULL DEFAULT 0,
			plan_text TEXT NOT NULL DEFAULT '',
			error_message TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		)`,
		`CREATE TABLE IF NOT EXISTS task_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id INTEGER NOT NULL REFERENCES tasks(id),
			stage TEXT NOT NULL,
			content TEXT NOT NULL,
			timestamp TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS chat_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id INTEGER NOT NULL REFERENCES tasks(id),
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		)`,
		`INSERT OR IGNORE INTO settings (key, value) VALUES ('claude_command', 'claude')`,
		`ALTER TABLE repos ADD COLUMN deploy_script TEXT NOT NULL DEFAULT ''`,
	}
	for _, m := range migrations {
		if _, err := database.Exec(m); err != nil {
			t.Fatalf("migration failed: %v", err)
		}
	}

	t.Cleanup(func() { database.Close() })

	q := db.NewQueries(database)
	hub := ws.NewHub()
	go hub.Run()

	engine := NewEngine(q, nil, hub)
	return q, engine
}

func createTestTask(t *testing.T, q *db.Queries, status string) (*db.Task, *db.Repo) {
	t.Helper()
	user, err := q.CreateUser("testuser")
	if err != nil {
		// User may already exist; try to get it
		user, _ = q.GetUser(1)
	}

	repo, err := q.CreateRepo(user.ID, "testrepo", "/tmp/testrepo")
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	task, err := q.CreateTask(user.ID, repo.ID, "test task")
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	if status != "pending" {
		_ = q.UpdateTaskStatus(task.ID, status)
		task, _ = q.GetTask(task.ID)
	}

	return task, repo
}

func TestApproveDeploy(t *testing.T) {
	q, engine := setupTestEngine(t)
	task, _ := createTestTask(t, q, "deploy_review")

	engine.ApproveDeploy(task.ID)

	// Give the goroutine a moment to update status
	time.Sleep(50 * time.Millisecond)

	updated, err := q.GetTask(task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}

	// The task should have transitioned to "deploying" (which will then fail
	// because Runner is nil, moving it to "failed")
	if updated.Status != "deploying" && updated.Status != "failed" {
		t.Errorf("expected status deploying or failed, got %q", updated.Status)
	}
}

func TestSkipDeploy(t *testing.T) {
	q, engine := setupTestEngine(t)
	task, _ := createTestTask(t, q, "deploy_review")

	engine.SkipDeploy(task.ID)

	updated, err := q.GetTask(task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}

	if updated.Status != "completed" {
		t.Errorf("expected status completed, got %q", updated.Status)
	}
}

func TestApproveDeployWrongStatus(t *testing.T) {
	q, engine := setupTestEngine(t)
	task, _ := createTestTask(t, q, "developing")

	engine.ApproveDeploy(task.ID)

	updated, err := q.GetTask(task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}

	// Should still be developing — ApproveDeploy should be a no-op
	if updated.Status != "developing" {
		t.Errorf("expected status developing (unchanged), got %q", updated.Status)
	}
}

func TestSkipDeployWrongStatus(t *testing.T) {
	q, engine := setupTestEngine(t)
	task, _ := createTestTask(t, q, "merging")

	engine.SkipDeploy(task.ID)

	updated, err := q.GetTask(task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}

	// Should still be merging — SkipDeploy should be a no-op
	if updated.Status != "merging" {
		t.Errorf("expected status merging (unchanged), got %q", updated.Status)
	}
}

func TestTransitionAfterMergeWithDeployScript(t *testing.T) {
	q, engine := setupTestEngine(t)
	task, repo := createTestTask(t, q, "merging")

	// Set deploy script on repo
	_ = q.UpdateRepoDeployScript(repo.ID, "./deploy.sh")
	repo, _ = q.GetRepo(repo.ID)

	err := engine.transitionAfterMerge(task.ID, repo)
	if err != nil {
		t.Fatalf("transitionAfterMerge: %v", err)
	}

	updated, _ := q.GetTask(task.ID)
	if updated.Status != "deploy_review" {
		t.Errorf("expected status deploy_review, got %q", updated.Status)
	}
}

func TestTransitionAfterMergeWithoutDeployScript(t *testing.T) {
	q, engine := setupTestEngine(t)
	task, repo := createTestTask(t, q, "merging")

	err := engine.transitionAfterMerge(task.ID, repo)
	if err != nil {
		t.Fatalf("transitionAfterMerge: %v", err)
	}

	updated, _ := q.GetTask(task.ID)
	if updated.Status != "completed" {
		t.Errorf("expected status completed, got %q", updated.Status)
	}
}
