---
id: TASK-050
title: "Orchestrator + SQLite: model-aware dispatch + StatusNoProvider + schema migration"
role: backend
planId: PLAN-005
status: todo
dependencies: [TASK-044, TASK-049]
createdAt: 2026-03-10T06:00:00.000Z
---

## Context
The orchestrator must use `DiscoveryService.FindForModel()` when a task has a `ModelID` set,
and fall back to `DetectActive()` when it is empty.  When no provider serves the model,
the task must reach `StatusNoProvider`.  The SQLite schema needs two new columns for
`model_id` and `provider_hint` added to the `tasks` table.

## Files to Read
- `internal/core/services/orchestrator.go`
- `internal/adapters/outbound/repo_sqlite/repo.go`
- `internal/core/domain/task.go` (after TASK-044)

## Implementation Steps

### 1. SQLite schema migration (`internal/adapters/outbound/repo_sqlite/repo.go`)
Find the `CREATE TABLE IF NOT EXISTS tasks` statement and add two columns to it:
```sql
model_id       TEXT NOT NULL DEFAULT '',
provider_hint  TEXT NOT NULL DEFAULT '',
```

Also add `ALTER TABLE` migration guards (for databases that already exist without these columns):
After the table creation, add:
```go
for _, col := range []struct{ name, def string }{
    {"model_id", "TEXT NOT NULL DEFAULT ''"},
    {"provider_hint", "TEXT NOT NULL DEFAULT ''"},
} {
    _, _ = db.Exec(fmt.Sprintf(
        "ALTER TABLE tasks ADD COLUMN %s %s", col.name, col.def,
    ))
    // Ignore error — column already exists returns "duplicate column name"
}
```

Also update the `Save`, `GetByID`, and scan functions to include `model_id` and `provider_hint`:
- In `Save`: insert `model_id` and `provider_hint` values from task
- In row scanning: scan `model_id` and `provider_hint` into task fields

### 2. Orchestrator dispatch (`internal/core/services/orchestrator.go`)
In `processNext()`, replace the current `DetectActive()` call:

```go
llm, err := o.discovery.FindForModel(task.ModelID, task.ProviderHint)
if err != nil {
    log.Printf("orchestrator: no provider for task %s (model=%q): %v", task.ID, task.ModelID, err)
    _ = o.repo.UpdateLogs(task.ID, err.Error())
    _ = o.repo.UpdateStatus(task.ID, domain.StatusNoProvider)
    o.emit(task.ID, domain.StatusNoProvider)
    return
}
```

Remove the old `if llm == nil { re-queue }` block entirely.

**Rationale**: The old nil-check re-queued forever when no LLM was available.
`FindForModel` returns an error which maps cleanly to `StatusNoProvider`.
When the model field is empty, `FindForModel("","")` returns the first active provider,
preserving the existing no-model behavior.

**Note on the `DiscoveryService` field**: 
The current type is `*DiscoveryService`. The `FindForModel` method is added to it in TASK-049.
No interface change needed — the orchestrator uses the concrete type.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `tasks` table has `model_id` and `provider_hint` columns
- [ ] Migration is backward-compatible (ignores "duplicate column" error)
- [ ] Task with `ModelID` set → `FindForModel` is called with that modelID
- [ ] Task with no `ModelID` → `FindForModel("","")` → `DetectActive()` path
- [ ] No active provider → `StatusNoProvider` (not re-queued forever)
- [ ] Existing orchestrator tests pass without modification

## Anti-patterns to Avoid
- NEVER use raw string SQL in `Save()` without the new columns — missing columns = data loss
- NEVER remove the existing `errors.Is(err, domain.ErrNotFound)` check in the session loader
- NEVER panic on `ALTER TABLE` errors — they mean the column already exists
