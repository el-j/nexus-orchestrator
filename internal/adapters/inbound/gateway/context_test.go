package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

// recordingClient remembers the messages it was asked to answer.
type recordingClient struct {
	fakeClient
	got []domain.Message
}

func (r *recordingClient) Chat(_ context.Context, msgs []domain.Message) (string, error) {
	r.got = msgs
	return r.chatReply, r.chatErr
}

func (r *recordingClient) ChatStream(_ context.Context, msgs []domain.Message, onChunk func(string)) (string, error) {
	r.got = msgs
	onChunk(r.chatReply)
	return r.chatReply, nil
}

type stubContext struct {
	text            string
	project, asking string
}

func (s *stubContext) Context(_ context.Context, project, question string) string {
	s.project, s.asking = project, question
	return s.text
}

type stubUsage struct {
	mu   sync.Mutex
	recs []domain.UsageRecord
}

func (s *stubUsage) RecordUsage(_ context.Context, rec domain.UsageRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recs = append(s.recs, rec)
	return nil
}

func newRecording(reply string) *recordingClient {
	return &recordingClient{fakeClient: fakeClient{name: "Anthropic", model: "claude", alive: true, chatReply: reply,
		usage: &domain.Usage{Model: "claude", InputTokens: 7, OutputTokens: 3}}}
}

func userAsk(q string) ChatCompletionRequest {
	return ChatCompletionRequest{Model: "coder", Messages: []ChatMessage{
		{Role: "system", Content: "be brief"}, {Role: "user", Content: q},
	}}
}

// The project's memory goes in front of the conversation as a system message; what the tool sent stays.
func TestExecuteChat_PutsTheProjectContextBeforeTheConversation(t *testing.T) {
	c := newRecording("ok")
	ctxProv := &stubContext{text: "PROJECT RULES"}
	gw := NewGateway(Config{Context: ctxProv}, services.NewDiscoveryService(c))
	if _, err := gw.ExecuteChat(context.Background(), userAsk("how do I add a route?"), "/proj"); err != nil {
		t.Fatal(err)
	}
	if len(c.got) != 3 || c.got[0].Role != domain.RoleSystem || !strings.Contains(c.got[0].Content, "PROJECT RULES") || c.got[1].Content != "be brief" {
		t.Fatalf("messages = %+v", c.got)
	}
	if ctxProv.project != "/proj" || ctxProv.asking != "how do I add a route?" {
		t.Errorf("context asked for %q / %q", ctxProv.project, ctxProv.asking)
	}
}

func TestStreamChat_PutsTheProjectContextBeforeTheConversation(t *testing.T) {
	c := newRecording("ok")
	gw := NewGateway(Config{Context: &stubContext{text: "PROJECT RULES"}}, services.NewDiscoveryService(c))
	if _, err := gw.StreamChat(context.Background(), userAsk("q"), "/proj", func(string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if len(c.got) != 3 || !strings.Contains(c.got[0].Content, "PROJECT RULES") {
		t.Fatalf("messages = %+v", c.got)
	}
}

// Without a project, a provider of context or any context text the conversation is passed on as it is.
func TestExecuteChat_LeavesTheConversationAloneWithoutContext(t *testing.T) {
	for name, tc := range map[string]struct {
		prov    ContextProvider
		project string
	}{
		"no project":  {&stubContext{text: "X"}, ""},
		"no provider": {nil, "/proj"},
		"empty text":  {&stubContext{text: ""}, "/proj"},
	} {
		c := newRecording("ok")
		gw := NewGateway(Config{Context: tc.prov}, services.NewDiscoveryService(c))
		if _, err := gw.ExecuteChat(context.Background(), userAsk("q"), tc.project); err != nil {
			t.Fatal(err)
		}
		if len(c.got) != 2 {
			t.Errorf("%s: %d messages, want the 2 that were sent", name, len(c.got))
		}
	}
}

// A call that reports its tokens goes into the usage ledger under the provider, role and project.
func TestExecuteChatAndStream_RecordUsage(t *testing.T) {
	c := newRecording("ok")
	led := &stubUsage{}
	gw := NewGateway(Config{Usage: led}, services.NewDiscoveryService(c))
	if _, err := gw.ExecuteChat(context.Background(), userAsk("q"), "/proj"); err != nil {
		t.Fatal(err)
	}
	if _, err := gw.StreamChat(context.Background(), userAsk("q"), "/proj", func(string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if len(led.recs) != 2 {
		t.Fatalf("recorded %d, want 2", len(led.recs))
	}
	r := led.recs[0]
	if r.Provider != "Anthropic" || r.Role != "coder" || r.ProjectPath != "/proj" || r.InputTokens != 7 {
		t.Errorf("record = %+v", r)
	}
	// A client that reports nothing records nothing.
	led2 := &stubUsage{}
	plain := &fakeClient{name: "Plain", model: "m", alive: true, chatReply: "x"}
	gw2 := NewGateway(Config{Usage: led2}, services.NewDiscoveryService(plain))
	if _, err := gw2.ExecuteChat(context.Background(), userAsk("q"), ""); err != nil || len(led2.recs) != 0 {
		t.Errorf("err %v, records %d", err, len(led2.recs))
	}
}

// The project of a request is named in X-Nexus-Project, else the one the server was started with.
func TestServer_TakesTheProjectFromTheHeader(t *testing.T) {
	c := newRecording("ok")
	ctxProv := &stubContext{text: "RULES"}
	cfg := Config{Context: ctxProv, ProjectPath: "/default"}
	srv := NewServer(cfg, NewGateway(cfg, services.NewDiscoveryService(c)))
	post := func(header string) {
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"model":"coder","messages":[{"role":"user","content":"q"}]}`))
		if header != "" {
			req.Header.Set("X-Nexus-Project", header)
		}
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("HTTP %d %s", w.Code, w.Body.String())
		}
	}
	post("/from/header")
	if ctxProv.project != "/from/header" {
		t.Errorf("project = %q", ctxProv.project)
	}
	post("")
	if ctxProv.project != "/default" {
		t.Errorf("project = %q, want the default", ctxProv.project)
	}
}

// Mounted in the daemon the gateway answers through Routes, without its own CORS and key check: the
// daemon's guards stand in front of it.
func TestServer_RoutesCarryNeitherCORSNorKeyCheck(t *testing.T) {
	cfg := Config{APIKey: "k"}
	srv := NewServer(cfg, NewGateway(cfg, services.NewDiscoveryService(newRecording("ok"))))
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()
	srv.Routes().ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("HTTP %d, CORS %q", w.Code, w.Header().Get("Access-Control-Allow-Origin"))
	}
}

// A client that has no usage report at all is answered and nothing is recorded.
func TestExecuteChat_ClientWithoutUsageReportRecordsNothing(t *testing.T) {
	led := &stubUsage{}
	c := &fakeNonStreamClient{name: "Bare", model: "m", alive: true}
	gw := NewGateway(Config{Usage: led}, services.NewDiscoveryService(c))
	if _, err := gw.ExecuteChat(context.Background(), userAsk("q"), "/p"); err != nil || len(led.recs) != 0 {
		t.Errorf("err %v, records %d", err, len(led.recs))
	}
}

// A usage report without a model is entered under the model the client answered with.
func TestExecuteChat_UsageWithoutModelTakesTheActiveModel(t *testing.T) {
	led := &stubUsage{}
	c := newRecording("ok")
	c.usage = &domain.Usage{InputTokens: 1}
	gw := NewGateway(Config{Usage: led}, services.NewDiscoveryService(c))
	if _, err := gw.ExecuteChat(context.Background(), userAsk("q"), ""); err != nil || len(led.recs) != 1 || led.recs[0].Model != "claude" {
		t.Errorf("err %v, records %+v", err, led.recs)
	}
}

// OpenAI clients may send a message's content as a list of typed parts instead of a string; the text
// parts are joined, other parts (images) are ignored, and a plain string still works.
func TestChatMessage_AcceptsContentAsStringOrParts(t *testing.T) {
	cases := map[string]string{
		`{"role":"user","content":"plain"}`:                                                                      "plain",
		`{"role":"user","content":[{"type":"text","text":"one"},{"type":"text","text":"two"}]}`:                  "one\ntwo",
		`{"role":"user","content":[{"type":"image_url","image_url":{"url":"x"}},{"type":"text","text":"only"}]}`: "only",
		`{"role":"user","content":null}`:                                                                         "",
		`{"role":"assistant","content":""}`:                                                                      "",
		`{"role":"user","content":[]}`:                                                                           "",
	}
	for in, want := range cases {
		var m ChatMessage
		if err := json.Unmarshal([]byte(in), &m); err != nil || m.Content != want {
			t.Errorf("%s -> %q, %v; want %q", in, m.Content, err, want)
		}
	}
	var m ChatMessage
	if err := json.Unmarshal([]byte(`{"role":"user","content":7}`), &m); err == nil {
		t.Error("a number is not content")
	}
	if err := json.Unmarshal([]byte(`{"role":5}`), &m); err == nil {
		t.Error("a role that is not a string must fail")
	}
	// What the gateway sends back stays a plain string.
	out, _ := json.Marshal(ChatMessage{Role: "assistant", Content: "hi"})
	if string(out) != `{"role":"assistant","content":"hi"}` {
		t.Errorf("marshal = %s", out)
	}
}
