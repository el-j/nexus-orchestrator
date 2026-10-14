package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

const (
	defaultWaitSeconds = 300
	maxWaitSeconds     = 3600
)

// waitPollInterval is how often wait_for_task re-reads the task. With an event feed this only
// catches events that were dropped; without one it is the only way to notice the end.
var waitPollInterval = 2 * time.Second

// progressFunc reports one progress message to the caller of a tool; nil when the caller did
// not ask for progress or the transport cannot carry it.
type progressFunc func(message string)

// toolWaitForTask blocks until the task has left the queue (it completed, failed, was cancelled
// and so on) or the timeout passes, and returns the task like get_task. While it waits it reports
// the reply as the model writes it through progress, if given. A timeout is not an error: the
// task is returned as it is, with timedOut set.
func (s *Server) toolWaitForTask(ctx context.Context, args json.RawMessage, progress progressFunc) (callToolResult, error) {
	var p struct {
		ID             string `json:"id"`
		TimeoutSeconds int    `json:"timeoutSeconds"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "invalid wait_for_task params"}
	}
	if p.ID == "" {
		return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "id is required"}
	}
	timeout := time.Duration(clampSeconds(p.TimeoutSeconds)) * time.Second

	// Subscribe before the first read so an event between the two cannot be lost.
	var events <-chan ports.TaskEvent
	if src, ok := s.orch.(ports.TaskEventSource); ok {
		var cancel func()
		events, cancel = src.SubscribeTaskEvents()
		defer cancel()
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	tick := time.NewTicker(waitPollInterval)
	defer tick.Stop()

	for {
		task, err := s.orch.GetTask(p.ID)
		if err != nil {
			return callToolResult{}, fmt.Errorf("mcp: wait_for_task: %w", err)
		}
		if task.Status != domain.StatusQueued && task.Status != domain.StatusProcessing {
			return waitResult(task, false)
		}
	wait:
		for {
			select {
			case <-ctx.Done():
				return callToolResult{}, ctx.Err()
			case <-deadline.C:
				if task, err = s.orch.GetTask(p.ID); err != nil {
					return callToolResult{}, fmt.Errorf("mcp: wait_for_task: %w", err)
				}
				return waitResult(task, true)
			case <-tick.C:
				break wait
			case ev, ok := <-events:
				if !ok {
					events = nil // the feed ended; the poll carries on alone
					continue
				}
				if ev.TaskID != p.ID {
					continue
				}
				switch ev.Type {
				case ports.EventTaskChunk:
					if progress != nil {
						progress(ev.Text)
					}
				case ports.EventTaskRestart:
					if progress != nil {
						progress("↻ new attempt")
					}
				default:
					break wait // a status change: look at the task
				}
			}
		}
	}
}

// clampSeconds applies the default for an unset (or negative) timeout and the upper bound.
func clampSeconds(n int) int {
	switch {
	case n <= 0:
		return defaultWaitSeconds
	case n > maxWaitSeconds:
		return maxWaitSeconds
	}
	return n
}

// waitResult is the task as get_task shows it, plus timedOut when the wait ran out.
func waitResult(task domain.Task, timedOut bool) (callToolResult, error) {
	b, err := json.Marshal(struct {
		domain.Task
		TimedOut bool `json:"timedOut,omitempty"`
	}{task, timedOut})
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: wait_for_task: marshal: %w", err)
	}
	return textResult(string(b)), nil
}

// ----- Progress over Streamable HTTP -----

// progressMeta is the "_meta" member of a tools/call request.
type progressMeta struct {
	ProgressToken json.RawMessage `json:"progressToken"`
}

// startProgress switches the response to a server-sent event stream when the caller asked for
// progress (a progressToken in _meta, and text/event-stream in Accept) and the connection can
// flush. It returns the progress function and the writer to send the final reply with, which
// frames it as one more event. Otherwise it returns nil and w unchanged: the reply is plain JSON.
func startProgress(w http.ResponseWriter, r *http.Request, meta progressMeta) (progressFunc, http.ResponseWriter) {
	flusher, ok := w.(http.Flusher)
	if len(meta.ProgressToken) == 0 || !ok || !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		return nil, w
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	fw := &eventWriter{ResponseWriter: w, flusher: flusher}
	n := 0
	return func(message string) {
		n++
		note, _ := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"method":  "notifications/progress",
			"params":  map[string]any{"progressToken": meta.ProgressToken, "progress": n, "message": message},
		})
		_, _ = fw.Write(note)
	}, fw
}

// eventWriter writes every Write call as one server-sent "message" event and flushes it.
type eventWriter struct {
	http.ResponseWriter
	flusher http.Flusher
}

// Write sends p (one JSON-RPC message) as an event.
func (e *eventWriter) Write(p []byte) (int, error) {
	if _, err := fmt.Fprintf(e.ResponseWriter, "event: message\ndata: %s\n\n", strings.TrimSpace(string(p))); err != nil {
		return 0, err
	}
	e.flusher.Flush()
	return len(p), nil
}

// ----- Schema -----

func waitToolDefs() []toolDef {
	return []toolDef{{
		Name: "wait_for_task",
		Description: "Block until a task has finished (completed, failed, cancelled, …) and return it like get_task. " +
			"Send a progressToken in _meta and Accept: text/event-stream to receive the model's reply as " +
			"notifications/progress while it is written. A timeout returns the task as it is with timedOut: true.",
		InputSchema: inputSchema{
			Type: "object",
			Properties: map[string]property{
				"id":             {Type: "string", Description: "Task ID returned by submit_task."},
				"timeoutSeconds": {Type: "integer", Description: "How long to wait (default 300, max 3600)."},
			},
			Required: []string{"id"},
		},
	}}
}
