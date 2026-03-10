---
id: TASK-062
title: "Data integrity: SQLite PRAGMAs + indexes + RowsAffected + path normalization"
role: backend
planId: PLAN-007
status: todo
dependencies: []
createdAt: 2026-03-10T10:00:00.000Z
---

## Context
SQLite repo has no WAL mode, no busy timeout, no foreign key enforcement, no connection pool config, and no indexes on frequently queried columns. `UpdateStatus`/`UpdateLogs` don't verify rows were affected. Project path is not normalized in `repo.Save()` but is in `SessionRepo`.

## Files to Read
- `internal/adapters/outbound/repo_sqlite/repo.go`
- `internal/adapters/outbound/repo_sqlite/session_repo.go`
- `internal/adapters/outbound/repo_sqlite/repo_test.go`

## Implementation Steps
1. In `New()`, after `sql.Open`, execute PRAGMAs: `journal_mode=WAL`, `busy_timeout=5000`, `foreign_keys=ON`.
2. Set `db.SetMaxOpenConns(1)` for SQLite write serialization and `db.SetMaxIdleConns(1)`.
3. In `migrate()`, add `CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)` and `CREATE INDEX IF NOT EXISTS idx_tasks_project_path ON tasks(project_path)`.
4. In `UpdateStatus` and `UpdateLogs`, check `result.RowsAffected()` — if 0, return `domain.ErrNotFound` wrapped with context.
5. In `Save()`, normalize `t.ProjectPath` with `filepath.Clean()` before inserting, consistent with `SessionRepo`.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] WAL mode enabled (query `PRAGMA journal_mode` returns `wal`)
- [ ] `UpdateStatus` on non-existent ID returns error wrapping `domain.ErrNotFound`
- [ ] Indexes exist on `tasks.status` and `tasks.project_path`

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
