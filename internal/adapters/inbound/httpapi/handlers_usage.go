package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// handleUsage serves GET /api/usage: tokens and cost per provider, model and role from the usage
// ledger. Query: since (a duration like 24h or 7d, an RFC 3339 time, or all; default 24h) and
// optional projectPath to filter by project repository path.
func (s *Server) handleUsage(w http.ResponseWriter, r *http.Request) {
	q, ok := s.orch.(ports.UsageQuerier)
	if !ok {
		writeJSONError(w, "usage ledger not available", http.StatusServiceUnavailable)
		return
	}
	since, err := domain.ParseSince(r.URL.Query().Get("since"), time.Now())
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	projectPath := r.URL.Query().Get("projectPath")
	var sums []domain.UsageSummary
	if projectPath != "" {
		sums, err = q.GetUsageForProject(r.Context(), since, projectPath)
	} else {
		sums, err = q.GetUsage(r.Context(), since)
	}
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if sums == nil {
		sums = []domain.UsageSummary{}
	}
	res := map[string]any{"since": since, "totals": domain.UsageTotals(sums), "usage": sums}
	if projectPath != "" {
		res["projectPath"] = projectPath
	}
	writeJSON(w, http.StatusOK, res)
}

// usageReportPayload defines the JSON body for reporting usage via POST /api/usage.
type usageReportPayload struct {
	ProjectPath  string  `json:"projectPath"`
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	Role         string  `json:"role"`
	TaskID       string  `json:"taskId"`
	InputTokens  int     `json:"inputTokens"`
	OutputTokens int     `json:"outputTokens"`
	CachedTokens int     `json:"cachedTokens"`
	CostUSD      float64 `json:"costUSD"`
}

// handleReportUsage serves POST /api/usage: allows external agents to record model call token usage.
func (s *Server) handleReportUsage(w http.ResponseWriter, r *http.Request) {
	q, ok := s.orch.(ports.UsageRecorder)
	if !ok {
		writeJSONError(w, "usage ledger not available", http.StatusServiceUnavailable)
		return
	}
	var p usageReportPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if p.Provider == "" {
		writeJSONError(w, "provider is required", http.StatusBadRequest)
		return
	}
	rec := domain.UsageRecord{
		At:          time.Now(),
		TaskID:      p.TaskID,
		ProjectPath: p.ProjectPath,
		Provider:    p.Provider,
		Role:        p.Role,
		Usage: domain.Usage{
			Model:        p.Model,
			InputTokens:  p.InputTokens,
			OutputTokens: p.OutputTokens,
			CachedTokens: p.CachedTokens,
			CostUSD:      p.CostUSD,
		},
	}
	if err := q.RecordUsage(r.Context(), rec); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]bool{"ok": true})
}
