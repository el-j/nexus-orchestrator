package llm_ollama

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

func ndjsonServer(t *testing.T, status int, body string, seen *map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if seen != nil {
			_ = json.NewDecoder(r.Body).Decode(seen)
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		for _, line := range strings.SplitAfter(body, "\n") {
			fmt.Fprint(w, line)
			w.(http.Flusher).Flush()
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func line(content string, done bool) string {
	return fmt.Sprintf("{\"message\":{\"role\":\"assistant\",\"content\":%q},\"done\":%v}\n", content, done)
}

func collect(a *Adapter) ([]string, string, error) {
	var pieces []string
	out, err := a.ChatStream(context.Background(), []domain.Message{{Role: domain.RoleUser, Content: "hi"}}, func(p string) { pieces = append(pieces, p) })
	return pieces, out, err
}

func TestChatStream_DeliversPiecesAndReturnsTheWholeReply(t *testing.T) {
	var seen map[string]any
	srv := ndjsonServer(t, http.StatusOK, line("Hel", false)+line("lo", false)+line("", true), &seen)
	pieces, out, err := collect(NewOllamaAdapter(srv.URL, "llama3"))
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hello" || strings.Join(pieces, "|") != "Hel|lo" {
		t.Errorf("out=%q pieces=%v", out, pieces)
	}
	if seen["stream"] != true || seen["model"] != "llama3" {
		t.Errorf("request must ask for a stream: %v", seen)
	}
}

func TestChatStream_TruncatedStreamIsAnError(t *testing.T) {
	pieces, _, err := collect(NewOllamaAdapter(ndjsonServer(t, http.StatusOK, line("cut", false), nil).URL, "m"))
	if err == nil || !strings.Contains(err.Error(), "ended before") || len(pieces) != 1 {
		t.Errorf("err=%v pieces=%v", err, pieces)
	}
}

func TestChatStream_ErrorLineAndMalformedLine(t *testing.T) {
	_, _, err := collect(NewOllamaAdapter(ndjsonServer(t, http.StatusOK, line("a", false)+"{\"error\":\"model crashed\"}\n", nil).URL, "m"))
	if err == nil || !strings.Contains(err.Error(), "model crashed") {
		t.Errorf("error line: %v", err)
	}
	_, _, err = collect(NewOllamaAdapter(ndjsonServer(t, http.StatusOK, "{nope\n", nil).URL, "m"))
	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Errorf("malformed: %v", err)
	}
}

func TestChatStream_HTTPFailures(t *testing.T) {
	_, _, err := collect(NewOllamaAdapter(ndjsonServer(t, http.StatusInternalServerError, "", nil).URL, "m"))
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Errorf("500: %v", err)
	}
	if _, _, err = collect(NewOllamaAdapter("http://127.0.0.1:1", "m")); err == nil {
		t.Error("unreachable server must fail")
	}
	if _, err := NewOllamaAdapter("http://bad\x7f", "m").ChatStream(context.Background(), nil, func(string) {}); err == nil {
		t.Error("bad URL must fail")
	}
}

func TestChatStream_OversizedLineIsAnError(t *testing.T) {
	_, _, err := collect(NewOllamaAdapter(ndjsonServer(t, http.StatusOK, strings.Repeat("x", maxLineBytes+10)+"\n", nil).URL, "m"))
	if err == nil || !strings.Contains(err.Error(), "read stream") {
		t.Errorf("err = %v", err)
	}
}

func TestChatStream_CancelStopsAHangingStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, line("first", false))
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithCancel(context.Background())
	got := make(chan string, 1)
	done := make(chan error, 1)
	go func() {
		_, err := NewOllamaAdapter(srv.URL, "m").ChatStream(ctx, nil, func(p string) { got <- p })
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

// An Ollama-compatible server that ignores "stream": true and answers with one JSON object
// still works: the reply arrives as one piece.
func TestChatStream_FallsBackToASingleJSONReply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message":{"role":"assistant","content":"whole reply"}}`)
	}))
	t.Cleanup(srv.Close)
	pieces, out, err := collect(NewOllamaAdapter(srv.URL, "m"))
	if err != nil || out != "whole reply" || len(pieces) != 1 {
		t.Errorf("out=%q pieces=%v err=%v", out, pieces, err)
	}
}

func TestChatStream_BrokenSingleJSONReplyIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{nope`)
	}))
	t.Cleanup(srv.Close)
	if _, _, err := collect(NewOllamaAdapter(srv.URL, "m")); err == nil {
		t.Error("want an error")
	}
}
