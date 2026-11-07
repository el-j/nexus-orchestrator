package httpapi_client_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/outbound/httpapi_client"
	"nexus-orchestrator/internal/core/domain"
)

func TestClient_GetRuntimeConfig_OK(t *testing.T) {
	want := domain.RuntimeConfig{QueueCap: 50}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/config" {
			http.Error(w, "wrong route", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"queueCap": want.QueueCap}) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	got, err := c.GetRuntimeConfig(context.Background())
	if err != nil {
		t.Fatalf("GetRuntimeConfig: %v", err)
	}
	if got.QueueCap != want.QueueCap {
		t.Errorf("expected QueueCap %d, got %d", want.QueueCap, got.QueueCap)
	}

	// non-200 returns error
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv2.Close()
	if _, err := httpapi_client.NewClient(srv2.URL).GetRuntimeConfig(context.Background()); err == nil {
		t.Error("expected error on non-200, got nil")
	}
}

func TestClient_UpdateRuntimeConfig_OK(t *testing.T) {
	queueCap := 25
	update := domain.RuntimeConfigUpdate{QueueCap: &queueCap}
	wantResp := domain.RuntimeConfig{QueueCap: 25, APIToken: "tok-abc"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/config" {
			http.Error(w, "wrong route", http.StatusInternalServerError)
			return
		}
		assertContentType(t, r)
		var received domain.RuntimeConfigUpdate
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if received.QueueCap == nil || *received.QueueCap != queueCap {
			t.Errorf("expected queueCap %d in body", queueCap)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{ //nolint:errcheck
			"queueCap": wantResp.QueueCap,
			"apiToken": wantResp.APIToken,
		})
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	got, err := c.UpdateRuntimeConfig(context.Background(), update)
	if err != nil {
		t.Fatalf("UpdateRuntimeConfig: %v", err)
	}
	if got.QueueCap != wantResp.QueueCap {
		t.Errorf("expected QueueCap %d, got %d", wantResp.QueueCap, got.QueueCap)
	}
	if got.APIToken != wantResp.APIToken {
		t.Errorf("expected APIToken %q, got %q", wantResp.APIToken, got.APIToken)
	}

	// non-200 returns error
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv2.Close()
	if _, err := httpapi_client.NewClient(srv2.URL).UpdateRuntimeConfig(context.Background(), update); err == nil {
		t.Error("expected error on non-200, got nil")
	}
}

func TestClient_GetDiscoveredPlanFiles(t *testing.T) {
	want := []domain.DiscoveredPlanFile{{ID: "p1", Path: "plan.md", Kind: domain.PlanFileKindMarkdown}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/plans/discovered" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("projectPath") != "/tmp/test" {
			http.Error(w, "missing projectPath", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want) //nolint:errcheck
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	got, err := c.GetDiscoveredPlanFiles(context.Background(), "/tmp/test")
	if err != nil {
		t.Fatalf("GetDiscoveredPlanFiles: %v", err)
	}
	if len(got) != 1 || got[0].Path != "plan.md" {
		t.Errorf("unexpected plan files: %+v", got)
	}

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv2.Close()
	if _, err := httpapi_client.NewClient(srv2.URL).GetDiscoveredPlanFiles(context.Background(), "/tmp/test"); err == nil {
		t.Error("expected error on non-200, got nil")
	}
}

func TestClient_GetUsage(t *testing.T) {
	var gotPath, gotSince string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotSince = r.URL.Path, r.URL.Query().Get("since")
		fmt.Fprint(w, `{"since":"x","totals":{},"usage":[{"provider":"Anthropic","model":"opus","calls":2,"inputTokens":300,"outputTokens":120}]}`)
	}))
	defer srv.Close()
	c := httpapi_client.NewClient(srv.URL)
	since := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	got, err := c.GetUsage(context.Background(), since)
	if err != nil || len(got) != 1 || got[0].Provider != "Anthropic" || got[0].InputTokens != 300 {
		t.Fatalf("%+v %v", got, err)
	}
	if gotPath != "/api/usage" || gotSince != "2026-10-16T12:00:00Z" {
		t.Errorf("request: %s since=%s", gotPath, gotSince)
	}
	if _, err := c.GetUsage(context.Background(), time.Time{}); err != nil || gotSince != "all" {
		t.Errorf("zero time asks for everything: since=%q err=%v", gotSince, err)
	}
}

func TestClient_PreviewRouting(t *testing.T) {
	var gotPath, gotProject, gotRole string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotProject, gotRole = r.URL.Path, r.URL.Query().Get("projectPath"), r.URL.Query().Get("role")
		fmt.Fprint(w, `{"role":"coder","quotaThreshold":0.9,"order":[{"position":1,"provider":"Ollama","model":"qwen","rung":0}]}`)
	}))
	defer srv.Close()
	c := httpapi_client.NewClient(srv.URL)
	got, err := c.PreviewRouting(context.Background(), "/my project", "coder")
	if err != nil || len(got.Order) != 1 || got.Order[0].Provider != "Ollama" || got.QuotaThreshold != 0.9 {
		t.Fatalf("%+v %v", got, err)
	}
	if gotPath != "/api/routing" || gotProject != "/my project" || gotRole != "coder" {
		t.Errorf("request: %s project=%q role=%q", gotPath, gotProject, gotRole)
	}
	if _, err := c.PreviewRouting(context.Background(), "/p", ""); err != nil || gotRole != "" {
		t.Errorf("no role: %v role=%q", err, gotRole)
	}
}

func TestClient_GetRecentActivities_And_GetTimeline(t *testing.T) {
	var lastURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"act-1","agentName":"claude","projectPath":"/proj","activityType":"message"}]`)
	}))
	defer srv.Close()

	c := httpapi_client.NewClient(srv.URL)
	since := time.Date(2026, 10, 19, 8, 0, 0, 0, time.UTC)
	acts, err := c.GetRecentActivities(context.Background(), domain.ActivityFilter{
		AgentName:   "claude",
		ProjectPath: "/proj",
		Type:        domain.ActivityTypeMessage,
		Since:       since,
		Limit:       25,
	})
	if err != nil || len(acts) != 1 || acts[0].ID != "act-1" {
		t.Fatalf("GetRecentActivities: %+v %v", acts, err)
	}
	for _, want := range []string{"/api/activities?", "agent=claude", "project=%2Fproj", "type=message", "since=2026-10-19T08%3A00%3A00Z", "limit=25"} {
		if !strings.Contains(lastURL, want) {
			t.Errorf("GetRecentActivities URL %q missing %q", lastURL, want)
		}
	}

	tl, err := c.GetTimeline(context.Background(), since, 50)
	if err != nil || len(tl) != 1 || tl[0].ID != "act-1" {
		t.Fatalf("GetTimeline: %+v %v", tl, err)
	}
	for _, want := range []string{"/api/activities/timeline?", "since=2026-10-19T08%3A00%3A00Z", "limit=50"} {
		if !strings.Contains(lastURL, want) {
			t.Errorf("GetTimeline URL %q missing %q", lastURL, want)
		}
	}
}
