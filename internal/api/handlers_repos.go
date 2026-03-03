package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"claude-bot/internal/db"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleListRepos(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)

	repos, err := s.Queries.ListRepos(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list repos")
		return
	}
	if repos == nil {
		repos = []db.Repo{}
	}
	writeJSON(w, http.StatusOK, repos)
}

func (s *Server) handleCreateRepo(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	// Validate path exists and is a directory
	info, err := os.Stat(req.Path)
	if err != nil || !info.IsDir() {
		writeError(w, http.StatusBadRequest, "path does not exist or is not a directory")
		return
	}

	name := filepath.Base(req.Path)

	repo, err := s.Queries.CreateRepo(userID, name, req.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create repo")
		return
	}
	writeJSON(w, http.StatusCreated, repo)
}

func (s *Server) handleDeleteRepo(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid repo id")
		return
	}

	// Verify ownership
	repo, err := s.Queries.GetRepo(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "repo not found")
		return
	}
	if repo.UserID != userID {
		writeError(w, http.StatusForbidden, "not your repo")
		return
	}

	if err := s.Queries.DeleteRepo(id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete repo")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleUpdateRepoDeployScript(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid repo id")
		return
	}

	repo, err := s.Queries.GetRepo(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "repo not found")
		return
	}
	if repo.UserID != userID {
		writeError(w, http.StatusForbidden, "not your repo")
		return
	}

	var body struct {
		DeployScript string `json:"deploy_script"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.Queries.UpdateRepoDeployScript(id, body.DeployScript); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update deploy script")
		return
	}

	updated, err := s.Queries.GetRepo(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get repo")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

type availableRepo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (s *Server) handleAvailableRepos(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to determine home directory")
		return
	}

	gitDir := filepath.Join(homeDir, "git")
	entries, err := os.ReadDir(gitDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read ~/git directory")
		return
	}

	// Get repos already added for this user
	existingRepos, err := s.Queries.ListRepos(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list existing repos")
		return
	}

	existingPaths := make(map[string]bool)
	for _, repo := range existingRepos {
		existingPaths[repo.Path] = true
	}

	available := []availableRepo{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		fullPath := filepath.Join(gitDir, entry.Name())
		if existingPaths[fullPath] {
			continue
		}
		available = append(available, availableRepo{
			Name: entry.Name(),
			Path: fullPath,
		})
	}

	writeJSON(w, http.StatusOK, available)
}
