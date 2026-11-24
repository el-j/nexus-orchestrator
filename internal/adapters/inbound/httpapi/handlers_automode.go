package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"

	"github.com/go-chi/chi/v5"
)

// handleStartAutoMode serves POST /api/auto/start: launches a Project Auto-Mode run.
func (s *Server) handleStartAutoMode(w http.ResponseWriter, r *http.Request) {
	mgr, ok := s.orch.(ports.AutoModeManager)
	if !ok {
		writeJSONError(w, "auto-mode not available", http.StatusServiceUnavailable)
		return
	}
	var req domain.AutoModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	run, err := mgr.StartAutoMode(r.Context(), req)
	if err != nil {
		if errors.Is(err, services.ErrAutoModeActive) {
			writeJSONError(w, err.Error(), http.StatusConflict)
			return
		}
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

// handleListAutoModeRuns serves GET /api/auto: lists Auto-Mode runs, optionally filtered by ?projectPath=.
func (s *Server) handleListAutoModeRuns(w http.ResponseWriter, r *http.Request) {
	mgr, ok := s.orch.(ports.AutoModeManager)
	if !ok {
		writeJSONError(w, "auto-mode not available", http.StatusServiceUnavailable)
		return
	}
	runs := mgr.ListAutoModeRuns(r.URL.Query().Get("projectPath"))
	if runs == nil {
		runs = []domain.AutoModeRun{}
	}
	writeJSON(w, http.StatusOK, runs)
}

// handleGetAutoModeRun serves GET /api/auto/{id}: returns the live snapshot of one Auto-Mode run.
func (s *Server) handleGetAutoModeRun(w http.ResponseWriter, r *http.Request) {
	mgr, ok := s.orch.(ports.AutoModeManager)
	if !ok {
		writeJSONError(w, "auto-mode not available", http.StatusServiceUnavailable)
		return
	}
	run, err := mgr.GetAutoModeRun(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// handleStopAutoMode serves POST /api/auto/{id}/stop: stops an active Auto-Mode run.
func (s *Server) handleStopAutoMode(w http.ResponseWriter, r *http.Request) {
	mgr, ok := s.orch.(ports.AutoModeManager)
	if !ok {
		writeJSONError(w, "auto-mode not available", http.StatusServiceUnavailable)
		return
	}
	id := chi.URLParam(r, "id")
	if err := mgr.StopAutoMode(id); err != nil {
		writeJSONError(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "stopped": true})
}
