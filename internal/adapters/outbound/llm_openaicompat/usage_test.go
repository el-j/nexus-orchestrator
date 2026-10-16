package llm_openaicompat

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
	if _, ok := NewAdapter("p", "http://x", "", "m").LastUsage(); ok {
		t.Error("no usage yet")
	}
}

func TestUsage_FromACompletion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"model":"gpt-x-2","choices":[{"message":{"content":"hi"}}],"usage":{"prompt_tokens":30,"completion_tokens":9,"prompt_tokens_details":{"cached_tokens":12}}}`)
	}))
	defer srv.Close()
	a := NewAdapter("p", srv.URL, "k", "gpt-x")
	if _, err := a.GenerateCode(context.Background(), "q"); err != nil {
		t.Fatal(err)
	}
	u, ok := a.LastUsage()
	want := domain.Usage{Model: "gpt-x-2", InputTokens: 30, OutputTokens: 9, CachedTokens: 12}
	if !ok || u != want {
		t.Errorf("usage %+v %v; want %+v", u, ok, want)
	}
}

func TestUsage_AStreamAsksForUsageAndReadsTheFinalChunk(t *testing.T) {
	var seen map[string]any
	body := delta("he") + delta("llo") +
		"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: {\"model\":\"gpt-s-1\",\"choices\":[],\"usage\":{\"prompt_tokens\":21,\"completion_tokens\":6}}\n\n" +
		"data: [DONE]\n\n"
	srv := sseServer(t, http.StatusOK, body, &seen)
	a := NewAdapter("p", srv.URL, "k", "gpt-x")
	if _, out, err := collect(a); err != nil || out != "hello" {
		t.Fatalf("%q %v", out, err)
	}
	opts, _ := seen["stream_options"].(map[string]any)
	if opts["include_usage"] != true {
		t.Errorf("the stream must ask for usage: %v", seen)
	}
	u, ok := a.LastUsage()
	if !ok || u.InputTokens != 21 || u.OutputTokens != 6 || u.Model != "gpt-s-1" {
		t.Errorf("usage %+v %v", u, ok)
	}
}

func TestUsage_AServerThatSendsNoUsageStillLeavesTheModel(t *testing.T) {
	srv := sseServer(t, http.StatusOK, delta("x")+"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n", nil)
	a := NewAdapter("p", srv.URL, "k", "gpt-x")
	if _, _, err := collect(a); err != nil {
		t.Fatal(err)
	}
	if u, ok := a.LastUsage(); !ok || u.InputTokens != 0 || u.Model != "gpt-x" {
		t.Errorf("%+v %v", u, ok)
	}
}

func TestUsage_AStreamServedAsOneCompletionStillReportsUsage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"x"}}],"usage":{"prompt_tokens":2,"completion_tokens":1}}`)
	}))
	defer srv.Close()
	a := NewAdapter("p", srv.URL, "k", "gpt-x")
	if _, _, err := collect(a); err != nil {
		t.Fatal(err)
	}
	if u, ok := a.LastUsage(); !ok || u.InputTokens != 2 {
		t.Errorf("%+v %v", u, ok)
	}
}
