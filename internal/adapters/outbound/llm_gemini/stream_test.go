package llm_gemini

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

var _ ports.ChatStreamer = (*Adapter)(nil)

func streamAdapter(t *testing.T, status int, contentType, body string, seen *string) *Adapter {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if seen != nil {
			*seen = r.URL.Path + "?" + r.URL.RawQuery
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)
		for _, line := range strings.SplitAfter(body, "\n") {
			fmt.Fprint(w, line)
			w.(http.Flusher).Flush()
		}
	}))
	t.Cleanup(srv.Close)
	return NewAdapter("key", "gemini-x", srv.URL)
}

func part(texts ...string) string {
	var parts []string
	for _, s := range texts {
		parts = append(parts, fmt.Sprintf("{\"text\":%q}", s))
	}
	return "data: {\"candidates\":[{\"content\":{\"parts\":[" + strings.Join(parts, ",") + "],\"role\":\"model\"}}]}\n\n"
}

const finish = "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"\"}]},\"finishReason\":\"STOP\"}]}\n\n"

func stream(a *Adapter) ([]string, string, error) {
	var pieces []string
	out, err := a.ChatStream(context.Background(),
		[]domain.Message{{Role: domain.RoleUser, Content: "hi"}}, func(p string) { pieces = append(pieces, p) })
	return pieces, out, err
}

func TestChatStream_DeliversPartsAndUsesTheStreamEndpoint(t *testing.T) {
	var seen string
	a := streamAdapter(t, 200, "text/event-stream", part("Hel")+part("lo", " there")+finish, &seen)
	pieces, out, err := stream(a)
	if err != nil || out != "Hello there" || strings.Join(pieces, "|") != "Hel|lo| there" {
		t.Fatalf("out=%q pieces=%v err=%v", out, pieces, err)
	}
	if seen != "/v1beta/models/gemini-x:streamGenerateContent?alt=sse" {
		t.Errorf("endpoint = %q", seen)
	}
}

func TestChatStream_TruncatedAndErrorStreams(t *testing.T) {
	_, out, err := stream(streamAdapter(t, 200, "text/event-stream", part("Hel"), nil))
	if err == nil || !strings.Contains(err.Error(), "before finishReason") || out != "Hel" {
		t.Errorf("truncated: out=%q err=%v", out, err)
	}
	_, _, err = stream(streamAdapter(t, 200, "text/event-stream", part("a")+"data: {\"error\":{\"message\":\"quota\"}}\n\n", nil))
	if err == nil || !strings.Contains(err.Error(), "quota") {
		t.Errorf("error event: %v", err)
	}
	_, _, err = stream(streamAdapter(t, 200, "text/event-stream", "data: {nope\n\n", nil))
	if err == nil || !strings.Contains(err.Error(), "decode stream event") {
		t.Errorf("malformed: %v", err)
	}
}

func TestChatStream_HTTPFailures(t *testing.T) {
	if _, _, err := stream(streamAdapter(t, 429, "application/json", `{}`, nil)); err == nil || !strings.Contains(err.Error(), "429") {
		t.Errorf("429: %v", err)
	}
	if _, _, err := stream(streamAdapter(t, 500, "application/json", `boom`, nil)); err == nil || !strings.Contains(err.Error(), "500") {
		t.Errorf("500: %v", err)
	}
	if _, err := NewAdapter("k", "m", "http://bad\x7f").ChatStream(context.Background(), nil, func(string) {}); err == nil {
		t.Error("bad url")
	}
	if _, err := NewAdapter("k", "m", "http://127.0.0.1:1").ChatStream(context.Background(), nil, func(string) {}); err == nil {
		t.Error("connection refused")
	}
}

func TestChatStream_PlainJSONReplyIsOnePiece(t *testing.T) {
	body := `{"candidates":[{"content":{"parts":[{"text":"whole"}]}}]}`
	pieces, out, err := stream(streamAdapter(t, 200, "application/json", body, nil))
	if err != nil || out != "whole" || len(pieces) != 1 {
		t.Fatalf("out=%q pieces=%v err=%v", out, pieces, err)
	}
	if _, _, err := stream(streamAdapter(t, 200, "application/json", `{"candidates":[]}`, nil)); err == nil {
		t.Error("empty plain reply must fail")
	}
}

func TestChatStream_CancelStopsAHangingStream(t *testing.T) {
	a := NewAdapter("k", "m", hangingServer(t).URL)
	ctx, cancel := cancelSoon()
	defer cancel()
	within(t, "ChatStream", func() {
		if _, err := a.ChatStream(ctx, []domain.Message{{Role: domain.RoleUser, Content: "x"}}, func(string) {}); err == nil {
			t.Error("want an error")
		}
	})
}
