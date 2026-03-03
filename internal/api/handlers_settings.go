package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// handleListSettings returns all settings as a JSON array.
func (s *Server) handleListSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.Queries.ListSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list settings")
		return
	}

	writeJSON(w, http.StatusOK, settings)
}

// handleUpdateSetting updates a single setting by key.
func (s *Server) handleUpdateSetting(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		writeError(w, http.StatusBadRequest, "setting key is required")
		return
	}

	var body struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.Queries.SetSetting(key, body.Value); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update setting")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"key": key, "value": body.Value})
}
