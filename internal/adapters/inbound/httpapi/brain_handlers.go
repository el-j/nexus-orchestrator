package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"nexus-orchestrator/internal/core/domain"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleIngestKnowledge(w http.ResponseWriter, r *http.Request) {
	if s.brain == nil {
		writeJSONError(w, "brain service not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		ProjectPath string `json:"projectPath"`
		FilePath    string `json:"filePath"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.ProjectPath == "" || req.FilePath == "" {
		writeJSONError(w, "projectPath and filePath are required", http.StatusBadRequest)
		return
	}

	count, err := s.brain.IngestFromFile(r.Context(), req.ProjectPath, req.FilePath)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"ingestedSections": count})
}

func (s *Server) handleGetBrainStatus(w http.ResponseWriter, r *http.Request) {
	if s.brain == nil {
		writeJSONError(w, "brain service not configured", http.StatusServiceUnavailable)
		return
	}

	projectPath := r.URL.Query().Get("projectPath")
	if projectPath == "" {
		writeJSONError(w, "projectPath query parameter is required", http.StatusBadRequest)
		return
	}

	status, err := s.brain.GetStatus(r.Context(), projectPath)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleGetProjectContext(w http.ResponseWriter, r *http.Request) {
	if s.brain == nil {
		writeJSONError(w, "brain service not configured", http.StatusServiceUnavailable)
		return
	}

	var req domain.ContextQuery
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.ProjectPath == "" {
		writeJSONError(w, "projectPath is required", http.StatusBadRequest)
		return
	}

	if req.MaxTokens <= 0 {
		req.MaxTokens = 400
	}

	resp, err := s.brain.GetContext(r.Context(), req)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetFocusedContext(w http.ResponseWriter, r *http.Request) {
	if s.brain == nil {
		writeJSONError(w, "brain service not configured", http.StatusServiceUnavailable)
		return
	}

	var req domain.ContextQuery
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.ProjectPath == "" || req.Question == "" {
		writeJSONError(w, "projectPath and question are required", http.StatusBadRequest)
		return
	}

	if req.MaxTokens <= 0 {
		req.MaxTokens = 400
	}

	resp, err := s.brain.GetFocusedContext(r.Context(), req)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSearchKnowledge(w http.ResponseWriter, r *http.Request) {
	if s.brain == nil {
		writeJSONError(w, "brain service not configured", http.StatusServiceUnavailable)
		return
	}

	projectPath := r.URL.Query().Get("projectPath")
	query := r.URL.Query().Get("q")
	limitStr := r.URL.Query().Get("limit")

	if projectPath == "" || query == "" {
		writeJSONError(w, "projectPath and q query parameters are required", http.StatusBadRequest)
		return
	}

	limit := 5
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	results, err := s.brain.SearchKnowledge(r.Context(), projectPath, query, limit)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"results": results})
}

func (s *Server) handleInitProject(w http.ResponseWriter, r *http.Request) {
	if s.brain == nil {
		writeJSONError(w, "brain service not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		ProjectPath  string `json:"projectPath"`
		ClaudeMDPath string `json:"claudeMDPath"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.ProjectPath == "" {
		writeJSONError(w, "projectPath is required", http.StatusBadRequest)
		return
	}

	status, err := s.brain.InitProject(r.Context(), req.ProjectPath, req.ClaudeMDPath)
	if err != nil {
		writeJSONError(w, fmt.Errorf("httpapi: brain: init-project: %w", err).Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleListKnowledge(w http.ResponseWriter, r *http.Request) {
	if s.brain == nil {
		writeJSONError(w, "brain service not configured", http.StatusServiceUnavailable)
		return
	}

	projectPath := r.URL.Query().Get("projectPath")
	if projectPath == "" {
		writeJSONError(w, "projectPath query parameter is required", http.StatusBadRequest)
		return
	}
	kind := r.URL.Query().Get("kind")

	entries, err := s.brain.ListKnowledge(r.Context(), projectPath, kind)
	if err != nil {
		writeJSONError(w, fmt.Errorf("httpapi: brain: list-knowledge: %w", err).Error(), http.StatusInternalServerError)
		return
	}

	if entries == nil {
		entries = []domain.ProjectKnowledge{}
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) handleDeleteKnowledge(w http.ResponseWriter, r *http.Request) {
	if s.brain == nil {
		writeJSONError(w, "brain service not configured", http.StatusServiceUnavailable)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSONError(w, "id path parameter is required", http.StatusBadRequest)
		return
	}

	err := s.brain.DeleteKnowledge(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "knowledge entry not found", http.StatusNotFound)
			return
		}
		writeJSONError(w, fmt.Errorf("httpapi: brain: delete-knowledge: %w", err).Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetFileMap(w http.ResponseWriter, r *http.Request) {
	if s.brain == nil {
		writeJSONError(w, "brain service not configured", http.StatusServiceUnavailable)
		return
	}

	projectPath := r.URL.Query().Get("projectPath")
	if projectPath == "" {
		writeJSONError(w, "projectPath query parameter is required", http.StatusBadRequest)
		return
	}
	focusArea := r.URL.Query().Get("focusArea")

	filePaths, err := s.brain.GetFileMap(r.Context(), projectPath, focusArea)
	if err != nil {
		writeJSONError(w, fmt.Errorf("httpapi: brain: get-file-map: %w", err).Error(), http.StatusInternalServerError)
		return
	}

	if filePaths == nil {
		filePaths = []string{}
	}
	writeJSON(w, http.StatusOK, map[string][]string{"filePaths": filePaths})
}
