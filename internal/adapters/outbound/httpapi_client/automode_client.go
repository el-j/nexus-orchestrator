package httpapi_client

import (
	"context"
	"net/http"
	"net/url"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// Compile-time check that Client satisfies ports.AutoModeManager.
var _ ports.AutoModeManager = (*Client)(nil)

// StartAutoMode asks the daemon to launch a Project Auto-Mode run.
func (r *Client) StartAutoMode(ctx context.Context, req domain.AutoModeRequest) (domain.AutoModeRun, error) {
	var out domain.AutoModeRun
	err := r.send(ctx, call{op: "start auto-mode", method: http.MethodPost, path: "/api/auto/start", body: req}, &out)
	return out, err
}

// GetAutoModeRun fetches the live snapshot of one Auto-Mode run.
func (r *Client) GetAutoModeRun(id string) (domain.AutoModeRun, error) {
	var out domain.AutoModeRun
	err := r.send(context.Background(), call{op: "get auto-mode run", method: http.MethodGet, path: "/api/auto/" + url.PathEscape(id)}, &out)
	return out, err
}

// ListAutoModeRuns lists Auto-Mode runs, optionally filtered by projectPath.
func (r *Client) ListAutoModeRuns(projectPath string) []domain.AutoModeRun {
	path := "/api/auto"
	if projectPath != "" {
		path += "?projectPath=" + url.QueryEscape(projectPath)
	}
	var out []domain.AutoModeRun
	if err := r.send(context.Background(), call{op: "list auto-mode runs", method: http.MethodGet, path: path}, &out); err != nil {
		return []domain.AutoModeRun{}
	}
	return out
}

// StopAutoMode asks the daemon to cancel an active Auto-Mode run.
func (r *Client) StopAutoMode(id string) error {
	return r.send(context.Background(), call{op: "stop auto-mode", method: http.MethodPost, path: "/api/auto/" + url.PathEscape(id) + "/stop"}, nil)
}
