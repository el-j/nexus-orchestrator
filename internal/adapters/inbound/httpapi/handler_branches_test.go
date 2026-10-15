package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

// nilSliceOrch returns nil slices where a handler promises "[]" in its JSON.
type nilSliceOrch struct{ *failOrch }

func (nilSliceOrch) GetQueue() ([]domain.Task, error)                           { return nil, nil }
func (nilSliceOrch) GetAllTasks() ([]domain.Task, error)                        { return nil, nil }
func (nilSliceOrch) GetBacklog(string) ([]domain.Task, error)                   { return nil, nil }
func (nilSliceOrch) ListAISessions(context.Context) ([]domain.AISession, error) { return nil, nil }
func (nilSliceOrch) GetDiscoveredAgents(context.Context) ([]domain.DiscoveredAgent, error) {
	return nil, nil
}
func (nilSliceOrch) GetDiscoveredPlanFiles(context.Context, string) ([]domain.DiscoveredPlanFile, error) {
	return nil, nil
}

func TestHandlers_AnswerAnEmptyJSONArrayNotNull(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	h := realHandler(nilSliceOrch{newFailOrch()}, nil)
	for _, path := range []string{
		"/api/tasks", "/api/tasks/all", "/api/tasks/backlog", "/api/ai-sessions",
		"/api/ai-sessions/discovered", "/api/plans/discovered",
	} {
		rec := send(h, http.MethodGet, path, "")
		if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
			t.Errorf("%s: %d %q, want 200 []", path, rec.Code, rec.Body.String())
		}
	}
	rec := send(h, http.MethodPost, "/api/plans/discovered/scan", "")
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("scan: %q", rec.Body.String())
	}
}

func TestHandlers_PlanFilesReportAnUnconfiguredSubsystemAs503(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	orch := newFailOrch().failWith("GetDiscoveredPlanFiles", domain.ErrSubsystemNotConfigured)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/plans/discovered"}, {http.MethodPost, "/api/plans/discovered/scan"},
	} {
		if rec := send(realHandler(orch, nil), tc.method, tc.path, ""); rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s %s: %d, want 503", tc.method, tc.path, rec.Code)
		}
	}
}

func TestHandlers_ClaimAndStatusMapDomainRulesToHTTPStatuses(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "")
	for name, tc := range map[string]struct {
		op, method, path, body string
		msg                    string
		want                   int
	}{
		"claim wrong project":   {"ClaimTask", "POST", "/api/tasks/t/claim", `{"sessionId":"s"}`, "session project does not match task project", http.StatusForbidden},
		"claim not queued":      {"ClaimTask", "POST", "/api/tasks/t/claim", `{"sessionId":"s"}`, "task is not QUEUED", http.StatusConflict},
		"claim disconnected":    {"ClaimTask", "POST", "/api/tasks/t/claim", `{"sessionId":"s"}`, "session is disconnected", http.StatusConflict},
		"status foreign owner":  {"UpdateTaskStatus", "PUT", "/api/tasks/t/status", `{"sessionId":"s","status":"COMPLETED"}`, "session does not own task", http.StatusForbidden},
		"status not processing": {"UpdateTaskStatus", "PUT", "/api/tasks/t/status", `{"sessionId":"s","status":"FAILED"}`, "task is not PROCESSING", http.StatusConflict},
		"status invalid target": {"UpdateTaskStatus", "PUT", "/api/tasks/t/status", `{"sessionId":"s","status":"FAILED"}`, "invalid target status", http.StatusConflict},
	} {
		t.Run(name, func(t *testing.T) {
			orch := newFailOrch().failWith(tc.op, errors.New(tc.msg))
			rec := send(realHandler(orch, nil), tc.method, tc.path, tc.body)
			if rec.Code != tc.want || !strings.Contains(rec.Body.String(), tc.msg) {
				t.Errorf("%d %s, want %d with %q", rec.Code, rec.Body.String(), tc.want, tc.msg)
			}
		})
	}
}
