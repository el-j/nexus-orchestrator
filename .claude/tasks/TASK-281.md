# TASK-281 — HTTP API: discovered agents + delegate + SSE session tasks

**Plan:** PLAN-044  
**Status:** TODO  
**Layer:** Go · inbound (`internal/adapters/inbound/httpapi/`)  
**Depends on:** TASK-280  

## Objective

Add three new HTTP endpoints. All follow existing server.go patterns (chi router, writeJSONError, context passing).

## Changes

### Route registration in `Handler()` in `server.go`

Add BEFORE the existing `r.Delete("/api/ai-sessions/{id}", ...)` to ensure literal segments take priority:
```go
r.Get("/api/ai-sessions/discovered",   s.handleGetDiscoveredAgents)
r.Post("/api/ai-sessions/{id}/delegate", s.handleDelegateToNexus)
```

The SSE upgrade for `r.Get("/api/ai-sessions/{id}/tasks", ...)` is an in-place modification.

### `handleGetDiscoveredAgents` (new file or append to existing ai_session handlers)

```go
func (s *Server) handleGetDiscoveredAgents(w http.ResponseWriter, r *http.Request) {
    agents, err := s.orch.GetDiscoveredAgents(r.Context())
    if err != nil {
        log.Printf("httpapi: get discovered agents: %v", err)
        // Return empty array, not 500 — scan errors are non-fatal
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode([]domain.DiscoveredAgent{})
        return
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(agents)
}
```

### `handleDelegateToNexus`

```go
func (s *Server) handleDelegateToNexus(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    instruction, err := s.orch.DelegateToNexus(r.Context(), id)
    if err != nil {
        if errors.Is(err, domain.ErrNotFound) {
            writeJSONError(w, "session not found", http.StatusNotFound)
            return
        }
        writeJSONError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]string{
        "instruction": instruction,
        "sessionId":   id,
    })
}
```

### `handleGetSessionTasks` — SSE upgrade

Check `r.Header.Get("Accept")`. If it contains `text/event-stream`, set SSE headers and subscribe to the Hub's session-scoped event channel (filter `event.SessionID == id`). Stream each `TaskEvent` as `data: <json>\n\n`. Fall back to existing JSON snapshot behaviour if no SSE header.

## API Shape

### `GET /api/ai-sessions/discovered` → 200
```json
[{"id":"claude-cli","kind":"claude-cli","name":"Claude CLI","detectionMethod":"fs-config","isRunning":false,"lastSeen":"..."}]
```

### `POST /api/ai-sessions/{id}/delegate` → 200
```json
{"instruction":"You are now operating under nexusOrchestrator...","sessionId":"abc123"}
```
→ 404 if session not found.

## Acceptance Criteria

- New routes present in router (verify via `go test ./internal/adapters/inbound/httpapi/...`)
- `GET /api/ai-sessions/discovered` returns 200 with `[]` when scanner is nil (graceful degradation)
- `POST /api/ai-sessions/nonexistent/delegate` returns 404
- Existing session task HTTP tests still pass
