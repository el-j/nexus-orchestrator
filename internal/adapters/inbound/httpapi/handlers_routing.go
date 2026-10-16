package httpapi

import (
	"net/http"
	"strings"

	"nexus-orchestrator/internal/core/ports"
)

// handleRouting serves GET /api/routing: how routing would order the providers for a task of a role
// in a project right now (the project's ladder, cooldowns and quota readings applied).
// Query: projectPath (required), role (optional).
func (s *Server) handleRouting(w http.ResponseWriter, r *http.Request) {
	p, ok := s.orch.(ports.RoutingPreviewer)
	if !ok {
		writeJSONError(w, "routing preview not available", http.StatusServiceUnavailable)
		return
	}
	project := strings.TrimSpace(r.URL.Query().Get("projectPath"))
	if project == "" {
		writeJSONError(w, "projectPath is required", http.StatusBadRequest)
		return
	}
	preview, err := p.PreviewRouting(r.Context(), project, strings.TrimSpace(r.URL.Query().Get("role")))
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}
