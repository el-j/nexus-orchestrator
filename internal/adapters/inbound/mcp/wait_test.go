package mcp_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/mcp"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// waitOrch is an orchestrator whose task changes under the test's control and that can publish
// task events like the real one.
type waitOrch struct {
	*mockOrch
	mu         sync.Mutex
	task       domain.Task
	getErr     error
	events     chan ports.TaskEvent
	cancelled  chan struct{}
	once       sync.Once
	subscribed chan struct{} // closed on the first subscription
	subOnce    sync.Once
	firstRead  chan struct{} // closed when the handler has read the task once
	readOnce   sync.Once
}

func newWaitOrch(status domain.TaskStatus) *waitOrch {
	return &waitOrch{
		mockOrch:   &mockOrch{},
		task:       domain.Task{ID: "t1", Status: status},
		events:     make(chan ports.TaskEvent, 16),
		cancelled:  make(chan struct{}),
		subscribed: make(chan struct{}),
		firstRead:  make(chan struct{}),
	}
}

func (w *waitOrch) GetTask(string) (domain.Task, error) {
	w.mu.Lock()
	task, err := w.task, w.getErr
	w.mu.Unlock()
	w.readOnce.Do(func() { close(w.firstRead) })
	return task, err
}

func (w *waitOrch) set(status domain.TaskStatus, logs string) {
	w.mu.Lock()
	w.task.Status, w.task.Logs = status, logs
	w.mu.Unlock()
}

func (w *waitOrch) SubscribeTaskEvents() (<-chan ports.TaskEvent, func()) {
	w.subOnce.Do(func() { close(w.subscribed) })
	return w.events, func() { w.once.Do(func() { close(w.cancelled) }) }
}

// noEvents hides SubscribeTaskEvents, like an orchestrator that cannot publish events.
type noEvents struct{ ports.Orchestrator }

func callWait(t *testing.T, h http.Handler, args map[string]any, meta map[string]any, accept string) (msgs []map[string]any, contentType string) {
	t.Helper()
	params := map[string]any{"name": "wait_for_task", "arguments": args}
	if meta != nil {
		params["_meta"] = meta
	}
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 9, "method": "tools/call", "params": params})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rpcMessages(t, rec.Result())
}

// rpcMessages reads the JSON-RPC messages out of a plain JSON or an event-stream body.
func rpcMessages(t *testing.T, resp *http.Response) ([]map[string]any, string) {
	t.Helper()
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out []map[string]any
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		sc := bufio.NewScanner(bytes.NewReader(raw))
		for sc.Scan() {
			if data, ok := strings.CutPrefix(sc.Text(), "data: "); ok {
				var m map[string]any
				if err := json.Unmarshal([]byte(data), &m); err != nil {
					t.Fatalf("bad event %q: %v", data, err)
				}
				out = append(out, m)
			}
		}
		return out, resp.Header.Get("Content-Type")
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("bad body %q: %v", raw, err)
	}
	return []map[string]any{m}, resp.Header.Get("Content-Type")
}

func taskFrom(t *testing.T, msg map[string]any) map[string]any {
	t.Helper()
	res, ok := msg["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result in %v", msg)
	}
	text := res["content"].([]any)[0].(map[string]any)["text"].(string)
	var task map[string]any
	if err := json.Unmarshal([]byte(text), &task); err != nil {
		t.Fatal(err)
	}
	return task
}

func TestWaitForTask_ReturnsAtOnceWhenTheTaskIsAlreadyDone(t *testing.T) {
	o := newWaitOrch(domain.StatusCompleted)
	o.set(domain.StatusCompleted, "all good")
	msgs, _ := callWait(t, mcp.NewMcpServer(o, nil), map[string]any{"id": "t1"}, nil, "")
	task := taskFrom(t, msgs[0])
	if task["status"] != "COMPLETED" || task["logs"] != "all good" || task["timedOut"] != nil {
		t.Errorf("task = %v", task)
	}
}

func TestWaitForTask_StreamsProgressWhenAskedAndEndsWithTheResult(t *testing.T) {
	o := newWaitOrch(domain.StatusProcessing)
	srv := httptest.NewServer(mcp.NewMcpServer(o, nil))
	defer srv.Close()

	go func() {
		<-o.firstRead // the handler has subscribed and seen the task still running; a task already done would end the wait before any event
		o.events <- ports.TaskEvent{Type: ports.EventTaskRestart, TaskID: "t1"}
		o.events <- ports.TaskEvent{Type: ports.EventTaskChunk, TaskID: "other", Text: "NOT MINE", Seq: 1}
		o.events <- ports.TaskEvent{Type: ports.EventTaskChunk, TaskID: "t1", Text: "Hel", Seq: 1}
		o.events <- ports.TaskEvent{Type: ports.EventTaskChunk, TaskID: "t1", Text: "lo", Seq: 2}
		o.set(domain.StatusCompleted, "done")
		o.events <- ports.TaskEvent{Type: ports.EventTaskCompleted, TaskID: "t1", Status: domain.StatusCompleted}
	}()

	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 9, "method": "tools/call", "params": map[string]any{
		"name": "wait_for_task", "arguments": map[string]any{"id": "t1"}, "_meta": map[string]any{"progressToken": "tok-1"}}})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/mcp", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content type = %q", ct)
	}
	msgs, _ := rpcMessages(t, resp)
	if len(msgs) != 4 {
		t.Fatalf("want 3 progress notifications and the result, got %d: %v", len(msgs), msgs)
	}
	var texts []string
	for i, m := range msgs[:3] {
		params := m["params"].(map[string]any)
		if m["method"] != "notifications/progress" || params["progressToken"] != "tok-1" || params["progress"] != float64(i+1) {
			t.Errorf("notification %d = %v", i, m)
		}
		texts = append(texts, params["message"].(string))
	}
	if texts[0] != "↻ new attempt" || texts[1] != "Hel" || texts[2] != "lo" {
		t.Errorf("progress messages = %q", texts)
	}
	if task := taskFrom(t, msgs[3]); task["status"] != "COMPLETED" || msgs[3]["id"] != float64(9) {
		t.Errorf("result = %v", msgs[3])
	}
	select {
	case <-o.cancelled:
	case <-time.After(time.Second):
		t.Error("the event subscription was not released")
	}
}

func TestWaitForTask_PlainJSONWithoutProgressTokenOrEventStreamAccept(t *testing.T) {
	for name, tc := range map[string]struct {
		meta   map[string]any
		accept string
	}{
		"no token":  {nil, "text/event-stream"},
		"no accept": {map[string]any{"progressToken": 1}, "application/json"},
	} {
		t.Run(name, func(t *testing.T) {
			o := newWaitOrch(domain.StatusProcessing)
			go func() {
				<-o.subscribed
				o.events <- ports.TaskEvent{Type: ports.EventTaskChunk, TaskID: "t1", Text: "x", Seq: 1}
				o.set(domain.StatusFailed, "boom")
				o.events <- ports.TaskEvent{Type: ports.EventTaskFailed, TaskID: "t1"}
			}()
			msgs, ct := callWait(t, mcp.NewMcpServer(o, nil), map[string]any{"id": "t1"}, tc.meta, tc.accept)
			if !strings.HasPrefix(ct, "application/json") {
				t.Fatalf("content type = %q", ct)
			}
			if len(msgs) != 1 || taskFrom(t, msgs[0])["status"] != "FAILED" {
				t.Errorf("msgs = %v", msgs)
			}
		})
	}
}

// A writer that cannot flush (the legacy /messages transport buffers the reply) gets the result
// only, even if the caller asked for progress.
type nonFlushingWriter struct {
	h    http.Header
	body bytes.Buffer
}

func (p *nonFlushingWriter) Header() http.Header         { return p.h }
func (p *nonFlushingWriter) Write(b []byte) (int, error) { return p.body.Write(b) }
func (p *nonFlushingWriter) WriteHeader(int)             {}

func TestWaitForTask_NoProgressOnAWriterThatCannotFlush(t *testing.T) {
	o := newWaitOrch(domain.StatusCompleted)
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
		"name": "wait_for_task", "arguments": map[string]any{"id": "t1"}, "_meta": map[string]any{"progressToken": 7}}})
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(b))
	req.Header.Set("Accept", "text/event-stream")
	w := &nonFlushingWriter{h: http.Header{}}
	mcp.NewMcpServer(o, nil).ServeHTTP(w, req)
	var m map[string]any
	if err := json.Unmarshal(w.body.Bytes(), &m); err != nil || taskFrom(t, m)["status"] != "COMPLETED" {
		t.Errorf("body = %q err=%v", w.body.String(), err)
	}
}

func TestWaitForTask_TimeoutReturnsTheCurrentTaskMarkedTimedOut(t *testing.T) {
	o := newWaitOrch(domain.StatusProcessing)
	start := time.Now()
	msgs, _ := callWait(t, mcp.NewMcpServer(o, nil), map[string]any{"id": "t1", "timeoutSeconds": 1}, nil, "")
	task := taskFrom(t, msgs[0])
	if task["status"] != "PROCESSING" || task["timedOut"] != true {
		t.Errorf("task = %v", task)
	}
	if d := time.Since(start); d < 900*time.Millisecond || d > 3*time.Second {
		t.Errorf("waited %v, want about 1s", d)
	}
}

func TestWaitForTask_WithoutAnEventSourceItPollsTheTask(t *testing.T) {
	old := *mcp.WaitPollInterval
	*mcp.WaitPollInterval = 10 * time.Millisecond
	defer func() { *mcp.WaitPollInterval = old }()

	o := newWaitOrch(domain.StatusQueued)
	time.AfterFunc(60*time.Millisecond, func() { o.set(domain.StatusCompleted, "polled") })
	msgs, _ := callWait(t, mcp.NewMcpServer(noEvents{o}, nil), map[string]any{"id": "t1"}, nil, "")
	if task := taskFrom(t, msgs[0]); task["status"] != "COMPLETED" {
		t.Errorf("task = %v", task)
	}
}

func TestWaitForTask_AMissedEventIsCaughtByThePollAndAClosedFeedIsSurvived(t *testing.T) {
	old := *mcp.WaitPollInterval
	*mcp.WaitPollInterval = 10 * time.Millisecond
	defer func() { *mcp.WaitPollInterval = old }()

	o := newWaitOrch(domain.StatusProcessing)
	close(o.events) // the feed ends; the poll must still find the result
	time.AfterFunc(60*time.Millisecond, func() { o.set(domain.StatusCancelled, "") })
	msgs, _ := callWait(t, mcp.NewMcpServer(o, nil), map[string]any{"id": "t1"}, nil, "")
	if task := taskFrom(t, msgs[0]); task["status"] != "CANCELLED" {
		t.Errorf("task = %v", task)
	}
}

func TestWaitForTask_StopsWhenTheClientGoesAway(t *testing.T) {
	o := newWaitOrch(domain.StatusProcessing)
	srv := httptest.NewServer(mcp.NewMcpServer(o, nil))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{
		"name": "wait_for_task", "arguments": map[string]any{"id": "t1"}}})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/mcp", bytes.NewReader(b))
	errc := make(chan error, 1)
	go func() {
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
		}
		errc <- err
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()
	if err := <-errc; err == nil {
		t.Error("the client's cancel must abort the request")
	}
	select {
	case <-o.cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("the handler kept waiting after the client left")
	}
}

func TestWaitForTask_InvalidArgumentsAndLookupErrors(t *testing.T) {
	o := newWaitOrch(domain.StatusCompleted)
	h := mcp.NewMcpServer(o, nil)
	for name, args := range map[string]any{"missing id": map[string]any{}, "bad type": map[string]any{"id": 5}} {
		msgs, _ := callWait(t, h, toMap(args), nil, "")
		if msgs[0]["error"] == nil {
			t.Errorf("%s: want an error, got %v", name, msgs[0])
		}
	}
	o.mu.Lock()
	o.getErr = errors.New("gone")
	o.mu.Unlock()
	if msgs, _ := callWait(t, h, map[string]any{"id": "t1"}, nil, ""); msgs[0]["error"] == nil {
		t.Errorf("lookup error not reported: %v", msgs[0])
	}
}

func toMap(v any) map[string]any { return v.(map[string]any) }

func TestWaitForTask_TheTimeoutIsClamped(t *testing.T) {
	o := newWaitOrch(domain.StatusCompleted)
	for _, secs := range []int{-5, 0, 999999} {
		msgs, _ := callWait(t, mcp.NewMcpServer(o, nil), map[string]any{"id": "t1", "timeoutSeconds": secs}, nil, "")
		if taskFrom(t, msgs[0])["status"] != "COMPLETED" {
			t.Errorf("timeoutSeconds=%d: %v", secs, msgs[0])
		}
	}
}

// A WAITING task is not finished: every provider is cooling down and the task is queued again at the
// reset. wait_for_task must keep waiting through it, not hand the caller a task that has not run.
func TestWaitForTask_KeepsWaitingWhileTheTaskWaitsForAQuotaReset(t *testing.T) {
	old := *mcp.WaitPollInterval
	*mcp.WaitPollInterval = 10 * time.Millisecond
	defer func() { *mcp.WaitPollInterval = old }()

	o := newWaitOrch(domain.StatusWaiting)
	time.AfterFunc(60*time.Millisecond, func() { o.set(domain.StatusQueued, "") })
	time.AfterFunc(120*time.Millisecond, func() { o.set(domain.StatusCompleted, "after the reset") })
	msgs, _ := callWait(t, mcp.NewMcpServer(noEvents{o}, nil), map[string]any{"id": "t1"}, nil, "")
	if task := taskFrom(t, msgs[0]); task["status"] != "COMPLETED" || task["logs"] != "after the reset" {
		t.Errorf("it returned early: %v", task)
	}
}
