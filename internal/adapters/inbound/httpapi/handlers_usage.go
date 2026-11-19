package httpapi

import (
	"net/http"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// handleUsage serves GET /api/usage: tokens and cost per provider, model and role from the usage
// ledger. Query: since (a duration like 24h or 7d, an RFC 3339 time, or all; default 24h).
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
	sums, err := q.GetUsage(r.Context(), since)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if sums == nil {
		sums = []domain.UsageSummary{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"since": since, "totals": domain.UsageTotals(sums), "usage": sums})
}
