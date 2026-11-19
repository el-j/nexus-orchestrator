package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/core/domain"
)

type usageOrchestrator struct {
	mockOrchestrator
	sums  []domain.UsageSummary
	err   error
	since time.Time
}

func (u *usageOrchestrator) GetUsage(_ context.Context, since time.Time) ([]domain.UsageSummary, error) {
	u.since = since
	return u.sums, u.err
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
