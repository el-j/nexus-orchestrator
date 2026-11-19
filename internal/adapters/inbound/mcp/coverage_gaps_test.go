package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/mcp"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// unencodable is a task whose time JSON cannot write (year 10000), so every tool that returns it as
// JSON hits its encoding error.
var unencodable = domain.Task{ID: "t1", CreatedAt: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)}

func TestTools_ReportATaskTheyCannotEncode(t *testing.T) {
	o := &mockOrch{getTask: unencodable, queue: []domain.Task{unencodable}, backlogTasks: []domain.Task{unencodable}, updateResult: unencodable}
	srv := httptest.NewServer(mcp.NewMcpServer(o, nil))
	defer srv.Close()
	for tool, args := range map[string]map[string]any{
		"get_task":      {"id": "t1"},
		"get_queue":     {},
		"get_all_tasks": {},
		"get_backlog":   {"projectPath": "/p"},
		"update_task":   {"id": "t1", "instruction": "x"},
	} {
		r := callTool(t, srv, 1, tool, args)
		if r.Error == nil || !strings.Contains(r.Error.Message, "marshal") {
			t.Errorf("%s: want a marshal error, got %+v", tool, r)
		}
	}
}

func TestMCPEndpoint_AcceptsAnExactOriginRejectsOthersAndGuardsOnlyMCPPaths(t *testing.T) {
	t.Setenv("NEXUS_MCP_TOKEN", "secret")
	srv := httptest.NewServer(mcp.NewMcpServer(&mockOrch{}, nil))
	defer srv.Close()
	do := func(method, path, origin, auth string) int {
		req, _ := http.NewRequest(method, srv.URL+path, bytes.NewReader([]byte(`{}`)))
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		return resp.StatusCode
	}
	if code := do(http.MethodGet, "/health", "http://localhost", ""); code != http.StatusOK {
		t.Errorf("health from an exact allowed origin without a token = %d, want 200 (not an MCP path)", code)
	}
	if code := do(http.MethodPost, "/mcp", "http://localhost", "Bearer wrong"); code != http.StatusUnauthorized {
		t.Errorf("a wrong token = %d, want 401", code)
	}
	if code := do(http.MethodPost, "/mcp", "https://evil.example", "Bearer secret"); code != http.StatusForbidden {
		t.Errorf("a foreign origin = %d, want 403", code)
	}
}

// waitOrchFlaky answers the first read normally and fails the next ones.
type waitOrchFlaky struct {
	*mockOrch
	reads int
}

func (w *waitOrchFlaky) GetTask(string) (domain.Task, error) {
	w.reads++
	if w.reads > 1 {
		return domain.Task{}, errors.New("database went away")
	}
	return domain.Task{ID: "t1", Status: domain.StatusProcessing}, nil
}

func TestWaitForTask_ReportsAFailedFinalRead(t *testing.T) {
	srv := httptest.NewServer(mcp.NewMcpServer(&waitOrchFlaky{mockOrch: &mockOrch{}}, nil))
	defer srv.Close()
	r := callTool(t, srv, 1, "wait_for_task", map[string]any{"id": "t1", "timeoutSeconds": 1})
	if r.Error == nil || !strings.Contains(r.Error.Message, "database went away") {
		t.Errorf("want the read error after the deadline, got %+v", r)
	}
}

func TestWaitForTask_ReportsAResultItCannotEncode(t *testing.T) {
	done := unencodable
	done.Status = domain.StatusCompleted
	srv := httptest.NewServer(mcp.NewMcpServer(&mockOrch{getTask: done}, nil))
	defer srv.Close()
	r := callTool(t, srv, 1, "wait_for_task", map[string]any{"id": "t1"})
	if r.Error == nil || !strings.Contains(r.Error.Message, "marshal") {
		t.Errorf("got %+v", r)
	}
}

// goneClientStream is a response stream whose client is gone: Flush works, writes fail.
type goneClientStream struct{ h http.Header }

func (b *goneClientStream) Header() http.Header {
	if b.h == nil {
		b.h = http.Header{}
	}
	return b.h
}
func (b *goneClientStream) Write([]byte) (int, error) { return 0, errors.New("client went away") }
func (b *goneClientStream) WriteHeader(int)           {}
func (b *goneClientStream) Flush()                    {}

func TestWaitForTask_StopsStreamingToAClientThatWentAway(t *testing.T) {
	o := newWaitOrch(domain.StatusProcessing)
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 9, "method": "tools/call", "params": map[string]any{
		"name": "wait_for_task", "arguments": map[string]any{"id": "t1", "timeoutSeconds": 1}, "_meta": map[string]any{"progressToken": "p"}}})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	done := make(chan struct{})
	go func() {
		mcp.NewMcpServer(o, nil).ServeHTTP(&goneClientStream{}, req)
		close(done)
	}()
	<-o.subscribed
	o.events <- ports.TaskEvent{Type: ports.EventTaskChunk, TaskID: "t1", Text: "hello", Seq: 1}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the handler kept waiting for a client that cannot be written to")
	}
}

func TestStartMCPServer_ClosesConnectionsThatOutliveTheGracePeriod(t *testing.T) {
	old := *mcp.ShutdownGrace
	*mcp.ShutdownGrace = 50 * time.Millisecond
	defer func() { *mcp.ShutdownGrace = old }()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- mcp.StartMCPServer(ctx, &mockOrch{}, nil, addr) }()
	var conn net.Conn
	for i := 0; i < 250 && conn == nil; i++ {
		if c, err := net.Dial("tcp", addr); err == nil {
			conn = c // a client that connected and has not sent a request yet counts as active
		} else {
			time.Sleep(20 * time.Millisecond)
		}
	}
	if conn == nil {
		t.Fatal("the server never accepted a connection")
	}
	defer conn.Close()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("StartMCPServer = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the server waited for the silent connection instead of closing it after the grace period")
	}
}

func TestJSONResult_ReportsAValueJSONCannotWrite(t *testing.T) {
	if _, err := mcp.JSONResult("get_things", unencodable); err == nil || !strings.Contains(err.Error(), "get_things") {
		t.Errorf("err = %v", err)
	}
	if res, err := mcp.JSONResult("get_things", map[string]int{"a": 1}); err != nil || len(res.Content) != 1 {
		t.Errorf("res=%+v err=%v", res, err)
	}
}

func TestIngestKnowledgeTool_DefaultsTheKindAndReportsAStoreFailure(t *testing.T) {
	ok := httptest.NewServer(mcp.NewMcpServer(&mockOrch{}, &mockBrainForMCP{ingestResult: domain.ProjectKnowledge{ID: "k"}}))
	defer ok.Close()
	r := callTool(t, ok, 1, "ingest_knowledge", map[string]any{"projectPath": "/p", "topic": "t", "content": "c"})
	if r.Error != nil {
		t.Errorf("a single entry without a kind is a learning: %+v", r.Error)
	}
	bad := httptest.NewServer(mcp.NewMcpServer(&mockOrch{}, &mockBrainForMCP{ingestErr: errors.New("disk full")}))
	defer bad.Close()
	r = callTool(t, bad, 2, "ingest_knowledge", map[string]any{"projectPath": "/p", "topic": "t", "content": "c"})
	if r.Error == nil || !strings.Contains(r.Error.Message, "disk full") {
		t.Errorf("got %+v", r)
	}
}
