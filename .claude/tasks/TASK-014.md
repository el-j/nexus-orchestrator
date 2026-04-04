---
id: TASK-014
title: Retry limits + exponential backoff + ProjectPath normalization + queue cap
role: backend
planId: PLAN-002
status: done
completedAt: 2026-03-14T18:30:00.000Z
dependencies: []
createdAt: 2026-03-09T12:00:00.000Z
---

## Context

Four MEDIUM/HIGH failsafe issues degrade reliability under load or LLM unavailability:

- **D2 (HIGH):** When the LLM is unavailable, the task is immediately re-queued and retried every 2 seconds forever — infinite tight loop, no backoff.
- **B3 (MEDIUM):** The in-memory queue has no size cap — malicious or buggy clients can exhaust memory via HTTP/MCP.
- **B5/F1 (MEDIUM):** `task.ProjectPath` is stored and used as a session key without `filepath.Clean()` — `/proj/a` and `/proj/a/` resolve to different sessions.
- **C1 (MEDIUM):** In `SubmitTask`, `repo.Save(task)` is called outside `o.mu.Lock()`, creating a window where the task exists in the DB but not yet in the queue.

## Files to Read

- `internal/core/services/orchestrator.go` — full file
- `internal/core/domain/task.go` — Task struct
- `internal/adapters/outbound/repo_sqlite/repo.go` — schema migration

## Implementation Steps

1. **Add `RetryCount int` to `domain.Task`** in `internal/core/domain/task.go`.
   - Default value 0. No other domain changes needed.

2. **Add SQLite column `retry_count INTEGER NOT NULL DEFAULT 0`** to the tasks table migration in `repo_sqlite/repo.go`:
   - Add to `CREATE TABLE IF NOT EXISTS tasks` DDL.
   - Add `ALTER TABLE tasks ADD COLUMN retry_count INTEGER NOT NULL DEFAULT 0` as a safe migration (wrapped in `PRAGMA table_info` check or just as idempotent DDL).
   - Update all `INSERT`, `SELECT`, and `UPDATE` queries to include `retry_count`.

3. **Implement retry limit with exponential backoff** in `processNext()`:

   ```go
   const maxRetries = 5

   // after LLM error:
   task.RetryCount++
   if task.RetryCount >= maxRetries {
       log.Printf("orchestrator: task %s exceeded max retries (%d), marking failed", task.ID, maxRetries)
       if err := o.repo.UpdateStatus(task.ID, domain.StatusFailed); err != nil {
           log.Printf("orchestrator: mark failed: %v", err)
       }
       return
   }
   // backoff: 2^retryCount seconds, capped at 120s
   backoff := time.Duration(1<<uint(task.RetryCount)) * time.Second
   if backoff > 120*time.Second {
       backoff = 120 * time.Second
   }
   log.Printf("orchestrator: task %s retry %d/%d in %s", task.ID, task.RetryCount, maxRetries, backoff)
   time.Sleep(backoff)   // this runs inside the worker goroutine — acceptable
   o.mu.Lock()
   o.queue = append(o.queue, task)
   o.mu.Unlock()
   ```

   Update `repo.Save`/`UpdateStatus` (or add `repo.UpdateRetryCount`) so `retry_count` is persisted before re-queuing.

4. **Add queue size cap** controlled by env var `NEXUS_MAX_QUEUE` (default 500):

   ```go
   func maxQueueSize() int {
       if s := os.Getenv("NEXUS_MAX_QUEUE"); s != "" {
           if n, err := strconv.Atoi(s); err == nil && n > 0 {
               return n
           }
       }
       return 500
   }
   ```

   In `SubmitTask`, after acquiring `o.mu.Lock()` check `len(o.queue) >= maxQueueSize()` and return `fmt.Errorf("orchestrator: submit task: queue at capacity (%d)", maxQueueSize())`.

5. **Normalize ProjectPath** in `SubmitTask` using `filepath.Clean`:

   ```go
   task.ProjectPath = filepath.Clean(task.ProjectPath)
   ```

   Add this line immediately after the task struct is populated, before `repo.Save`.

6. **Fix the race window in `SubmitTask`** — move `repo.Save` inside the mutex:
   ```go
   o.mu.Lock()
   defer o.mu.Unlock()
   if len(o.queue) >= maxQueueSize() {
       return "", fmt.Errorf("orchestrator: submit task: queue at capacity")
   }
   task.ProjectPath = filepath.Clean(task.ProjectPath)
   if err := o.repo.Save(task); err != nil {
       return "", fmt.Errorf("orchestrator: submit task: %w", err)
   }
   o.queue = append(o.queue, task)
   return task.ID, nil
   ```
   Note: `repo.Save` holding the mutex is acceptable because SQLite write latency is <1ms for local file DB.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] New test `TestRetryLimitExceeded`: simulate LLM always failing → after maxRetries the task status becomes `StatusFailed`
- [ ] New test `TestQueueAtCapacity`: fill queue to cap, next SubmitTask returns error
- [ ] New test `TestProjectPathNormalization`: submit with trailing slash, verify stored path has no trailing slash and matches session key
- [ ] `filepath.Clean` called on ProjectPath in SubmitTask — confirmed by code review
- [ ] `retry_count` column present in SQLite schema after migration

## Anti-patterns to Avoid

- NEVER hold a mutex across an HTTP call or blocking I/O — the `repo.Save` inside mutex is OK only because it is a local SQLite write
- NEVER import adapters from core services
- NEVER hard-code retry constants outside a named `const` block
- NEVER use `time.Sleep` in tests — use a fake clock or mock that returns errors immediately
