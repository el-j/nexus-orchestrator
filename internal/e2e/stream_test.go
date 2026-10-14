//go:build e2e

package e2e_test

import (
	"bufio"
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
