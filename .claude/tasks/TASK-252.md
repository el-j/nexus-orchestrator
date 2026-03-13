---
id: TASK-252
title: "Backend: SQLite migration + repo methods for task-session persistence"
role: backend
planId: PLAN-038
status: todo
dependencies: [TASK-251]
createdAt: 2026-03-13T16:00:00.000Z
---

## Context
The SQLite tasks table has no `ai_session_id` column, and the AISession repo has no method to append routed task IDs. This task adds the persistence layer so the service can store task-session bindings.

## Files to Read
- `internal/adapters/outbound/repo_sqlite/repo.go` (schema / migrations)
- `internal/adapters/outbound/repo_sqlite/ai_session_repo.go`
- `internal/adapters/outbound/repo_sqlite/repo_test.go`
- `internal/adapters/outbound/repo_sqlite/ai_session_repo_test.go`

## Implementation Steps
1. In `repo.go` `initSchema()`, add `ai_session_id TEXT DEFAULT ''` column to the tasks table (use ALTER TABLE IF NOT EXISTS pattern or add to CREATE TABLE).
2. Update `Save()` and `GetByID()` / `GetAll()` / `GetPending()` / `ClaimNextQueued()` to include `ai_session_id` in INSERT/SELECT.
3. Implement `GetTasksBySessionID(sessionID string) ([]domain.Task, error)` — SELECT * FROM tasks WHERE ai_session_id = ? ORDER BY created_at DESC.
4. Implement `AppendRoutedTaskID(ctx, sessionID, taskID)` on `AISessionRepo` — read current `routed_task_ids` JSON, append taskID, write back.
5. Add unit tests for `GetTasksBySessionID` and `AppendRoutedTaskID`.

## Acceptance Criteria
- [ ] `ai_session_id` column exists in tasks table
- [ ] All task CRUD operations read/write `AISessionID`
- [ ] `GetTasksBySessionID` returns correct filtered results
- [ ] `AppendRoutedTaskID` appends without duplicates
- [ ] `CGO_ENABLED=1 go test -race ./internal/adapters/outbound/repo_sqlite/...` passes
