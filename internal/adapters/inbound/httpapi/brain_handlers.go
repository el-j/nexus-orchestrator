package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

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
		switch {
		case errors.Is(err, domain.ErrPathNotAllowed):
			writeJSONError(w, err.Error(), http.StatusForbidden)
		case errors.Is(err, os.ErrNotExist):
			writeJSONError(w, "file not found", http.StatusNotFound)
		default:
			log.Printf("httpapi: ingest %q: %v", req.FilePath, err)
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"ingestedSections": count})
}

// contentIngester is implemented by brain services that can ingest markdown sent
// in the request instead of read from disk. It is optional so the BrainService
// port, and every implementation of it, stays unchanged.
type contentIngester interface {
	IngestContent(ctx context.Context, projectPath, name, content string) (int, error)
}

// handleIngestContent ingests markdown supplied in the request body. Browsers
// cannot give the daemon a file path, so the UI posts the file's content instead.
func (s *Server) handleIngestContent(w http.ResponseWriter, r *http.Request) {
	if s.brain == nil {
		writeJSONError(w, "brain service not configured", http.StatusServiceUnavailable)
		return
	}
	ci, ok := s.brain.(contentIngester)
	if !ok {
		writeJSONError(w, "this brain service cannot ingest uploaded content", http.StatusNotImplemented)
		return
	}

	// MaxBytesReader stops a huge upload before it is buffered; JSON escaping can
	// roughly double the size, hence the headroom over the content limit.
	r.Body = http.MaxBytesReader(w, r.Body, 2*domain.MaxIngestBytes+4096)
	var req struct {
		ProjectPath string `json:"projectPath"`
		Name        string `json:"name"`
		Content     string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeJSONError(w, "content is too large", http.StatusRequestEntityTooLarge)
			return
		}
		writeJSONError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.ProjectPath == "" || req.Content == "" {
		writeJSONError(w, "projectPath and content are required", http.StatusBadRequest)
		return
	}

	count, err := ci.IngestContent(r.Context(), req.ProjectPath, req.Name, req.Content)
	if err != nil {
		if errors.Is(err, domain.ErrInputTooLarge) {
			writeJSONError(w, "content is too large", http.StatusRequestEntityTooLarge)
			return
		}
		log.Printf("httpapi: ingest content %q: %v", req.Name, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
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

func (s *Server) handleGetOnboardingContext(w http.ResponseWriter, r *http.Request) {
	if s.brain == nil {
		writeJSONError(w, "brain service not configured", http.StatusServiceUnavailable)
		return
	}

	projectPath := r.URL.Query().Get("projectPath")
	if projectPath == "" {
		writeJSONError(w, "projectPath query parameter is required", http.StatusBadRequest)
		return
	}

	maxTokens := 800
	if mtStr := r.URL.Query().Get("maxTokens"); mtStr != "" {
		if parsed, err := strconv.Atoi(mtStr); err == nil && parsed > 0 {
			maxTokens = parsed
		}
	}

	summary, err := s.brain.GetOnboardingContext(r.Context(), projectPath, maxTokens)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"projectPath": projectPath,
		"maxTokens":   maxTokens,
		"content":     summary,
	})
}

// handleUpsertKnowledge serves POST /api/brain/knowledge: it stores one knowledge entry (a lesson, a
// convention, a glossary term) sent as JSON, without a file. Required: projectPath, kind, topic, content.
func (s *Server) handleUpsertKnowledge(w http.ResponseWriter, r *http.Request) {
	if s.brain == nil {
		writeJSONError(w, "brain service not configured", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2*domain.MaxIngestBytes+4096)
	var k domain.ProjectKnowledge
	if err := json.NewDecoder(r.Body).Decode(&k); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeJSONError(w, "content is too large", http.StatusRequestEntityTooLarge)
			return
		}
		writeJSONError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	switch {
	case strings.TrimSpace(k.ProjectPath) == "" || strings.TrimSpace(k.Topic) == "" || strings.TrimSpace(k.Content) == "":
		writeJSONError(w, "projectPath, topic and content are required", http.StatusBadRequest)
		return
	case !k.Kind.IsValid():
		writeJSONError(w, "kind must be one of architecture, convention, file_map, learning, glossary, task_summary", http.StatusBadRequest)
		return
	case len(k.Content) > domain.MaxIngestBytes:
		writeJSONError(w, "content is too large", http.StatusRequestEntityTooLarge)
		return
	}
	stored, err := s.brain.IngestKnowledge(r.Context(), k)
	if err != nil {
		log.Printf("httpapi: upsert knowledge %q: %v", k.Topic, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, stored)
}
