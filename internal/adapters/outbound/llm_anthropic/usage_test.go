package llm_anthropic

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

var _ ports.UsageReporter = (*Adapter)(nil)

func TestUsage_NothingBeforeTheFirstCall(t *testing.T) {
	if _, ok := NewAdapter("k", "m").LastUsage(); ok {
		t.Error("no usage yet")
	}
}

func TestUsage_FromAMessagesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"model":"claude-x-1","content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":11,"output_tokens":7,"cache_creation_input_tokens":100,"cache_read_input_tokens":20}}`)
	}))
	defer srv.Close()
	a := NewAdapter("k", "claude-x")
	a.baseURL = srv.URL
	if _, err := a.GenerateCode(context.Background(), "p"); err != nil {
		t.Fatal(err)
	}
	u, ok := a.LastUsage()
	want := domain.Usage{Model: "claude-x-1", InputTokens: 11, OutputTokens: 7, CachedTokens: 120}
	if !ok || u != want {
		t.Errorf("usage %+v %v; want %+v", u, ok, want)
	}
}

func TestUsage_AResponseWithoutUsageKeepsTheModelOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"content":[{"type":"text","text":"hi"}]}`)
	}))
	defer srv.Close()
	a := NewAdapter("k", "claude-x")
	a.baseURL = srv.URL
	if _, err := a.GenerateCode(context.Background(), "p"); err != nil {
		t.Fatal(err)
	}
	if u, ok := a.LastUsage(); !ok || u.InputTokens != 0 || u.Model != "claude-x" {
		t.Errorf("%+v %v", u, ok)
	}
}

func TestUsage_FromAStream(t *testing.T) {
	body := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"model\":\"claude-s-2\",\"usage\":{\"input_tokens\":25,\"cache_read_input_tokens\":5,\"output_tokens\":1}}}\n\n" +
		textDelta("he") + textDelta("llo") +
		"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":15}}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
	a := streamServer(t, 200, "text/event-stream", body, nil)
	if _, out, err := run(a); err != nil || out != "hello" {
		t.Fatalf("%q %v", out, err)
	}
	u, ok := a.LastUsage()
	want := domain.Usage{Model: "claude-s-2", InputTokens: 25, OutputTokens: 15, CachedTokens: 5}
	if !ok || u != want {
		t.Errorf("usage %+v %v; want %+v", u, ok, want)
	}
}

func TestUsage_AStreamServedAsOneJSONBodyStillReportsUsage(t *testing.T) {
	a := streamServer(t, 200, "application/json", `{"content":[{"type":"text","text":"x"}],"usage":{"input_tokens":4,"output_tokens":2}}`, nil)
	if _, _, err := run(a); err != nil {
		t.Fatal(err)
	}
	if u, ok := a.LastUsage(); !ok || u.InputTokens != 4 || u.OutputTokens != 2 {
		t.Errorf("%+v %v", u, ok)
	}
}
