---
id: TASK-174
title: SQLite — add columns and repo methods for backlog fields
role: backend
planId: PLAN-024
status: todo
dependencies: [TASK-173]
createdAt: 2026-03-11T22:00:00.000Z
---

## Context
The new Task fields (`ProviderName`, `Priority`, `Tags`) need SQLite columns, and the new `TaskRepository` methods (`GetByProjectPathAndStatus`, `Update`) need implementations.

## Files to Read
- `internal/adapters/outbound/repo_sqlite/repo.go` — schema migration, scanTask, Save, existing queries
- `internal/core/domain/task.go` — Task struct with new fields
- `internal/core/ports/ports.go` — updated TaskRepository interface from TASK-173

## Implementation Steps

1. Add additive column migrations in `migrate()`:
   ```go
   {"provider_name", "TEXT NOT NULL DEFAULT ''"},
   {"priority", "INTEGER NOT NULL DEFAULT 2"},
   {"tags", "TEXT NOT NULL DEFAULT '[]'"},
   ```

2. Update `scanTask()` to scan the 3 new columns. Tags are stored as JSON array string.

3. Update `Save()` to persist `ProviderName`, `Priority`, `Tags` (marshal Tags to JSON).

4. Implement `GetByProjectPathAndStatus(projectPath string, statuses ...domain.TaskStatus) ([]domain.Task, error)`:
   - Build SQL: `SELECT ... FROM tasks WHERE project_path = ? AND status IN (?, ?, ...)`
   - Use variadic statuses to build the IN clause dynamically
   - Order by `priority ASC, created_at ASC`

5. Implement `Update(t domain.Task) error`:
   - Update mutable fields: `instruction`, `target_file`, `context_files`, `status`, `provider_name`, `provider_hint`, `model_id`, `priority`, `tags`, `command`, `updated_at`
   - WHERE `id = ?`
   - Return `domain.ErrNotFound` if no rows affected

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `ALTER TABLE tasks ADD COLUMN provider_name/priority/tags` runs on existing DB
- [ ] `GetByProjectPathAndStatus` filters correctly by project + statuses
- [ ] `Update` persists all mutable fields and returns `ErrNotFound` for missing IDs
- [ ] Tags are stored as JSON array and deserialized correctly

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER skip `fmt.Errorf("sqlite: operation: %w", err)` error wrapping
- NEVER use raw string formatting for SQL parameters — use `?` placeholders
