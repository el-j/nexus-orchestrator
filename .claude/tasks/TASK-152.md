---
id: TASK-152
title: SQLite — AISession repository implementation + schema migration
role: backend
planId: PLAN-022
status: todo
dependencies: [TASK-151]
priority: high
estimated_effort: M
createdAt: 2026-03-12T11:00:00.000Z
---

## Goal
Implement `ports.AISessionRepository` in the `repo_sqlite` adapter, creating the `ai_sessions` table via the existing migration pattern and wiring it into `NewSessionRepo` or a parallel `NewAISessionRepo`.

## Context
`internal/adapters/outbound/repo_sqlite/` already has:
- `repo.go` — `Repository` struct, `*sql.DB`, auto-migration with `ensureSchema()`
- `session_repo.go` — `SessionRepo` struct sharing the same `*sql.DB` via `NewSessionRepo(r *Repository)`
- `provider_config_repo.go` — `ProviderConfigRepo` sharing the same `*sql.DB` via `NewProviderConfigRepo(r *Repository)`

The new `AISessionRepo` must follow the **identical pattern** as `ProviderConfigRepo`:
- Struct `AISessionRepo` with a single field `db *sql.DB`
- Constructor `NewAISessionRepo(r *Repository) *AISessionRepo` — shares `r.db`
- Table `ai_sessions` created in `repo.go`'s `ensureSchema()` — NOT in a separate migration file
- All methods use `context.Context` as first argument

## Scope

### Files to modify
- `internal/adapters/outbound/repo_sqlite/repo.go` — add `ai_sessions` CREATE TABLE to `ensureSchema()`

### Files to create
- `internal/adapters/outbound/repo_sqlite/ai_session_repo.go`
- `internal/adapters/outbound/repo_sqlite/ai_session_repo_test.go`

## Implementation Steps

### repo.go — add table DDL
Add to `ensureSchema()` (after the existing `provider_configs` CREATE TABLE statement):
```sql
CREATE TABLE IF NOT EXISTS ai_sessions (
    id           TEXT PRIMARY KEY,
    source       TEXT NOT NULL,
    external_id  TEXT NOT NULL DEFAULT '',
    agent_name   TEXT NOT NULL,
    project_path TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'active',
    last_activity DATETIME NOT NULL,
    routed_task_ids TEXT NOT NULL DEFAULT '[]',
    created_at   DATETIME NOT NULL,
    updated_at   DATETIME NOT NULL
);
```
`routed_task_ids` is stored as a JSON array string (e.g. `["uuid1","uuid2"]`).

### ai_session_repo.go — implement `ports.AISessionRepository`

Implement all 5 interface methods:

1. **`SaveAISession`** — `INSERT OR REPLACE INTO ai_sessions ...`. Marshal `RoutedTaskIDs` to JSON string before insert. Parse times as RFC3339.

2. **`GetAISessionByID`** — `SELECT * FROM ai_sessions WHERE id = ?`. Return `fmt.Errorf("repo_sqlite: get ai session: %w", domain.ErrNotFound)` when `sql.ErrNoRows`.

3. **`ListAISessions`** — `SELECT * FROM ai_sessions ORDER BY last_activity DESC`. Scan all rows, unmarshal `routed_task_ids` JSON column into `[]string`.

4. **`UpdateAISessionStatus`** — `UPDATE ai_sessions SET status=?, last_activity=?, updated_at=? WHERE id=?`.

5. **`DeleteAISession`** — `DELETE FROM ai_sessions WHERE id=?`. Error on `rows affected == 0` is acceptable (idempotent delete).

Error wrapping: `fmt.Errorf("repo_sqlite: <operation>: %w", err)`.

### ai_session_repo_test.go
- Use `newTestDB(t)` helper from the existing test file (or create it inline: `sql.Open("sqlite3", ":memory:")` + `ensureSchema()`)
- Test: `Save → GetByID` roundtrip; `ListAISessions` returns all; `UpdateAISessionStatus` changes status; `DeleteAISession` removes row
- Use `t.Parallel()` and `testify/assert` if already used in the package — otherwise use `testing` stdlib only

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./internal/adapters/outbound/repo_sqlite/...` exits 0
- [ ] `*AISessionRepo` satisfies `ports.AISessionRepository` (confirmed by compile-time interface check `var _ ports.AISessionRepository = (*AISessionRepo)(nil)`)
- [ ] `routed_task_ids` JSON roundtrips correctly through the DB layer
- [ ] Existing repo tests still pass

## Anti-patterns to Avoid
- NEVER import from `internal/core/services/` or any inbound adapter
- NEVER use `database/sql`'s `db.QueryRow` without checking `sql.ErrNoRows`
- NEVER store time as string without consistent format — use `time.RFC3339` throughout
- NEVER skip the compile-time interface satisfaction check `var _ ports.AISessionRepository = ...`
