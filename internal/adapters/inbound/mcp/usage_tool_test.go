package mcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/mcp"
	"nexus-orchestrator/internal/core/domain"
)

type usageOrch struct {
	mockOrch
	sums  []domain.UsageSummary
	err   error
	since time.Time
}

func (u *usageOrch) GetUsage(_ context.Context, since time.Time) ([]domain.UsageSummary, error) {
	u.since = since
	return u.sums, u.err
}

func usageServer(t *testing.T, o *usageOrch) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(mcp.NewMcpServer(o, nil))
	t.Cleanup(srv.Close)
	return srv
}

func TestGetUsage_ReturnsTotalsAndGroups(t *testing.T) {
	o := &usageOrch{sums: []domain.UsageSummary{
		{Provider: "Anthropic", Model: "opus", Role: "architect", Calls: 2, InputTokens: 300, OutputTokens: 120, CostUSD: 0.75},
		{Provider: "Ollama", Model: "qwen", Calls: 1, InputTokens: 50, OutputTokens: 5},
	}}
	srv := usageServer(t, o)
	before := time.Now()
	r := callTool(t, srv, 1, "get_usage", map[string]any{"since": "7d"})
	if r.Error != nil {
		t.Fatalf("error: %+v", r.Error)
	}
	var res struct {
		Content []struct{ Text string } `json:"content"`
	}
	if err := json.Unmarshal(r.Result, &res); err != nil || len(res.Content) != 1 {
		t.Fatalf("result: %s %v", r.Result, err)
	}
	var out struct {
		Since  time.Time             `json:"since"`
		Totals domain.UsageSummary   `json:"totals"`
		Usage  []domain.UsageSummary `json:"usage"`
	}
	if err := json.Unmarshal([]byte(res.Content[0].Text), &out); err != nil {
		t.Fatal(err)
	}
	if out.Totals.Calls != 3 || out.Totals.InputTokens != 350 || len(out.Usage) != 2 || out.Usage[0].Role != "architect" {
		t.Errorf("out: %+v", out)
	}
	if d := before.Sub(o.since); d < 7*24*time.Hour-time.Minute || d > 7*24*time.Hour+time.Minute {
		t.Errorf("since = %v (7d back from now expected)", o.since)
	}
}

func TestGetUsage_DefaultsToADayAndRejectsABadWindow(t *testing.T) {
	o := &usageOrch{}
	srv := usageServer(t, o)
	if r := callTool(t, srv, 1, "get_usage", nil); r.Error != nil {
		t.Fatalf("no arguments: %+v", r.Error)
	}
	if time.Since(o.since) < 23*time.Hour || time.Since(o.since) > 25*time.Hour {
		t.Errorf("default window: %v", o.since)
	}
	r := callTool(t, srv, 2, "get_usage", map[string]any{"since": "yesterday"})
	if r.Error == nil || !strings.Contains(r.Error.Message, "invalid since") {
		t.Errorf("bad window: %+v", r.Error)
	}
}

func TestGetUsage_ErrorsAreReported(t *testing.T) {
	srv := usageServer(t, &usageOrch{err: errors.New("ledger down")})
	if r := callTool(t, srv, 1, "get_usage", nil); r.Error == nil && !strings.Contains(string(r.Result), "ledger down") {
		t.Errorf("a ledger error must reach the caller: %s", r.Result)
	}
	// An orchestrator without a ledger says so.
	plain := httptest.NewServer(mcp.NewMcpServer(&mockOrch{}, nil))
	defer plain.Close()
	if r := callTool(t, plain, 2, "get_usage", nil); r.Error == nil && !strings.Contains(string(r.Result), "not available") {
		t.Errorf("no ledger: %s", r.Result)
	}
}
