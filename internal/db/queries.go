package db

import (
	"database/sql"
	"fmt"
)

type Queries struct {
	db *sql.DB
}

func NewQueries(db *sql.DB) *Queries {
	return &Queries{db: db}
}

// Users

func (q *Queries) ListUsers() ([]User, error) {
	rows, err := q.db.Query("SELECT id, username, created_at FROM users ORDER BY username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (q *Queries) CreateUser(username string) (*User, error) {
	res, err := q.db.Exec("INSERT INTO users (username) VALUES (?)", username)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return q.GetUser(id)
}

func (q *Queries) GetUser(id int64) (*User, error) {
	var u User
	err := q.db.QueryRow("SELECT id, username, created_at FROM users WHERE id = ?", id).
		Scan(&u.ID, &u.Username, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// Repos

func (q *Queries) ListRepos(userID int64) ([]Repo, error) {
	rows, err := q.db.Query("SELECT id, user_id, name, path, deploy_script, added_at FROM repos WHERE user_id = ? ORDER BY name", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var repos []Repo
	for rows.Next() {
		var r Repo
		if err := rows.Scan(&r.ID, &r.UserID, &r.Name, &r.Path, &r.DeployScript, &r.AddedAt); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

func (q *Queries) CreateRepo(userID int64, name, path string) (*Repo, error) {
	res, err := q.db.Exec("INSERT INTO repos (user_id, name, path) VALUES (?, ?, ?)", userID, name, path)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return q.GetRepo(id)
}

func (q *Queries) GetRepo(id int64) (*Repo, error) {
	var r Repo
	err := q.db.QueryRow("SELECT id, user_id, name, path, deploy_script, added_at FROM repos WHERE id = ?", id).
		Scan(&r.ID, &r.UserID, &r.Name, &r.Path, &r.DeployScript, &r.AddedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (q *Queries) UpdateRepoDeployScript(id int64, script string) error {
	_, err := q.db.Exec("UPDATE repos SET deploy_script = ? WHERE id = ?", script, id)
	return err
}

func (q *Queries) DeleteRepo(id int64) error {
	_, err := q.db.Exec("DELETE FROM repos WHERE id = ?", id)
	return err
}

// Tasks

func (q *Queries) ListTasks(repoID int64) ([]Task, error) {
	rows, err := q.db.Query(`SELECT id, user_id, repo_id, title, description, type, status,
		branch_name, pr_number, plan_text, error_message, created_at, updated_at
		FROM tasks WHERE repo_id = ? ORDER BY created_at DESC`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.UserID, &t.RepoID, &t.Title, &t.Description, &t.Type, &t.Status,
			&t.BranchName, &t.PRNumber, &t.PlanText, &t.ErrorMessage, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (q *Queries) CreateTask(userID, repoID int64, description string) (*Task, error) {
	res, err := q.db.Exec("INSERT INTO tasks (user_id, repo_id, description) VALUES (?, ?, ?)", userID, repoID, description)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return q.GetTask(id)
}

func (q *Queries) GetTask(id int64) (*Task, error) {
	var t Task
	err := q.db.QueryRow(`SELECT id, user_id, repo_id, title, description, type, status,
		branch_name, pr_number, plan_text, error_message, created_at, updated_at
		FROM tasks WHERE id = ?`, id).
		Scan(&t.ID, &t.UserID, &t.RepoID, &t.Title, &t.Description, &t.Type, &t.Status,
			&t.BranchName, &t.PRNumber, &t.PlanText, &t.ErrorMessage, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (q *Queries) UpdateTaskStatus(id int64, status string) error {
	_, err := q.db.Exec("UPDATE tasks SET status = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?", status, id)
	return err
}

func (q *Queries) UpdateTaskPlan(id int64, planText string) error {
	_, err := q.db.Exec("UPDATE tasks SET plan_text = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?", planText, id)
	return err
}

func (q *Queries) UpdateTaskBranch(id int64, branchName string) error {
	_, err := q.db.Exec("UPDATE tasks SET branch_name = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?", branchName, id)
	return err
}

func (q *Queries) UpdateTaskPR(id int64, prNumber int) error {
	_, err := q.db.Exec("UPDATE tasks SET pr_number = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?", prNumber, id)
	return err
}

func (q *Queries) UpdateTaskError(id int64, errMsg string) error {
	_, err := q.db.Exec("UPDATE tasks SET error_message = ?, status = 'failed', updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?", errMsg, id)
	return err
}

func (q *Queries) UpdateTaskTitle(id int64, title, taskType string) error {
	_, err := q.db.Exec("UPDATE tasks SET title = ?, type = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?", title, taskType, id)
	return err
}

// Task Logs

func (q *Queries) CreateTaskLog(taskID int64, stage, content string) error {
	_, err := q.db.Exec("INSERT INTO task_logs (task_id, stage, content) VALUES (?, ?, ?)", taskID, stage, content)
	return err
}

func (q *Queries) GetTaskLogs(taskID int64) ([]TaskLog, error) {
	rows, err := q.db.Query("SELECT id, task_id, stage, content, timestamp FROM task_logs WHERE task_id = ? ORDER BY timestamp", taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []TaskLog
	for rows.Next() {
		var l TaskLog
		if err := rows.Scan(&l.ID, &l.TaskID, &l.Stage, &l.Content, &l.Timestamp); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// Settings

func (q *Queries) GetSetting(key string) (string, error) {
	var value string
	err := q.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("setting not found: %s", key)
	}
	return value, err
}

func (q *Queries) SetSetting(key, value string) error {
	_, err := q.db.Exec("INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = ?", key, value, value)
	return err
}

func (q *Queries) ListSettings() ([]Setting, error) {
	rows, err := q.db.Query("SELECT key, value FROM settings ORDER BY key")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var settings []Setting
	for rows.Next() {
		var s Setting
		if err := rows.Scan(&s.Key, &s.Value); err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}
	return settings, rows.Err()
}

// Chat Messages

func (q *Queries) CreateChatMessage(taskID int64, role, content string) error {
	_, err := q.db.Exec("INSERT INTO chat_messages (task_id, role, content) VALUES (?, ?, ?)", taskID, role, content)
	return err
}

func (q *Queries) GetChatMessages(taskID int64) ([]ChatMessage, error) {
	rows, err := q.db.Query("SELECT id, task_id, role, content, created_at FROM chat_messages WHERE task_id = ? ORDER BY created_at", taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var messages []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.ID, &m.TaskID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}
