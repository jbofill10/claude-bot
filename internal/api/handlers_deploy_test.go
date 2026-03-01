package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"claude-bot/internal/db"
	"claude-bot/internal/workflow"
	"claude-bot/internal/ws"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestServer(t *testing.T) (*Server, *db.Queries) {
	t.Helper()
	database, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

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
			t.Fatalf("migration: %v", err)
		}
	}

	t.Cleanup(func() { database.Close() })

	q := db.NewQueries(database)
	hub := ws.NewHub()
	go hub.Run()
	engine := workflow.NewEngine(q, nil, hub)

	s := &Server{
		DB:      database,
		Queries: q,
		Hub:     hub,
		Engine:  engine,
	}
	return s, q
}

// withChiURLParam sets a chi URL param on the request context.
func withChiURLParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// withUserID adds the user ID to the request context (simulates middleware).
func withUserID(r *http.Request, userID int64) *http.Request {
	ctx := context.WithValue(r.Context(), userIDKey, userID)
	return r.WithContext(ctx)
}

func TestHandleApproveDeployTask(t *testing.T) {
	s, q := setupTestServer(t)

	user, _ := q.CreateUser("testuser")
	repo, _ := q.CreateRepo(user.ID, "repo", "/tmp/repo")
	task, _ := q.CreateTask(user.ID, repo.ID, "test task")
	_ = q.UpdateTaskStatus(task.ID, "deploy_review")

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/1/approve-deploy", nil)
	req = withChiURLParam(req, "id", "1")
	w := httptest.NewRecorder()

	s.handleApproveDeployTask(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result db.Task
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// Should be deploying or failed (failed because Runner is nil)
	if result.Status != "deploying" && result.Status != "failed" {
		t.Errorf("expected deploying or failed status, got %q", result.Status)
	}
}

func TestHandleSkipDeployTask(t *testing.T) {
	s, q := setupTestServer(t)

	user, _ := q.CreateUser("testuser")
	repo, _ := q.CreateRepo(user.ID, "repo", "/tmp/repo")
	task, _ := q.CreateTask(user.ID, repo.ID, "test task")
	_ = q.UpdateTaskStatus(task.ID, "deploy_review")

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/1/skip-deploy", nil)
	req = withChiURLParam(req, "id", "1")
	w := httptest.NewRecorder()

	s.handleSkipDeployTask(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result db.Task
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("expected completed status, got %q", result.Status)
	}
}

func TestHandleUpdateRepoDeployScript(t *testing.T) {
	s, q := setupTestServer(t)

	user, _ := q.CreateUser("testuser")
	repo, _ := q.CreateRepo(user.ID, "repo", "/tmp/repo")

	body := `{"deploy_script":"./deploy.sh"}`
	req := httptest.NewRequest(http.MethodPut, "/api/repos/1/deploy-script", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "id", "1")
	req = withUserID(req, user.ID)
	w := httptest.NewRecorder()

	s.handleUpdateRepoDeployScript(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var result db.Repo
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if result.DeployScript != "./deploy.sh" {
		t.Errorf("expected deploy_script %q, got %q", "./deploy.sh", result.DeployScript)
	}

	// Verify persisted
	updated, _ := q.GetRepo(repo.ID)
	if updated.DeployScript != "./deploy.sh" {
		t.Errorf("expected persisted deploy_script %q, got %q", "./deploy.sh", updated.DeployScript)
	}
}

func TestHandleUpdateRepoDeployScriptForbidden(t *testing.T) {
	s, q := setupTestServer(t)

	user1, _ := q.CreateUser("user1")
	user2, _ := q.CreateUser("user2")
	_, _ = q.CreateRepo(user1.ID, "repo", "/tmp/repo")

	body := `{"deploy_script":"./hack.sh"}`
	req := httptest.NewRequest(http.MethodPut, "/api/repos/1/deploy-script", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParam(req, "id", "1")
	req = withUserID(req, user2.ID)
	w := httptest.NewRecorder()

	s.handleUpdateRepoDeployScript(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}
