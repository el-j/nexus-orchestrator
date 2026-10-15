//go:build e2e

package e2e_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

type liveEvent struct {
	Type   string `json:"type"`
	TaskID string `json:"taskId"`
	Text   string `json:"text"`
	Seq    int    `json:"seq"`
}

// subscribe opens GET /api/events and forwards the task events it reads.
func subscribe(t *testing.T, d *daemon) <-chan liveEvent {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, d.http+"/api/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	out := make(chan liveEvent, 256)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			data, ok := strings.CutPrefix(sc.Text(), "data: ")
			if !ok {
				continue
			}
			var ev liveEvent
			if json.Unmarshal([]byte(data), &ev) == nil && ev.TaskID != "" {
				out <- ev
			}
		}
		close(out)
	}()
	return out
}

// A subscriber sees the model's text while the model is still writing, and the finished task
// holds exactly that text.
func TestDaemonStreamsAReplyWhileTheModelIsStillWriting(t *testing.T) {
	llm := newFakeOllama(t)
	llm.hold = make(chan struct{})
	llm.reply = func(string) string { return "FIRSTHALF-SECONDHALF" }
	d := startDaemon(t, llm, daemonOpts{})
	d.waitHealthy()
	events := subscribe(t, d)

	var created struct {
		TaskID string `json:"task_id"`
	}
	if code := d.do(http.MethodPost, "/api/tasks", map[string]any{
		"projectPath": t.TempDir(), "targetFile": "out.txt", "instruction": "stream it", "command": "plan",
	}, &created); code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("submit: HTTP %d\n%s", code, d.logs())
	}

	var live strings.Builder
	timeout := time.After(15 * time.Second)
	for !strings.Contains(live.String(), "FIRSTHALF") {
		select {
		case ev := <-events:
			if ev.TaskID == created.TaskID && ev.Type == "task.chunk" {
				live.WriteString(ev.Text)
			}
		case <-timeout:
			t.Fatalf("no live text before the model finished; got %q\n%s", live.String(), d.logs())
		}
	}
	// The model is blocked after its first piece, so the task cannot be finished yet.
	var mid apiTask
	d.do(http.MethodGet, "/api/tasks/"+created.TaskID, nil, &mid)
	if mid.Status == "COMPLETED" {
		t.Fatalf("task completed while the model was held: %+v", mid)
	}
	close(llm.hold)

	d.waitStatus(created.TaskID, "COMPLETED")
	for !strings.Contains(live.String(), "SECONDHALF") {
		select {
		case ev := <-events:
			if ev.TaskID == created.TaskID && ev.Type == "task.chunk" {
				live.WriteString(ev.Text)
			}
		case <-timeout:
			t.Fatalf("second half never streamed; got %q", live.String())
		}
	}
	if live.String() != "FIRSTHALF-SECONDHALF" {
		t.Errorf("streamed text = %q", live.String())
	}
}

// An MCP client that waits for a task sees the model's text as progress notifications while the
// model is still writing, and the call ends with the finished task.
func TestDaemonWaitForTaskStreamsProgressOverMCP(t *testing.T) {
	llm := newFakeOllama(t)
	llm.hold = make(chan struct{})
	llm.reply = func(string) string { return "FIRSTHALF-SECONDHALF" }
	d := startDaemon(t, llm, daemonOpts{})
	d.waitHealthy()

	var created struct {
		TaskID string `json:"task_id"`
	}
	if code := d.do(http.MethodPost, "/api/tasks", map[string]any{
		"projectPath": t.TempDir(), "targetFile": "out.txt", "instruction": "wait for me", "command": "plan",
	}, &created); code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("submit: HTTP %d\n%s", code, d.logs())
	}

	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 5, "method": "tools/call", "params": map[string]any{
		"name": "wait_for_task", "arguments": map[string]any{"id": created.TaskID, "timeoutSeconds": 60},
		"_meta": map[string]any{"progressToken": "e2e"},
	}})
	req, _ := http.NewRequest(http.MethodPost, d.mcp+"/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("wait_for_task: %v\n%s", err, d.logs())
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content type = %q", ct)
	}

	var progress strings.Builder
	var final map[string]any
	released := false
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 1<<16), 1<<20)
	deadline := time.AfterFunc(30*time.Second, func() { resp.Body.Close() })
	defer deadline.Stop()
	for sc.Scan() {
		data, ok := strings.CutPrefix(sc.Text(), "data: ")
		if !ok {
			continue
		}
		var msg map[string]any
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			t.Fatalf("bad event %q", data)
		}
		if msg["method"] == "notifications/progress" {
			progress.WriteString(msg["params"].(map[string]any)["message"].(string))
			if !released && strings.Contains(progress.String(), "FIRSTHALF") {
				// The model is blocked after its first piece: this text arrived mid-generation.
				released = true
				close(llm.hold)
			}
			continue
		}
		final = msg
		break
	}
	if !released {
		t.Fatalf("no progress arrived before the model finished; got %q\n%s", progress.String(), d.logs())
	}
	text := final["result"].(map[string]any)["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, `"status":"COMPLETED"`) {
		t.Errorf("final result = %s", text)
	}
	if !strings.Contains(progress.String(), "SECONDHALF") {
		t.Errorf("progress text = %q", progress.String())
	}
}
