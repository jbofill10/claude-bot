package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"claude-bot/internal/db"
	"claude-bot/internal/ws"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

// handleListTasks returns all tasks for a given repo.
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	repoID, err := strconv.ParseInt(chi.URLParam(r, "repoId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid repo id")
		return
	}

	tasks, err := s.Queries.ListTasks(repoID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}
	if tasks == nil {
		tasks = []db.Task{}
	}

	writeJSON(w, http.StatusOK, tasks)
}

// handleCreateTask creates a new task for the authenticated user in the given repo
// and kicks off the workflow engine asynchronously.
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	repoID, err := strconv.ParseInt(chi.URLParam(r, "repoId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid repo id")
		return
	}

	var body struct {
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Description == "" {
		writeError(w, http.StatusBadRequest, "description is required")
		return
	}

	userID := GetUserID(r)

	task, err := s.Queries.CreateTask(userID, repoID, body.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	go s.Engine.Start(task.ID)

	writeJSON(w, http.StatusCreated, task)
}

// handleGetTask returns a single task by ID.
func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	task, err := s.Queries.GetTask(taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// handleApproveTask approves a task via the workflow engine and returns the updated task.
func (s *Server) handleApproveTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	s.Engine.Approve(taskID)

	task, err := s.Queries.GetTask(taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// handleRejectTask rejects a task with feedback via the workflow engine and returns the updated task.
func (s *Server) handleRejectTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	var body struct {
		Feedback string `json:"feedback"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	s.Engine.Reject(taskID, body.Feedback)

	task, err := s.Queries.GetTask(taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// handleRetryTask retries a task via the workflow engine and returns the updated task.
func (s *Server) handleRetryTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	s.Engine.Retry(taskID)

	task, err := s.Queries.GetTask(taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// handleApproveDeployTask approves deployment via the workflow engine.
func (s *Server) handleApproveDeployTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	s.Engine.ApproveDeploy(taskID)

	task, err := s.Queries.GetTask(taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// handleSkipDeployTask skips deployment via the workflow engine.
func (s *Server) handleSkipDeployTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	s.Engine.SkipDeploy(taskID)

	task, err := s.Queries.GetTask(taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// handleCancelTask cancels a running task via the workflow engine.
func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	if err := s.Engine.Cancel(taskID); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	task, err := s.Queries.GetTask(taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// handleGetTaskLogs returns all logs for a given task.
func (s *Server) handleGetTaskLogs(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	logs, err := s.Queries.GetTaskLogs(taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get task logs")
		return
	}

	writeJSON(w, http.StatusOK, logs)
}

// upgrader is configured to allow all origins for WebSocket connections.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// handleTaskWebSocket upgrades the HTTP connection to a WebSocket and registers
// the client with the hub for real-time task updates.
func (s *Server) handleTaskWebSocket(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := ws.NewClient(s.Hub, conn, taskID)
	s.Hub.Register(taskID, client)

	go client.ReadPump()
	go client.WritePump()
}
