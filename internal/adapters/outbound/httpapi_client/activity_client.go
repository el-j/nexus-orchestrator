package httpapi_client

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// GetRecentActivities queries the daemon's activity observatory (GET /api/activities).
func (r *Client) GetRecentActivities(ctx context.Context, f domain.ActivityFilter) ([]domain.AIActivity, error) {
	q := url.Values{}
	if f.AgentName != "" {
		q.Set("agent", f.AgentName)
	}
	if f.ProjectPath != "" {
		q.Set("project", f.ProjectPath)
	}
	if f.Type != "" {
		q.Set("type", string(f.Type))
	}
	if !f.Since.IsZero() {
		q.Set("since", f.Since.UTC().Format(time.RFC3339))
	}
	if f.Limit > 0 {
		q.Set("limit", strconv.Itoa(f.Limit))
	}
	path := "/api/activities"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	var out []domain.AIActivity
	err := r.send(ctx, call{op: "get recent activities", method: http.MethodGet, path: path}, &out)
	return out, err
}

// GetTimeline queries the daemon's chronological activity timeline (GET /api/activities/timeline).
func (r *Client) GetTimeline(ctx context.Context, since time.Time, limit int) ([]domain.AIActivity, error) {
	q := url.Values{}
	if !since.IsZero() {
		q.Set("since", since.UTC().Format(time.RFC3339))
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	path := "/api/activities/timeline"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	var out []domain.AIActivity
	err := r.send(ctx, call{op: "get activity timeline", method: http.MethodGet, path: path}, &out)
	return out, err
}
