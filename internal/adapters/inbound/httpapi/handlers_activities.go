package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// activityQuerier is a local interface so handlers_activities.go depends on
// behaviour only; it is satisfied by *services.ActivityService.
type activityQuerier interface {
	GetRecentActivities(ctx context.Context, f domain.ActivityFilter) ([]domain.AIActivity, error)
	GetTimeline(ctx context.Context, since time.Time, limit int) ([]domain.AIActivity, error)
}

// handleListActivities serves GET /api/activities
// Query params: since (RFC3339), agent, project, type, limit (default 50)
func (s *Server) handleListActivities(w http.ResponseWriter, r *http.Request) {
	if s.activitySvc == nil {
		http.Error(w, `{"error":"activity service not available"}`, http.StatusServiceUnavailable)
		return
	}
	q := r.URL.Query()
	f := domain.ActivityFilter{
		AgentName:   q.Get("agent"),
		ProjectPath: q.Get("project"),
	}
	if t := q.Get("type"); t != "" {
		f.Type = domain.ActivityType(t)
	}
	if sinceStr := q.Get("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			f.Since = t
		}
	}
	if limitStr := q.Get("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			f.Limit = n
		}
	}
	if f.Limit == 0 {
		f.Limit = 50
	}

	activities, err := s.activitySvc.GetRecentActivities(r.Context(), f)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	if activities == nil {
		activities = []domain.AIActivity{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(activities)
}

// handleActivityTimeline serves GET /api/activities/timeline
// Query params: since (RFC3339), limit (default 100)
func (s *Server) handleActivityTimeline(w http.ResponseWriter, r *http.Request) {
	if s.activitySvc == nil {
		http.Error(w, `{"error":"activity service not available"}`, http.StatusServiceUnavailable)
		return
	}
	q := r.URL.Query()
	since := time.Now().Add(-24 * time.Hour) // last 24h default
	if sinceStr := q.Get("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = t
		}
	}
	limit := 100
	if limitStr := q.Get("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	activities, err := s.activitySvc.GetTimeline(r.Context(), since, limit)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	if activities == nil {
		activities = []domain.AIActivity{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(activities)
}
