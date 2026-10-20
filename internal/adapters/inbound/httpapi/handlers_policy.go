package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// policyReply is the answer of GET /api/policy: the policy and whether the project has a policy file.
type policyReply struct {
	Found  bool                 `json:"found"`
	Policy domain.ProjectPolicy `json:"policy"`
}

// handleGetPolicy serves GET /api/policy?projectPath=: the project's .nexus/policy.json.
func (s *Server) handleGetPolicy(w http.ResponseWriter, r *http.Request) {
	pm, ok := s.orch.(ports.PolicyManager)
	if !ok {
		writeJSONError(w, "policy management not available", http.StatusServiceUnavailable)
		return
	}
	project := strings.TrimSpace(r.URL.Query().Get("projectPath"))
	if project == "" {
		writeJSONError(w, "projectPath is required", http.StatusBadRequest)
		return
	}
	pol, found, err := pm.GetProjectPolicy(r.Context(), project)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, policyReply{Found: found, Policy: pol})
}

// handlePutPolicy serves PUT /api/policy?projectPath=: it validates the body as a policy and writes it
// as the project's .nexus/policy.json. Unknown fields and invalid values are refused.
func (s *Server) handlePutPolicy(w http.ResponseWriter, r *http.Request) {
	pm, ok := s.orch.(ports.PolicyManager)
	if !ok {
		writeJSONError(w, "policy management not available", http.StatusServiceUnavailable)
		return
	}
	project := strings.TrimSpace(r.URL.Query().Get("projectPath"))
	if project == "" {
		writeJSONError(w, "projectPath is required", http.StatusBadRequest)
		return
	}
	var pol domain.ProjectPolicy
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&pol); err != nil {
		writeJSONError(w, "invalid policy: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := pm.SetProjectPolicy(r.Context(), project, pol); err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, policyReply{Found: true, Policy: pol})
}
