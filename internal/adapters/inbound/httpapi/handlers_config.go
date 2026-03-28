package httpapi

import (
	"encoding/json"
	"net/http"

	"nexus-orchestrator/internal/core/domain"
)

type runtimeConfigResponse struct {
	QueueCap int `json:"queueCap"`

	APITokenEnabled bool `json:"apiTokenEnabled"`
	MCPTokenEnabled bool `json:"mcpTokenEnabled"`

	// Plaintext tokens are returned only in PUT responses when explicitly set or rotated.
	APIToken string `json:"apiToken,omitempty"`
	MCPToken string `json:"mcpToken,omitempty"`
}

type putRuntimeConfigRequest struct {
	QueueCap *int `json:"queueCap,omitempty"`

	APIToken *string `json:"apiToken,omitempty"`
	MCPToken *string `json:"mcpToken,omitempty"`

	RotateAPIToken bool `json:"rotateApiToken,omitempty"`
	RotateMCPToken bool `json:"rotateMcpToken,omitempty"`
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	if s.orch == nil {
		writeJSONError(w, "orchestrator unavailable", http.StatusServiceUnavailable)
		return
	}
	cfg, err := s.orch.GetRuntimeConfig(r.Context())
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := runtimeConfigResponse{
		QueueCap:        cfg.QueueCap,
		APITokenEnabled: cfg.APIToken != "",
		MCPTokenEnabled: cfg.MCPToken != "",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	if s.orch == nil {
		writeJSONError(w, "orchestrator unavailable", http.StatusServiceUnavailable)
		return
	}

	var req putRuntimeConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "invalid json", http.StatusBadRequest)
		return
	}
	update := domain.RuntimeConfigUpdate{
		QueueCap:       req.QueueCap,
		APIToken:       req.APIToken,
		MCPToken:       req.MCPToken,
		RotateAPIToken: req.RotateAPIToken,
		RotateMCPToken: req.RotateMCPToken,
	}

	cfg, err := s.orch.UpdateRuntimeConfig(r.Context(), update)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp := runtimeConfigResponse{
		QueueCap:        cfg.QueueCap,
		APITokenEnabled: cfg.APIToken != "",
		MCPTokenEnabled: cfg.MCPToken != "",
	}
	// Return plaintext tokens only when explicitly set/rotated by this request.
	if req.RotateAPIToken || req.APIToken != nil {
		resp.APIToken = cfg.APIToken
	}
	if req.RotateMCPToken || req.MCPToken != nil {
		resp.MCPToken = cfg.MCPToken
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
