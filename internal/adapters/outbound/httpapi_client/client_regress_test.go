package httpapi_client_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/outbound/httpapi_client"
	"nexus-orchestrator/internal/core/domain"
)

func jsonErrorServer(status int, msg string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
	}))
}

// A rejected claim (409/500...) must be an error. Previously the JSON error body
// was decoded into an empty Task and returned as a successful claim.
func TestClaimTask_NonSuccessStatusIsAnError(t *testing.T) {
	for _, status := range []int{http.StatusConflict, http.StatusBadRequest, http.StatusInternalServerError} {
		srv := jsonErrorServer(status, "task t-1 is not QUEUED")
		c := httpapi_client.NewClient(srv.URL)
		got, err := c.ClaimTask(context.Background(), "t-1", "s-1")
		srv.Close()
		if err == nil {
			t.Fatalf("status %d: ClaimTask returned success with %+v", status, got)
		}
		if !strings.Contains(err.Error(), "task t-1 is not QUEUED") {
			t.Errorf("status %d: the server's message should reach the caller, got %q", status, err)
		}
	}
}

func TestPurgeDisconnectedSessions_NonSuccessStatusIsAnError(t *testing.T) {
	srv := jsonErrorServer(http.StatusInternalServerError, "db down")
	defer srv.Close()
	if n, err := httpapi_client.NewClient(srv.URL).PurgeDisconnectedSessions(context.Background()); err == nil {
		t.Fatalf("a failed purge was reported as success (%d deleted)", n)
	}
}

func TestTriggerScan_NoContentMeansNoProviders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	got, err := httpapi_client.NewClient(srv.URL).TriggerScan(context.Background())
	if err != nil {
		t.Fatalf("204 must not fail decoding an empty body: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("want an empty non-nil slice, got %#v", got)
	}
}

// Project paths routinely contain spaces, '&', '#', '+' and '%'.
func TestProjectScopedQueries_EscapeThePath(t *testing.T) {
	const path = "/Users/me/My Project & Co/#1+2%25"
	var gotQuery map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = map[string]string{"projectPath": r.URL.Query().Get("projectPath"), "raw": r.URL.RawQuery}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()
	c := httpapi_client.NewClient(srv.URL)

	if _, err := c.GetQueueForProject(path); err != nil {
		t.Fatal(err)
	}
	if gotQuery["projectPath"] != path {
		t.Errorf("GetQueueForProject: server saw %q (raw %q), want %q", gotQuery["projectPath"], gotQuery["raw"], path)
	}
	if _, err := c.GetTasksForProject(path); err != nil {
		t.Fatal(err)
	}
	if gotQuery["projectPath"] != path {
		t.Errorf("GetTasksForProject: server saw %q (raw %q), want %q", gotQuery["projectPath"], gotQuery["raw"], path)
	}
}

func TestNotFoundIsMappedToErrNotFoundWhereCallersRely_OnIt(t *testing.T) {
	srv := jsonErrorServer(http.StatusNotFound, "no such thing")
	defer srv.Close()
	c := httpapi_client.NewClient(srv.URL)
	ctx := context.Background()

	checks := map[string]func() error{
		"GetTask":              func() error { _, err := c.GetTask("x"); return err },
		"CancelTask":           func() error { return c.CancelTask("x") },
		"PromoteTask":          func() error { _, err := c.PromoteTask("x"); return err },
		"UpdateTask":           func() error { _, err := c.UpdateTask("x", domain.Task{}); return err },
		"PromoteProvider":      func() error { return c.PromoteProvider(ctx, "x") },
		"ClaimTask":            func() error { _, err := c.ClaimTask(ctx, "x", "s"); return err },
		"HeartbeatTask":        func() error { return c.HeartbeatTask(ctx, "x", "s") },
		"UpdateTaskStatus":     func() error { _, err := c.UpdateTaskStatus(ctx, "x", "s", domain.StatusCompleted, ""); return err },
		"DelegateToNexus":      func() error { _, err := c.DelegateToNexus(ctx, "x"); return err },
		"RemoveProvider":       func() error { return c.RemoveProvider("x") },
		"RemoveProviderConfig": func() error { return c.RemoveProviderConfig(ctx, "x") },
		"DeregisterAISession":  func() error { return c.DeregisterAISession(ctx, "x") },
	}
	for name, call := range checks {
		if err := call(); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("%s: a 404 must map to domain.ErrNotFound, got %v", name, err)
		}
	}
}
