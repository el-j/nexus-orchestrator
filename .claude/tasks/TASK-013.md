---
id: TASK-013
title: Orchestrator startup recovery + Stop() idempotency
role: backend
planId: PLAN-002
status: todo
dependencies: []
createdAt: 2026-03-09T12:00:00.000Z
---

## Context

Two CRITICAL failsafe issues make the orchestrator lose work on every daemon restart:
- **A1 (CRITICAL):** Tasks in `PROCESSING` state when the daemon crashes are orphaned — they stay `PROCESSING` in the DB forever and are never retried.
- **A2 (CRITICAL):** Tasks in `QUEUED` state in the DB are NOT loaded into the in-memory queue on startup — they are silently lost until someone re-submits them.
- **E1 (HIGH):** `Stop()` calls `close(o.stopCh)` unconditionally — a second call panics with "close of closed channel".
- **D1 (HIGH):** All `repo.UpdateStatus()` calls use `_ =` — DB write failures are silently swallowed.

## Files to Read

- `internal/core/services/orchestrator.go` — full file
- `internal/core/ports/ports.go` — TaskRepository interface (check if GetPending exists)
- `internal/adapters/outbound/repo_sqlite/repo.go` — SQLite implementation

## Implementation Steps

1. **Add `GetPending() ([]domain.Task, error)` to `TaskRepository` port** in `internal/core/ports/ports.go`.
   - Returns all tasks with `StatusQueued` OR `StatusProcessing`.

2. **Implement `GetPending()` in `repo_sqlite/repo.go`**:
   ```go
   func (r *Repository) GetPending() ([]domain.Task, error) {
       rows, err := r.db.Query(
           `SELECT id, project_path, prompt, status, created_at, updated_at, retry_count,
                   source_project_path, source_task_id, source_plan_id
            FROM tasks WHERE status IN (?,?)`,
           string(domain.StatusQueued), string(domain.StatusProcessing),
       )
       // scan + return
   }
   ```
   Note: `retry_count`, `source_project_path`, `source_task_id`, `source_plan_id` columns are added by TASK-014 and TASK-026 respectively — add them to the schema migration so they exist with defaults when this task runs.

3. **In `NewOrchestrator()` / startup**, after creating the service struct call `repo.GetPending()`:
   - For each task with `StatusProcessing`: call `repo.UpdateStatus(task.ID, StatusQueued)` first, then enqueue.
   - For each task with `StatusQueued`: enqueue directly (do NOT reset status).
   - Log recovered task IDs with `log.Printf("orchestrator: recovered %d pending tasks", count)`.
   - If `GetPending()` returns an error, return the error from `NewOrchestrator` (makes the daemon fail-fast rather than silently lose data).
   - `NewOrchestrator()` must return `(*OrchestratorService, error)` — update all callers in `main.go`, `cmd/nexus-daemon/main.go`.

4. **Make `Stop()` idempotent using `sync.Once`**:
   ```go
   type OrchestratorService struct {
       // ... existing fields ...
       stopOnce sync.Once
   }

   func (o *OrchestratorService) Stop() {
       o.stopOnce.Do(func() { close(o.stopCh) })
   }
   ```

5. **Remove all `_ = o.repo.UpdateStatus(...)` patterns** — replace with proper error logging:
   ```go
   if err := o.repo.UpdateStatus(task.ID, domain.StatusProcessing); err != nil {
       log.Printf("orchestrator: update status processing: %v", err)
       // re-enqueue the task so it is not lost
       o.mu.Lock()
       o.queue = append([]domain.Task{task}, o.queue...)
       o.mu.Unlock()
       return
   }
   ```

6. **Make the worker context-aware** — pass a `context.Context` through to LLM calls so shutdown can interrupt in-flight I/O:
   - Change `processNext()` to accept a `ctx context.Context`.
   - Thread context through `LLMClient.Chat(ctx, messages)` and `LLMClient.GenerateCode(ctx, prompt)`.
   - Update port signatures: `Chat(ctx context.Context, messages []domain.Message) (string, error)`.
   - Update LMStudio + Ollama adapters to use `http.NewRequestWithContext(ctx, ...)`.
   - In `run()`, pass `ctx` derived from the stop channel: use `context.WithCancel`.

7. **Update the in-memory stubs** in `orchestrator_test.go` to implement the new `GetPending() ([]domain.Task, error)` method.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go build .` (Wails binary) exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] New test `TestOrchestratorStartupRecovery`: seed DB with 2 QUEUED + 1 PROCESSING tasks, create new OrchestratorService, verify all 3 appear in queue with StatusQueued
- [ ] New test `TestStopIdempotent`: call `Stop()` twice — no panic
- [ ] `Stop()` uses `sync.Once` — confirmed by code review
- [ ] No `_ = o.repo.UpdateStatus(...)` pattern remains anywhere in orchestrator.go

## Anti-patterns to Avoid

- NEVER import adapters from core services
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("orchestrator: operation: %w", err)` error wrapping
- NEVER silently swallow `UpdateStatus` errors — log AND re-enqueue to avoid data loss
