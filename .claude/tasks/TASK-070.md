---
id: TASK-070
title: SSE event lifecycle test (submit → events → verify order)
role: qa
planId: PLAN-008
status: todo
dependencies: []
createdAt: 2026-03-10T14:00:00.000Z
---

## Context
The existing SSE test only verifies that GET /api/events returns a 200 with `text/event-stream` content type. It doesn't verify that **actual task lifecycle events** are received. We need a test that submits a task, listens on SSE, and verifies the correct events arrive in order: `task.queued` → `task.processing` → `task.completed`.

## Files to Read
- `internal/core/services/integration_test.go` (testStack + existing SSE test)
- `internal/adapters/inbound/httpapi/hub.go` (SSE broadcasting)
- `internal/core/ports/ports.go` (EventType constants, TaskEvent)
- `internal/core/services/orchestrator.go` (emit() calls)

## Implementation Steps

1. Add `TestSSEEventLifecycle` to `internal/core/services/integration_test.go`.
2. Start the testStack httptest.Server as usual.
3. Open a goroutine that connects to GET /api/events and reads SSE `data:` lines into a channel. Parse each line as JSON `TaskEvent`.
4. Wait briefly (100ms) for the SSE connection to be established (read the initial `{"type":"connected"}` ping).
5. Submit a task via POST /api/tasks.
6. Collect events from the channel for up to 15s.
7. Verify the collected events include, in order:
   - `task.queued` with the submitted task ID
   - `task.processing` with the submitted task ID
   - `task.completed` with the submitted task ID
8. Close the SSE connection.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `TestSSEEventLifecycle` verifies at least `task.queued`, `task.processing`, `task.completed` events
- [ ] Events are verified in correct chronological order
- [ ] Test does not flake — uses proper synchronization, not just sleep

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
