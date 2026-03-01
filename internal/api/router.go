package api

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"claude-bot/internal/claude"
	"claude-bot/internal/db"
	"claude-bot/internal/workflow"
	"claude-bot/internal/ws"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Server struct {
	DB      *sql.DB
	Queries *db.Queries
	Hub     *ws.Hub
	Engine  *workflow.Engine
	Runner  *claude.Runner
}

func NewRouter(s *Server) http.Handler {
	r := chi.NewRouter()

	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:8080"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
	}))

	// Public routes (no user cookie required)
	r.Get("/api/users", s.handleListUsers)
	r.Post("/api/users", s.handleCreateUser)
	r.Post("/api/users/{id}/select", s.handleSelectUser)

	// WebSocket route
	r.Get("/ws/tasks/{id}", s.handleTaskWebSocket)

	// Protected routes (require user cookie)
	r.Group(func(r chi.Router) {
		r.Use(UserCookieMiddleware)

		r.Get("/api/repos", s.handleListRepos)
		r.Post("/api/repos", s.handleCreateRepo)
		r.Delete("/api/repos/{id}", s.handleDeleteRepo)
		r.Get("/api/repos/available", s.handleAvailableRepos)

		r.Get("/api/repos/{repoId}/tasks", s.handleListTasks)
		r.Post("/api/repos/{repoId}/tasks", s.handleCreateTask)

		r.Get("/api/tasks/{id}", s.handleGetTask)
		r.Post("/api/tasks/{id}/approve", s.handleApproveTask)
		r.Post("/api/tasks/{id}/reject", s.handleRejectTask)
		r.Post("/api/tasks/{id}/retry", s.handleRetryTask)
		r.Post("/api/tasks/{id}/approve-deploy", s.handleApproveDeployTask)
		r.Post("/api/tasks/{id}/skip-deploy", s.handleSkipDeployTask)
		r.Get("/api/tasks/{id}/logs", s.handleGetTaskLogs)

		r.Put("/api/repos/{id}/deploy-script", s.handleUpdateRepoDeployScript)

		r.Get("/api/settings", s.handleListSettings)
		r.Put("/api/settings/{key}", s.handleUpdateSetting)
	})

	// Serve frontend static files with SPA fallback
	frontendDir := "frontend/dist"
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// Try to serve the static file
		filePath := filepath.Join(frontendDir, path)
		if !strings.HasPrefix(path, "/api") && !strings.HasPrefix(path, "/ws") {
			if _, err := os.Stat(filePath); err == nil {
				http.ServeFile(w, r, filePath)
				return
			}
		}
		// Fallback to index.html for SPA routing
		http.ServeFile(w, r, filepath.Join(frontendDir, "index.html"))
	})

	return r
}
