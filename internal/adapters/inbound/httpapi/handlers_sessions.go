package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"nexus-orchestrator/internal/core/domain"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleRegisterAISession(w http.ResponseWriter, r *http.Request) {
	var session domain.AISession
	if err := json.NewDecoder(r.Body).Decode(&session); err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if session.AgentName == "" {
		writeJSONError(w, "agentName is required", http.StatusBadRequest)
		return
	}
	if session.Source != domain.SessionSourceMCP && session.Source != domain.SessionSourceVSCode && session.Source != domain.SessionSourceHTTP {
		writeJSONError(w, "source must be one of: mcp, vscode, http", http.StatusBadRequest)
		return
	}
	created, err := s.orch.RegisterAISession(r.Context(), session)
	if err != nil {
		log.Printf("httpapi: register ai session: %v", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (s *Server) handleListAISessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.orch.ListAISessions(r.Context())
	if err != nil {
		log.Printf("httpapi: list ai sessions: %v", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if sessions == nil {
		sessions = []domain.AISession{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sessions)
}

func (s *Server) handleDeregisterAISession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.orch.DeregisterAISession(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "ai session not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: deregister ai session %s: %v", id, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handlePurgeDisconnectedSessions deletes all AI sessions that are disconnected
// and have been inactive for more than 2 hours. Returns {"deleted": N}.
func (s *Server) handlePurgeDisconnectedSessions(w http.ResponseWriter, r *http.Request) {
	n, err := s.orch.PurgeDisconnectedSessions(r.Context())
	if err != nil {
		log.Printf("httpapi: purge disconnected sessions: %v", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int{"deleted": n})
}

func (s *Server) handleHeartbeatAISession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.orch.HeartbeatAISession(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "ai session not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: heartbeat ai session %s: %v", id, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTerminateAISession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Force bool `json:"force"`
	}
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONError(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
	}
	if err := s.orch.TerminateAISession(r.Context(), id, body.Force); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "ai session not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: terminate ai session %s: %v", id, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetDiscoveredAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := s.orch.GetDiscoveredAgents(r.Context())
	if err != nil {
		log.Printf("httpapi: get discovered agents: %v", err)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]domain.DiscoveredAgent{})
		return
	}
	if agents == nil {
		agents = []domain.DiscoveredAgent{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(agents)
}

// handleGetDiscoveredPlanFiles handles GET /api/plans/discovered?projectPath=<path>
func (s *Server) handleGetDiscoveredPlanFiles(w http.ResponseWriter, r *http.Request) {
	projectPath := r.URL.Query().Get("projectPath")
	files, err := s.orch.GetDiscoveredPlanFiles(r.Context(), projectPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if files == nil {
		files = []domain.DiscoveredPlanFile{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(files)
}

// handleScanPlanFiles handles POST /api/plans/discovered/scan?projectPath=<path>
func (s *Server) handleScanPlanFiles(w http.ResponseWriter, r *http.Request) {
	projectPath := r.URL.Query().Get("projectPath")
	files, err := s.orch.GetDiscoveredPlanFiles(r.Context(), projectPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if files == nil {
		files = []domain.DiscoveredPlanFile{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(files)
}

func (s *Server) handleDelegateToNexus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	instruction, err := s.orch.DelegateToNexus(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "session not found", http.StatusNotFound)
			return
		}
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"instruction": instruction,
		"sessionId":   id,
	})
}
