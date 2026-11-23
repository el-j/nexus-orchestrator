package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/core/domain"
)

type usageOrchestrator struct {
	mockOrchestrator
	sums    []domain.UsageSummary
	err     error
	since   time.Time
	project string
	rec     domain.UsageRecord
}

func (u *usageOrchestrator) GetUsage(_ context.Context, since time.Time) ([]domain.UsageSummary, error) {
	u.since = since
	return u.sums, u.err
}

func (u *usageOrchestrator) GetUsageForProject(_ context.Context, since time.Time, projectPath string) ([]domain.UsageSummary, error) {
	u.since = since
	u.project = projectPath
	return u.sums, u.err
}

func (u *usageOrchestrator) RecordUsage(_ context.Context, r domain.UsageRecord) error {
	u.rec = r
	return u.err
}

func getJSON(t *testing.T, url string) (int, map[string]any) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestUsageEndpoint(t *testing.T) {
	o := &usageOrchestrator{sums: []domain.UsageSummary{
		{Provider: "Anthropic", Model: "opus", Calls: 2, InputTokens: 300, OutputTokens: 120, CostUSD: 0.75},
		{Provider: "Ollama", Model: "qwen", Calls: 1, InputTokens: 50},
	}}
	ts := httptest.NewServer(httpapi.NewServer(o, nil, nil).Handler())
	defer ts.Close()

	code, body := getJSON(t, ts.URL+"/api/usage?since=7d")
	if code != http.StatusOK {
		t.Fatalf("status %d: %v", code, body)
	}
	totals := body["totals"].(map[string]any)
	if totals["calls"] != float64(3) || totals["inputTokens"] != float64(350) {
		t.Errorf("totals: %v", totals)
	}
	if usage := body["usage"].([]any); len(usage) != 2 {
		t.Errorf("usage: %v", usage)
	}
	if d := time.Since(o.since); d < 7*24*time.Hour-time.Minute || d > 7*24*time.Hour+time.Minute {
		t.Errorf("since = %v", o.since)
	}
	if _, _ = getJSON(t, ts.URL+"/api/usage"); time.Since(o.since) < 23*time.Hour || time.Since(o.since) > 25*time.Hour {
		t.Errorf("the default window is 24h: %v", o.since)
	}
	if _, _ = getJSON(t, ts.URL+"/api/usage?since=all"); !o.since.IsZero() {
		t.Errorf("all = the zero time: %v", o.since)
	}
	code, body = getJSON(t, ts.URL+"/api/usage?projectPath=/my/project")
	if code != http.StatusOK || body["projectPath"] != "/my/project" || o.project != "/my/project" {
		t.Errorf("projectPath parameter: %d %v, o.project=%q", code, body, o.project)
	}
}

func TestReportUsageEndpoint(t *testing.T) {
	o := &usageOrchestrator{}
	ts := httptest.NewServer(httpapi.NewServer(o, nil, nil).Handler())
	defer ts.Close()

	// 1. Valid POST /api/usage
	body := `{"projectPath":"/my/project","provider":"Antigravity","model":"flash","role":"coder","inputTokens":10,"outputTokens":20,"cachedTokens":5,"costUSD":0.001}`
	resp, err := http.Post(ts.URL+"/api/usage", "application/json", strings.NewReader(body))
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %v %v", resp, err)
	}
	_ = resp.Body.Close()
	if o.rec.Provider != "Antigravity" || o.rec.ProjectPath != "/my/project" || o.rec.InputTokens != 10 {
		t.Errorf("recorded record: %+v", o.rec)
	}

	// 2. Missing provider
	resp, _ = http.Post(ts.URL+"/api/usage", "application/json", strings.NewReader(`{"inputTokens":10}`))
	if resp != nil {
		_ = resp.Body.Close()
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing provider, got %d", resp.StatusCode)
	}

	// 3. Bad JSON
	resp, _ = http.Post(ts.URL+"/api/usage", "application/json", strings.NewReader(`not-json`))
	if resp != nil {
		_ = resp.Body.Close()
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for bad JSON, got %d", resp.StatusCode)
	}

	// 4. Ledger unavailable
	plain := httptest.NewServer(httpapi.NewServer(&mockOrchestrator{}, nil, nil).Handler())
	defer plain.Close()
	resp, _ = http.Post(plain.URL+"/api/usage", "application/json", strings.NewReader(body))
	if resp != nil {
		_ = resp.Body.Close()
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when ledger unavailable, got %d", resp.StatusCode)
	}

	// 5. Orchestrator error
	failing := httptest.NewServer(httpapi.NewServer(&usageOrchestrator{err: errors.New("write failed")}, nil, nil).Handler())
	defer failing.Close()
	resp, _ = http.Post(failing.URL+"/api/usage", "application/json", strings.NewReader(body))
	if resp != nil {
		_ = resp.Body.Close()
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 when orchestrator fails, got %d", resp.StatusCode)
	}
}

func TestUsageEndpoint_Errors(t *testing.T) {
	bad := httptest.NewServer(httpapi.NewServer(&usageOrchestrator{}, nil, nil).Handler())
	defer bad.Close()
	if code, _ := getJSON(t, bad.URL+"/api/usage?since=yesterday"); code != http.StatusBadRequest {
		t.Errorf("bad window: %d", code)
	}
	if code, body := getJSON(t, bad.URL+"/api/usage"); code != http.StatusOK || body["usage"] == nil {
		t.Errorf("an empty ledger answers an empty list, not null: %d %v", code, body)
	}
	failing := httptest.NewServer(httpapi.NewServer(&usageOrchestrator{err: errors.New("db: /secret/path")}, nil, nil).Handler())
	defer failing.Close()
	if code, body := getJSON(t, failing.URL+"/api/usage"); code != http.StatusInternalServerError || body["error"] == nil {
		t.Errorf("ledger failure: %d %v", code, body)
	} else if msg, _ := body["error"].(string); msg != "internal server error" {
		t.Errorf("the error must not leak details: %q", msg)
	}
	plain := httptest.NewServer(httpapi.NewServer(&mockOrchestrator{}, nil, nil).Handler())
	defer plain.Close()
	if code, _ := getJSON(t, plain.URL+"/api/usage"); code != http.StatusServiceUnavailable {
		t.Errorf("no ledger: %d", code)
	}
}
