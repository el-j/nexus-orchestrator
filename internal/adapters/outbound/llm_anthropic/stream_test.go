package llm_anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

var _ ports.ChatStreamer = (*Adapter)(nil)

func streamServer(t *testing.T, status int, contentType, body string, seen *map[string]any) *Adapter {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if seen != nil {
			_ = json.NewDecoder(r.Body).Decode(seen)
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)
		for _, line := range strings.SplitAfter(body, "\n") {
			fmt.Fprint(w, line)
			w.(http.Flusher).Flush()
		}
	}))
	t.Cleanup(srv.Close)
	a := NewAdapter("k", "claude-x")
	a.baseURL = srv.URL
	return a
}

func textDelta(s string) string {
	return fmt.Sprintf("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":%q}}\n\n", s)
}

func run(a *Adapter) ([]string, string, error) {
	var pieces []string
	out, err := a.ChatStream(context.Background(),
		[]domain.Message{{Role: domain.RoleSystem, Content: "be brief"}, {Role: domain.RoleUser, Content: "hi"}},
		func(p string) { pieces = append(pieces, p) })
	return pieces, out, err
}

const (
	start = "event: message_start\ndata: {\"type\":\"message_start\"}\n\nevent: ping\ndata: {\"type\":\"ping\"}\n\n"
	stop  = "event: message_delta\ndata: {\"type\":\"message_delta\"}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
)

func TestChatStream_DeliversTextDeltasAndAsksForAStream(t *testing.T) {
	var seen map[string]any
	a := streamServer(t, 200, "text/event-stream", start+textDelta("Hel")+
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{}\"}}\n\n"+
		textDelta("lo")+stop, &seen)
	pieces, out, err := run(a)
	if err != nil || out != "Hello" || strings.Join(pieces, "|") != "Hel|lo" {
		t.Fatalf("out=%q pieces=%v err=%v", out, pieces, err)
	}
	sysBlocks, _ := seen["system"].([]any)
	sysBlock, _ := map[string]any(nil), false
	if len(sysBlocks) == 1 {
		sysBlock, _ = sysBlocks[0].(map[string]any)
	}
	cc, _ := sysBlock["cache_control"].(map[string]any)
	if seen["stream"] != true || sysBlock["text"] != "be brief" || cc["type"] != "ephemeral" {
		t.Errorf("request: %v", seen)
	}
}

func TestChatStream_TruncatedStreamIsAnError(t *testing.T) {
	_, out, err := run(streamServer(t, 200, "text/event-stream", start+textDelta("Hel"), nil))
	if err == nil || !strings.Contains(err.Error(), "before message_stop") || out != "Hel" {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestChatStream_ErrorEventAndMalformedEvent(t *testing.T) {
	_, _, err := run(streamServer(t, 200, "text/event-stream", start+textDelta("a")+
		"event: error\ndata: {\"type\":\"error\",\"error\":{\"type\":\"overloaded_error\",\"message\":\"Overloaded\"}}\n\n", nil))
	if err == nil || !strings.Contains(err.Error(), "Overloaded") {
		t.Errorf("error event: %v", err)
	}
	_, _, err = run(streamServer(t, 200, "text/event-stream", "data: {nope\n\n", nil))
	if err == nil || !strings.Contains(err.Error(), "decode stream event") {
		t.Errorf("malformed: %v", err)
	}
}

func TestChatStream_HTTPFailures(t *testing.T) {
	_, _, err := run(streamServer(t, 429, "application/json", `{}`, nil))
	if err == nil || !strings.Contains(err.Error(), "429") {
		t.Errorf("429: %v", err)
	}
	_, _, err = run(streamServer(t, 400, "application/json", `{"error":{"message":"prompt is too long"}}`, nil))
	if err == nil || !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "prompt is too long") {
		t.Errorf("400: %v", err)
	}
	a := NewAdapter("k", "m")
	a.baseURL = "http://bad\x7f"
	if _, err := a.ChatStream(context.Background(), []domain.Message{{Role: domain.RoleUser, Content: "x"}}, func(string) {}); err == nil {
		t.Error("bad url")
	}
	a.baseURL = "http://127.0.0.1:1"
	if _, err := a.ChatStream(context.Background(), []domain.Message{{Role: domain.RoleUser, Content: "x"}}, func(string) {}); err == nil {
		t.Error("connection refused")
	}
	if _, err := NewAdapter("k", "m").ChatStream(context.Background(), nil, func(string) {}); err == nil {
		t.Error("no user message")
	}
}

func TestChatStream_PlainJSONReplyIsOnePiece(t *testing.T) {
	a := streamServer(t, 200, "application/json", `{"content":[{"type":"text","text":"whole"}]}`, nil)
	pieces, out, err := run(a)
	if err != nil || out != "whole" || len(pieces) != 1 {
		t.Fatalf("out=%q pieces=%v err=%v", out, pieces, err)
	}
	_, _, err = run(streamServer(t, 200, "application/json", `{"content":[]}`, nil))
	if err == nil {
		t.Error("empty plain reply must fail")
	}
}

func TestChatStream_CancelStopsAHangingStream(t *testing.T) {
	srv := hangingServer(t)
	a := NewAdapter("k", "m")
	a.baseURL = srv.URL
	ctx, cancel := cancelSoon()
	defer cancel()
	within(t, "ChatStream", func() {
		if _, err := a.ChatStream(ctx, []domain.Message{{Role: domain.RoleUser, Content: "x"}}, func(string) {}); err == nil {
			t.Error("want an error")
		}
	})
}
