package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/core/domain"
)

// mockActivityService implements the activityQuerier interface that the
// activity handlers depend on. Even though activityQuerier is unexported,
// Go allows passing a structurally-compatible concrete type to
// Server.WithActivityService from external packages.
type mockActivityService struct {
	recentActivities    []domain.AIActivity
	recentActivitiesErr error
	timelineActivities  []domain.AIActivity
	timelineErr         error
}

func (m *mockActivityService) GetRecentActivities(_ context.Context, _ domain.ActivityFilter) ([]domain.AIActivity, error) {
	return m.recentActivities, m.recentActivitiesErr
}

func (m *mockActivityService) GetTimeline(_ context.Context, _ time.Time, _ int) ([]domain.AIActivity, error) {
	return m.timelineActivities, m.timelineErr
}

// serveWithActivity creates a Server wired with the given activity service and
// serves the request into the recorder.
func serveWithActivity(svc *mockActivityService, req *http.Request, rec *httptest.ResponseRecorder) {
	orch := &mockOrchestrator{}
	srv := httpapi.NewServer(orch, nil, nil)
	srv.WithActivityService(svc)
	srv.Handler().ServeHTTP(rec, req)
}

// ---------------------------------------------------------------------------
// GET /api/activities
// ---------------------------------------------------------------------------

func TestHandleListActivities_OK(t *testing.T) {
	activities := []domain.AIActivity{
		{
			ID:           "a1",
			AgentName:    "claude",
			ActivityType: domain.ActivityTypeMessage,
			Summary:      "wrote some code",
			ProjectPath:  "/tmp/proj",
			Timestamp:    time.Now(),
		},
		{
			ID:           "a2",
			AgentName:    "claude",
			ActivityType: domain.ActivityTypeFileEdit,
			Summary:      "edited main.go",
			ProjectPath:  "/tmp/proj",
			Timestamp:    time.Now(),
		},
	}
	svc := &mockActivityService{recentActivities: activities}
	req := httptest.NewRequest(http.MethodGet, "/api/activities?project=/tmp/proj", nil)
	rec := httptest.NewRecorder()

	serveWithActivity(svc, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var got []domain.AIActivity
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 activities, got %d", len(got))
	}
	if got[0].ID != "a1" {
		t.Errorf("expected first activity ID %q, got %q", "a1", got[0].ID)
	}
}

func TestHandleListActivities_Empty(t *testing.T) {
	svc := &mockActivityService{recentActivities: nil}
	req := httptest.NewRequest(http.MethodGet, "/api/activities", nil)
	rec := httptest.NewRecorder()

	serveWithActivity(svc, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	// Nil should be normalised to empty JSON array
	body := rec.Body.String()
	if body == "null\n" || body == "null" {
		t.Errorf("expected JSON array, got %q", body)
	}
}

func TestHandleListActivities_NilService(t *testing.T) {
	// When activitySvc is nil the handler must return 503
	orch := &mockOrchestrator{}
	srv := httpapi.NewServer(orch, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/activities", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestHandleListActivities_WithFilters(t *testing.T) {
	activities := []domain.AIActivity{
		{ID: "a3", AgentName: "copilot", ActivityType: domain.ActivityTypeToolUse, Summary: "ran tool"},
	}
	svc := &mockActivityService{recentActivities: activities}
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/activities?agent=copilot&type=tool_use&limit=10",
		nil,
	)
	rec := httptest.NewRecorder()

	serveWithActivity(svc, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var got []domain.AIActivity
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(got))
	}
}

// ---------------------------------------------------------------------------
// GET /api/activities/timeline
// ---------------------------------------------------------------------------

func TestHandleActivityTimeline_OK(t *testing.T) {
	activities := []domain.AIActivity{
		{
			ID:           "t1",
			AgentName:    "claude",
			ActivityType: domain.ActivityTypeGeneration,
			Summary:      "generated text",
			Timestamp:    time.Now().Add(-1 * time.Hour),
		},
	}
	svc := &mockActivityService{timelineActivities: activities}
	req := httptest.NewRequest(http.MethodGet, "/api/activities/timeline", nil)
	rec := httptest.NewRecorder()

	serveWithActivity(svc, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	var got []domain.AIActivity
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 timeline activity, got %d", len(got))
	}
	if got[0].ID != "t1" {
		t.Errorf("expected ID %q, got %q", "t1", got[0].ID)
	}
}

func TestHandleActivityTimeline_WithSinceAndLimit(t *testing.T) {
	svc := &mockActivityService{timelineActivities: []domain.AIActivity{}}
	since := time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/activities/timeline?since="+since+"&limit=25",
		nil,
	)
	rec := httptest.NewRecorder()

	serveWithActivity(svc, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleActivityTimeline_NilService(t *testing.T) {
	orch := &mockOrchestrator{}
	srv := httpapi.NewServer(orch, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/activities/timeline", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestHandleActivityTimeline_Empty(t *testing.T) {
	svc := &mockActivityService{timelineActivities: nil}
	req := httptest.NewRequest(http.MethodGet, "/api/activities/timeline", nil)
	rec := httptest.NewRecorder()

	serveWithActivity(svc, req, rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	// Nil should be normalised to an empty JSON array
	body := rec.Body.String()
	if body == "null\n" || body == "null" {
		t.Errorf("expected JSON array, got %q", body)
	}
}
