package llm_lmstudio

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

func streamingServer(t *testing.T, status int, contentType, body string, seen *map[string]any) *Adapter {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		if seen != nil {
			_ = json.NewDecoder(r.Body).Decode(seen)
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)
		for _, line := range strings.SplitAfter(body, "\n") {
			fmt.Fprint(w, line)
			w.(http.Flusher).Flush()
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return NewLMStudioAdapter(srv.URL + "/v1")
}

func piece(s string) string {
	return fmt.Sprintf("data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", s)
}

func drive(a *Adapter) ([]string, string, error) {
	var pieces []string
	out, err := a.ChatStream(context.Background(), []domain.Message{{Role: domain.RoleUser, Content: "hi"}},
		func(p string) { pieces = append(pieces, p) })
	return pieces, out, err
}

func TestChatStream_DeliversPiecesAndAsksForAStream(t *testing.T) {
	var seen map[string]any
	a := streamingServer(t, 200, "text/event-stream", piece("Hel")+piece("lo")+
		"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n", &seen)
	pieces, out, err := drive(a)
	if err != nil || out != "Hello" || strings.Join(pieces, "|") != "Hel|lo" {
		t.Fatalf("out=%q pieces=%v err=%v", out, pieces, err)
	}
	if seen["stream"] != true || seen["model"] != "local-model" {
		t.Errorf("request: %v", seen)
	}
}

func TestChatStream_FinishReasonWithoutDoneIsComplete_TruncationIsNot(t *testing.T) {
	_, out, err := drive(streamingServer(t, 200, "text/event-stream", piece("a")+"data: {\"choices\":[{\"finish_reason\":\"stop\"}]}\n\n", nil))
	if err != nil || out != "a" {
		t.Errorf("finish reason: out=%q err=%v", out, err)
	}
	_, _, err = drive(streamingServer(t, 200, "text/event-stream", piece("a"), nil))
	if err == nil || !strings.Contains(err.Error(), "before [DONE]") {
		t.Errorf("truncated: %v", err)
	}
}

func TestChatStream_ErrorMalformedAndEmptyChoices(t *testing.T) {
	_, _, err := drive(streamingServer(t, 200, "text/event-stream", piece("a")+"data: {\"error\":{\"message\":\"model unloaded\"}}\n\n", nil))
	if err == nil || !strings.Contains(err.Error(), "model unloaded") {
		t.Errorf("error event: %v", err)
	}
	_, _, err = drive(streamingServer(t, 200, "text/event-stream", "data: {nope\n\n", nil))
	if err == nil || !strings.Contains(err.Error(), "decode stream event") {
		t.Errorf("malformed: %v", err)
	}
	_, out, err := drive(streamingServer(t, 200, "text/event-stream", "data: {\"choices\":[]}\n\n"+piece("x")+"data: [DONE]\n\n", nil))
	if err != nil || out != "x" {
		t.Errorf("empty choices: out=%q err=%v", out, err)
	}
}

func TestChatStream_HTTPFailuresAndPlainJSON(t *testing.T) {
	if _, _, err := drive(streamingServer(t, 500, "application/json", `{}`, nil)); err == nil || !strings.Contains(err.Error(), "500") {
		t.Errorf("500: %v", err)
	}
	if _, err := NewLMStudioAdapter("http://127.0.0.1:1/v1").ChatStream(context.Background(), nil, func(string) {}); err == nil {
		t.Error("connection refused")
	}
	pieces, out, err := drive(streamingServer(t, 200, "application/json", `{"choices":[{"message":{"content":"whole"}}]}`, nil))
	if err != nil || out != "whole" || len(pieces) != 1 {
		t.Errorf("plain: out=%q pieces=%v err=%v", out, pieces, err)
	}
	if _, _, err := drive(streamingServer(t, 200, "application/json", `{"choices":[]}`, nil)); err == nil {
		t.Error("empty plain reply must fail")
	}
}

func TestChatStream_CancelStopsAHangingStream(t *testing.T) {
	a := NewLMStudioAdapter(hangingServer(t).URL + "/v1")
	ctx, cancel := cancelSoon()
	defer cancel()
	within(t, "ChatStream", func() {
		if _, err := a.ChatStream(ctx, []domain.Message{{Role: domain.RoleUser, Content: "x"}}, func(string) {}); err == nil {
			t.Error("want an error")
		}
	})
}
