---
id: TASK-063
title: "Concurrency: stop swallowing repo errors + CancelTask sentinel + post-Stop guard"
role: backend
planId: PLAN-007
status: todo
dependencies: []
createdAt: 2026-03-10T10:00:00.000Z
---

## Context
`processNext()` discards all repo errors with `_ =`, silently diverging DB from memory. `CancelTask()` uses bare `errors.New()` instead of `domain.ErrNotFound`. After `Stop()`, `SubmitTask()` silently accepts and drops tasks.

## Files to Read
- `internal/core/services/orchestrator.go`
- `internal/core/services/orchestrator_test.go`
- `internal/core/domain/task.go`
- `internal/core/ports/ports.go`

## Implementation Steps
1. In `processNext()`, replace all `_ = o.repo.UpdateStatus(...)` and `_ = o.repo.UpdateLogs(...)` calls with error logging via `log.Printf("orchestrator: update status/logs for task %s: %v", task.ID, err)` when the error is non-nil. Do NOT stop processing for log failures — but DO log them.
2. In `CancelTask()`, change `errors.New("orchestrator: task not found in queue")` to `fmt.Errorf("orchestrator: cancel task: %w", domain.ErrNotFound)` so callers can use `errors.Is()`.
3. Add a `stopped bool` field to `OrchestratorService`. Set it in `Stop()` under lock. In `SubmitTask()`, check `o.stopped` under lock and return a descriptive error if true.
4. Make `Stop()` idempotent using `sync.Once`.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `errors.Is(svc.CancelTask("nonexistent"), domain.ErrNotFound)` returns true
- [ ] `svc.Stop(); _, err := svc.SubmitTask(task)` returns non-nil error
- [ ] Repo update failures in processNext are logged, not silently discarded

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
