package llm_openaicompat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

var _ ports.ChatStreamer = (*Adapter)(nil)

// sseServer answers /chat/completions with the given raw body, flushing line by line.
func sseServer(t *testing.T, status int, body string, seen *map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if seen != nil {
			_ = json.NewDecoder(r.Body).Decode(seen)
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
			fmt.Fprint(w, body)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, line := range strings.SplitAfter(body, "\n") {
			fmt.Fprint(w, line)
			w.(http.Flusher).Flush()
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func delta(content string) string {
	return fmt.Sprintf("data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", content)
}

func collect(a *Adapter) ([]string, string, error) {
	var pieces []string
	out, err := a.ChatStream(context.Background(), []domain.Message{{Role: domain.RoleUser, Content: "hi"}}, func(p string) { pieces = append(pieces, p) })
	return pieces, out, err
}

func TestChatStream_DeliversPiecesInOrderAndReturnsTheWholeReply(t *testing.T) {
	var seen map[string]any
	body := "data: {\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\n" + delta("Hel") + ": keep-alive\n\n" + delta("lo") +
		"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: {\"usage\":{\"total_tokens\":3}}\n\n" + "data: [DONE]\n\n"
	srv := sseServer(t, http.StatusOK, body, &seen)
	a := NewAdapter("p", srv.URL, "key", "gpt-4o")

	pieces, out, err := collect(a)
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hello" || strings.Join(pieces, "|") != "Hel|lo" {
		t.Errorf("out=%q pieces=%v", out, pieces)
	}
	if seen["stream"] != true || seen["model"] != "gpt-4o" {
		t.Errorf("request must ask for a stream: %v", seen)
	}
}

func TestChatStream_AcceptsAStreamThatEndsAfterFinishReasonWithoutDone(t *testing.T) {
	body := delta("ok") + "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n"
	_, out, err := collect(NewAdapter("p", sseServer(t, http.StatusOK, body, nil).URL, "", "m"))
	if err != nil || out != "ok" {
		t.Errorf("out=%q err=%v", out, err)
	}
}

func TestChatStream_ATruncatedStreamIsAnError(t *testing.T) {
	pieces, _, err := collect(NewAdapter("p", sseServer(t, http.StatusOK, delta("cut"), nil).URL, "", "m"))
	if err == nil || !strings.Contains(err.Error(), "ended before") {
		t.Errorf("err = %v", err)
	}
	if len(pieces) != 1 {
		t.Errorf("the piece before the cut must still have been delivered: %v", pieces)
	}
}

func TestChatStream_ErrorEventMidStream(t *testing.T) {
	body := delta("part") + "data: {\"error\":{\"message\":\"quota exceeded\"}}\n\n"
	_, _, err := collect(NewAdapter("p", sseServer(t, http.StatusOK, body, nil).URL, "", "m"))
	if err == nil || !strings.Contains(err.Error(), "quota exceeded") {
		t.Errorf("err = %v", err)
	}
}

func TestChatStream_MalformedEventIsAnError(t *testing.T) {
	_, _, err := collect(NewAdapter("p", sseServer(t, http.StatusOK, "data: {nope\n\n", nil).URL, "", "m"))
	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Errorf("err = %v", err)
	}
}

func TestChatStream_HTTPFailures(t *testing.T) {
	_, _, err := collect(NewAdapter("p", sseServer(t, http.StatusTooManyRequests, "", nil).URL, "", "m"))
	if err == nil || !strings.Contains(err.Error(), "rate limited") {
		t.Errorf("429: %v", err)
	}
	_, _, err = collect(NewAdapter("p", sseServer(t, http.StatusBadGateway, `{"error":{"message":"upstream down"}}`, nil).URL, "", "m"))
	if err == nil || !strings.Contains(err.Error(), "502") || !strings.Contains(err.Error(), "upstream down") {
		t.Errorf("502: %v", err)
	}
	_, _, err = collect(NewAdapter("p", "http://127.0.0.1:1", "", "m"))
	if err == nil {
		t.Error("unreachable server must fail")
	}
	if _, err := NewAdapter("p", "http://bad\x7f", "", "m").ChatStream(context.Background(), nil, func(string) {}); err == nil {
		t.Error("bad URL must fail")
	}
}

func TestChatStream_CancelStopsAHangingStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, delta("first"))
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithCancel(context.Background())
	got := make(chan string, 1)
	done := make(chan error, 1)
	go func() {
		_, err := NewAdapter("p", srv.URL, "", "m").ChatStream(ctx, nil, func(p string) { got <- p })
		done <- err
	}()
	select {
	case <-got:
	case <-time.After(5 * time.Second):
		t.Fatal("first piece never arrived")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Error("a cancelled stream must report an error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancel did not stop the stream")
	}
}

func TestChatStream_AnOversizedEventIsAnError(t *testing.T) {
	huge := "data: " + strings.Repeat("x", maxEventBytes+10) + "\n\n"
	_, _, err := collect(NewAdapter("p", sseServer(t, http.StatusOK, huge, nil).URL, "", "m"))
	if err == nil || !strings.Contains(err.Error(), "read stream") {
		t.Errorf("err = %v", err)
	}
}
