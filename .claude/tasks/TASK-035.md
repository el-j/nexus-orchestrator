---
id: TASK-035
title: SSE events endpoint GET /api/events + OrchestratorService event fan-out
role: backend
planId: PLAN-003
status: todo
dependencies: []
createdAt: 2026-03-09T15:00:00.000Z
---

## Context

The embedded web dashboard (TASK-034) currently polls every 2 seconds. This task adds a Server-Sent Events (SSE) endpoint at `GET /api/events` that pushes task lifecycle events to clients in real time, enabling the dashboard to switch from polling to event-driven updates.

## Architecture

To respect hexagonal architecture, the event bus is injected into the orchestrator as an optional port:

```
OrchestratorService ──broadcast──▶ EventBroadcaster (port)
                                           │
                                    httpapi.Hub (adapter)
                                           │
                              HTTP SSE clients (browsers)
```

## Files to Create / Modify

- `internal/core/ports/ports.go` — add `EventBroadcaster` interface + `TaskEvent` struct
- `internal/core/services/orchestrator.go` — inject optional `EventBroadcaster`, call `Broadcast` after status changes
- `internal/adapters/inbound/httpapi/hub.go` — SSE Hub implementation
- `internal/adapters/inbound/httpapi/server.go` — wire Hub, add `GET /api/events` route
- `main.go` + `cmd/nexus-daemon/main.go` — pass Hub to orchestrator

## New Port (ports.go)

```go
// TaskEvent represents a single task lifecycle change notification.
type TaskEvent struct {
    Type   string            `json:"type"`   // "task.queued", "task.processing", "task.completed", "task.failed"
    TaskID string            `json:"taskId"`
    Status domain.TaskStatus `json:"status"`
}

// EventBroadcaster is the outbound port for publishing task lifecycle events.
// Implementations must be safe for concurrent use.
type EventBroadcaster interface {
    Broadcast(event TaskEvent)
}
```

## Hub Implementation (httpapi/hub.go)

```go
package httpapi

import (
    "encoding/json"
    "fmt"
    "net/http"
    "sync"

    "nexus-orchestrator/internal/core/ports"
)

// Hub manages SSE connections and broadcasts events to all subscribers.
type Hub struct {
    mu      sync.RWMutex
    clients map[chan []byte]struct{}
}

func NewHub() *Hub {
    return &Hub{clients: make(map[chan []byte]struct{})}
}

func (h *Hub) Broadcast(event ports.TaskEvent) {
    data, err := json.Marshal(event)
    if err != nil {
        return
    }
    msg := []byte(fmt.Sprintf("data: %s\n\n", data))
    h.mu.RLock()
    defer h.mu.RUnlock()
    for ch := range h.clients {
        select {
        case ch <- msg:
        default: // slow client — skip this event
        }
    }
}

func (h *Hub) ServeSSE(w http.ResponseWriter, r *http.Request) {
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "SSE not supported", http.StatusInternalServerError)
        return
    }

    ch := make(chan []byte, 16)
    h.mu.Lock()
    h.clients[ch] = struct{}{}
    h.mu.Unlock()
    defer func() {
        h.mu.Lock()
        delete(h.clients, ch)
        close(ch)
        h.mu.Unlock()
    }()

    w.Header().Set("Content-Type",  "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection",    "keep-alive")
    // Initial ping so client knows connection is alive
    fmt.Fprint(w, "data: {\"type\":\"connected\"}\n\n")
    flusher.Flush()

    for {
        select {
        case msg, ok := <-ch:
            if !ok {
                return
            }
            w.Write(msg)
            flusher.Flush()
        case <-r.Context().Done():
            return
        }
    }
}
```

## OrchestratorService Changes

Add field and broadcast calls:
```go
type OrchestratorService struct {
    // ... existing fields ...
    broadcaster ports.EventBroadcaster // optional, nil = no events
}
```

In `processNext()`, after each `repo.UpdateStatus(...)`:
```go
if o.broadcaster != nil {
    o.broadcaster.Broadcast(ports.TaskEvent{
        Type:   "task." + strings.ToLower(string(status)),
        TaskID: task.ID,
        Status: status,
    })
}
```

## Dashboard Update (TASK-034 integration)

If SSE is available, dashboard switches from polling to EventSource:
```javascript
// Attempt SSE; fall back to polling on error
const es = new EventSource('/api/events')
es.onmessage = (e) => {
    const evt = JSON.parse(e.data)
    if (evt.type !== 'connected') refresh()
}
es.onerror = () => es.close() // polling continues anyway
```

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `ports.EventBroadcaster` interface defined
- [ ] `httpapi.Hub` implements `ports.EventBroadcaster`
- [ ] `GET /api/events` returns `Content-Type: text/event-stream`
- [ ] Connecting to `/api/events` and submitting a task via `POST /api/tasks` produces an event
- [ ] Multiple concurrent SSE clients supported without data race (`-race` passes)
- [ ] Slow clients do not block event delivery to fast clients (buffered channel + `select default`)
- [ ] `OrchestratorService` works correctly when `broadcaster == nil` (existing tests unchanged)
- [ ] Wiring in `main.go` and `cmd/nexus-daemon/main.go` passes hub to orchestrator

## Anti-patterns to Avoid

- NEVER close the Hub's client channel from the Broadcast goroutine — only from ServeSSE cleanup
- NEVER use a global Hub — inject via dependency
- NEVER make EventBroadcaster required — nil check, backward compat required
- NEVER block Broadcast waiting for slow clients — use buffered channel + select with default
- NEVER import httpapi from services — only inject the port interface
