---
id: TASK-026
title: Domain — add source writeback fields to Task
role: architecture
planId: PLAN-002
status: todo
dependencies: []
createdAt: 2026-03-09T13:00:00.000Z
---

## Context

The writeback system (TASK-027, TASK-028) needs to know WHERE a task came from: which external project submitted it, what the source task ID was in that project's `.claude/orchestrator.json`, and what plan ID it belonged to. These three optional fields must be added to the `domain.Task` struct so they can be persisted, passed through the HTTP/MCP API, and used by the `fs_writeback` adapter on completion.

Without these fields, the orchestrator cannot know where to write results back when a task is done.

## Files to Read

- `internal/core/domain/task.go` — current Task struct + TaskStatus consts
- `internal/adapters/outbound/repo_sqlite/repo.go` — SQL schema + all queries (to plan migration)
- `internal/core/ports/ports.go` — TaskRepository interface

## Implementation Steps

1. **Add three optional fields to `domain.Task`** in `internal/core/domain/task.go`:
   ```go
   type Task struct {
       ID          string
       ProjectPath string
       Prompt      string
       Status      TaskStatus
       CreatedAt   time.Time
       UpdatedAt   time.Time
       RetryCount  int          // added in TASK-014

       // Writeback: set when task was submitted by an external project via push-to-nexus
       SourceProjectPath string `json:"sourceProjectPath,omitempty"`
       SourceTaskID      string `json:"sourceTaskId,omitempty"`
       SourcePlanID      string `json:"sourcePlanId,omitempty"`
   }
   ```
   All three fields are optional (zero value = empty string = "not a writeback task").

2. **Update SQLite schema** in `repo_sqlite/repo.go`:
   - Add three columns to `CREATE TABLE IF NOT EXISTS tasks`:
     ```sql
     source_project_path TEXT NOT NULL DEFAULT '',
     source_task_id      TEXT NOT NULL DEFAULT '',
     source_plan_id      TEXT NOT NULL DEFAULT ''
     ```
   - Add safe `ALTER TABLE` migrations (run after schema creation, idempotent via `PRAGMA table_info` check or `IF NOT EXISTS`-style):
     ```sql
     ALTER TABLE tasks ADD COLUMN source_project_path TEXT NOT NULL DEFAULT '';
     ALTER TABLE tasks ADD COLUMN source_task_id TEXT NOT NULL DEFAULT '';
     ALTER TABLE tasks ADD COLUMN source_plan_id TEXT NOT NULL DEFAULT '';
     ```
     Note: SQLite `ALTER TABLE ADD COLUMN` ignores errors if wrapped in a transaction-safe helper; alternatively check `PRAGMA table_info(tasks)` first.
   - Update all `INSERT` statements to include these columns.
   - Update all `SELECT` scan calls to scan these columns into `Task` fields.
   - No changes needed to `UPDATE` queries (these fields are immutable after creation).

3. **Update `repo_sqlite/repo.go` GetPending / GetAll / GetBySourceProject** (added in TASK-013/015) to scan the new fields.

4. **JSON serialisation note**: the `omitempty` tag means these fields are omitted from JSON when empty — this is correct behaviour for backward compatibility with existing API clients.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `domain.Task` has `SourceProjectPath`, `SourceTaskID`, `SourcePlanID` fields
- [ ] SQLite schema includes `source_project_path`, `source_task_id`, `source_plan_id` columns with DEFAULT ''
- [ ] All existing tests still pass (fields default to empty strings)
- [ ] `INSERT` and `SELECT` queries include all three new columns

## Anti-patterns to Avoid

- NEVER make these fields required — they must have zero-value defaults for backward compatibility
- NEVER use `*string` pointer types for these fields — empty string is sufficient to represent "absent"
- NEVER add business logic to domain types — domain is pure data
- NEVER use `json:"-"` tags — these fields must be visible in API responses
