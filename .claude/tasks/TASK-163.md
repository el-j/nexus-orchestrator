---
id: TASK-163
title: SSE log capture backend for in-app log console
role: backend
planId: PLAN-023
status: todo
dependencies: [TASK-159]
createdAt: 2026-03-11T21:00:00.000Z
---

## Context
The user wants the daemon/desktop app logs to appear inside the app window instead of a separate console. We need to intercept Go's `log.Printf` output and broadcast log entries as SSE events on the existing `/api/events` endpoint. This replaces the need for a visible console window.

## Files to Read
- `internal/core/domain/log_entry.go` — `LogEntry`, `LogLevel` from TASK-159
- `internal/adapters/inbound/httpapi/server.go` — existing SSE hub (`/api/events`, `TaskEvent`)
- `main.go` — where `log.Printf` is used for operational logging

## Implementation Steps

1. Create `internal/adapters/inbound/httpapi/log_hub.go` with a `LogHub` that:
   - Implements `io.Writer` (so it can be set as `log.SetOutput(hub)`)
   - Parses each `Write(p []byte)` call into a `domain.LogEntry` (timestamp from log prefix, best-effort level detection from prefixed keywords like `ERROR:`, `WARN:`, default=info)
   - Maintains a ring buffer of last 500 entries
   - Broadcasts each new entry to connected SSE clients as event type `log`
   - Also writes to `os.Stderr` so terminal output is not lost in daemon mode

2. Extend the existing SSE hub in `server.go` (or create a shared event bus) to support a new event type `"log"` alongside existing `"task_*"` events.

3. Add `GET /api/logs` endpoint that returns the ring buffer contents as `[]LogEntry` JSON (for initial load when frontend connects).

4. In `main.go` and `cmd/nexus-daemon/main.go`, set up the `LogHub` early in startup:
   ```go
   logHub := httpapi.NewLogHub()
   log.SetOutput(logHub)
   ```
   Pass `logHub` into the HTTP server so it can register SSE listeners.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `LogHub` implements `io.Writer`
- [ ] Log entries broadcast as SSE event type `"log"` on `/api/events`
- [ ] `GET /api/logs` returns buffered log entries
- [ ] Original stderr output is preserved (tee pattern)

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER drop log output — always tee to stderr alongside the SSE broadcast
