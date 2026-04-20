---
id: TASK-558
title: Fix handleGetSessionTasks — replace O(n) GetAllTasks scan with GetTasksBySessionID
role: api
planId: PLAN-071
status: todo
dependencies: [TASK-555]
createdAt: 2026-04-15T00:00:00Z
---

## Context

`handleGetSessionTasks` in `handlers_tasks.go` calls `GetAllTasks()` and then filters the entire task list in Go memory to find tasks for a session. The `TaskRepository` port already has (or should have) a `GetTasksBySessionID(sessionID string)` method backed by a SQLite index. Using it avoids a full-table scan that grows with total task count.

## Files to Read

- `internal/adapters/inbound/httpapi/handlers_tasks.go`
- `internal/core/ports/ports.go`
- `internal/adapters/outbound/repo_sqlite/repo.go`
- `internal/adapters/outbound/repo_sqlite/task_repo.go` (if exists separately)

## Implementation Steps

1. Check whether `TaskRepository` in `ports.go` already has `GetTasksBySessionID(string) ([]domain.Task, error)` — if not, add it to the interface
2. Check whether the SQLite repo implements it — if not, add: `SELECT * FROM tasks WHERE claimedBySession = ? ORDER BY updatedAt DESC` and add to the concrete type
3. In `handlers_tasks.go:handleGetSessionTasks`: replace the `GetAllTasks()` + in-Go filter with a single call to `orch.GetTasksBySessionID(sessionID)` (add the method to the Orchestrator port if needed)
4. Add a test in `httpapi/server_test.go` for the new handler path: mock `GetTasksBySessionID`, assert only matching tasks are returned, assert non-existent session returns empty slice (not 404)

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `handleGetSessionTasks` never calls `GetAllTasks()` — calls a session-scoped repo method instead
- [ ] New test covers success path and empty-session case

## Anti-patterns to Avoid

- NEVER add goroutines inside `internal/core/services/`
- NEVER skip error wrapping with `fmt.Errorf("package: operation: %w", err)`
