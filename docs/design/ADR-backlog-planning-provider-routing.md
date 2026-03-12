# ADR: Idea Backlog, Project-Scoped Planning & Per-Task Provider Routing

**Status:** Proposed  
**Date:** 2026-03-11  
**Author:** UXArchitect  

---

## 1. Build vs. Buy

**VERDICT: Build inside nexusOrchestrator.**

Reasoning:
- nexusOrchestrator already owns `ProjectPath` isolation, task lifecycle, provider routing, SQLite, and session history — backlog is a natural lifecycle extension (add states, not systems)
- A separate service would duplicate project scoping, need its own persistence, and require IPC (HTTP/gRPC) to promote backlog items into the queue — unnecessary complexity
- The user wants a single desktop window; two processes means two lifecycles, two crash domains, and a worse UX
- The `Task` entity already has `Command` (plan/execute/auto), `ModelID`, and `ProviderHint` — we're extending what exists, not inventing new concerns
- SQLite is the single source of truth; adding a `backlog_items` table (or new status values) is a trivial migration

**Risk:** feature creep. Mitigation: strict scope — backlog is just pre-queue states on the task lifecycle, not a full project management system.

---

## 2. Domain Model Extensions

### 2a. New TaskStatus Values

Add two statuses that exist *before* `QUEUED`:

```
StatusDraft   TaskStatus = "DRAFT"      // idea captured, not yet actionable
StatusBacklog TaskStatus = "BACKLOG"    // reviewed, actionable, awaiting promotion
```

These are **not** processed by the worker loop. Only `QUEUED` tasks enter the work queue.

Why two states (not one):
- `DRAFT` = raw capture, may be vague ("explore caching options")
- `BACKLOG` = refined, has enough detail to promote to queue
- Allows filtering: "show me ideas" vs "show me ready-to-go items"

### 2b. Extend Task — Do NOT Create a Separate Entity

Rationale: a backlog item has the same shape as a task — instruction, project path, context files, model preference, command type. Creating a `BacklogItem` struct would duplicate 90% of `Task` fields and require a conversion function. Instead, `DRAFT`/`BACKLOG` are just lifecycle states of `Task`.

New fields on `Task`:

```go
type Task struct {
    // ... existing fields ...

    // Priority orders tasks within a status group (lower = higher priority).
    // Default 0. Only meaningful for DRAFT/BACKLOG/QUEUED.
    Priority int `json:"priority,omitempty"`

    // ProviderName is the explicit provider assignment for this task.
    // When non-empty, FindForModel skips discovery and targets this provider directly.
    // Empty = use ProviderHint/discovery as today.
    ProviderName string `json:"providerName,omitempty"`

    // ParentID links a task to a parent (e.g., a plan task that spawned sub-tasks).
    // Empty = top-level task.
    ParentID string `json:"parentId,omitempty"`

    // Tags are free-form labels for filtering/grouping in the UI.
    Tags []string `json:"tags,omitempty"`
}
```

### 2c. Project Scoping — No Changes

`ProjectPath` already isolates tasks and sessions. The new statuses and fields are orthogonal — filtered by `ProjectPath + Status` in queries. No new scoping mechanism needed.

### 2d. Provider Routing Fields — Summary

| Field | Role | Match semantics |
|-------|------|-----------------|
| `ProviderName` (NEW) | Explicit provider lock | Exact match on `ProviderName()` |
| `ProviderHint` (existing) | Soft preference | Case-insensitive substring |
| `ModelID` (existing) | Model constraint | Exact model match via discovery |

Precedence in `FindForModel`:
1. If `ProviderName` is set → find that provider directly, verify it has `ModelID` (if set)
2. Else → existing 2-pass discovery with `ProviderHint`

---

## 3. Port & Service Changes

### 3a. TaskRepository Port — New Methods

```go
type TaskRepository interface {
    // ... existing methods ...

    // GetByProjectAndStatus returns tasks for a project filtered by status(es).
    GetByProjectAndStatus(projectPath string, statuses []domain.TaskStatus) ([]domain.Task, error)

    // UpdateTask persists all mutable fields (status, priority, provider assignment, tags, logs).
    // Replaces the need for individual Update* methods on new fields.
    UpdateTask(t domain.Task) error
}
```

`GetPending()` continues to return only `QUEUED` tasks (worker loop contract unchanged).

### 3b. Orchestrator Port — New Methods

```go
type Orchestrator interface {
    // ... existing methods ...

    // CreateDraft captures an idea without queuing it.
    // Returns the task ID. Status is set to DRAFT.
    CreateDraft(task domain.Task) (string, error)

    // PromoteTask moves a task forward in the lifecycle:
    //   DRAFT → BACKLOG, BACKLOG → QUEUED
    // Returns error if transition is invalid.
    PromoteTask(id string) error

    // DemoteTask moves a task backward:
    //   BACKLOG → DRAFT, QUEUED → BACKLOG
    // Cannot demote PROCESSING or terminal states.
    DemoteTask(id string) error

    // ListBacklog returns DRAFT + BACKLOG tasks for a project, ordered by priority.
    ListBacklog(projectPath string) ([]domain.Task, error)

    // UpdateTaskFields patches mutable fields (priority, providerName, modelID, tags, instruction).
    // Does NOT change status — use PromoteTask/DemoteTask/CancelTask for that.
    UpdateTaskFields(id string, patch domain.TaskPatch) error
}
```

### 3c. New Domain Type — TaskPatch

```go
// TaskPatch carries optional field updates for UpdateTaskFields.
// Pointer fields: nil = no change, non-nil = set to value.
type TaskPatch struct {
    Instruction  *string      `json:"instruction,omitempty"`
    Priority     *int         `json:"priority,omitempty"`
    ProviderName *string      `json:"providerName,omitempty"`
    ModelID      *string      `json:"modelId,omitempty"`
    ProviderHint *string      `json:"providerHint,omitempty"`
    Tags         *[]string    `json:"tags,omitempty"`
    Command      *CommandType `json:"command,omitempty"`
    TargetFile   *string      `json:"targetFile,omitempty"`
    ContextFiles *[]string    `json:"contextFiles,omitempty"`
}
```

### 3d. Worker Loop — No Change in Shape

The worker loop already drains `o.queue` which is populated only with `QUEUED` tasks. `DRAFT` and `BACKLOG` tasks never enter the queue slice. `processNext()` is unchanged.

The only enhancement: when `ProviderName` is set, `FindForModel` short-circuits discovery.

### 3e. DiscoveryService — FindForModel Enhancement

```
FindForModel(modelID, providerHint, providerName string) (LLMClient, error)
```

Add `providerName` parameter (or accept a struct). When `providerName != ""`:
- Look up that provider by exact `ProviderName()` match
- Verify it's alive (`Ping()`)  
- If `modelID` is set, verify it can serve that model
- Skip all other candidates

Backward-compatible: existing callers pass `""` for `providerName`.

---

## 4. Lifecycle State Machine

```
                  ┌──────────┐
   CreateDraft───▶│  DRAFT   │
                  └────┬─────┘
                       │ PromoteTask
                  ┌────▼─────┐
                  │ BACKLOG  │◄── DemoteTask ──┐
                  └────┬─────┘                 │
                       │ PromoteTask      ┌────┴─────┐
                  ┌────▼─────┐            │  QUEUED   │
                  │  QUEUED  ├────────────┘  (in mem) │
                  └────┬─────┘                        │
                       │ worker dequeue               │
                  ┌────▼──────┐                       │
                  │PROCESSING │                       │
                  └──┬─────┬──┘                       │
                     │     │                          │
              ┌──────▼┐  ┌▼───────┐                   │
              │COMPLETED│ │ FAILED │                   │
              └────────┘  └───┬───┘                   │
                              │ (retry = re-submit)   │
                              └───────────────────────┘
```

**Valid transitions:**

| From | To | Trigger |
|------|----|---------|
| (new) | DRAFT | `CreateDraft()` |
| (new) | QUEUED | `SubmitTask()` (existing — unchanged) |
| DRAFT | BACKLOG | `PromoteTask()` |
| DRAFT | CANCELLED | `CancelTask()` |
| BACKLOG | QUEUED | `PromoteTask()` |
| BACKLOG | DRAFT | `DemoteTask()` |
| BACKLOG | CANCELLED | `CancelTask()` |
| QUEUED | BACKLOG | `DemoteTask()` |
| QUEUED | PROCESSING | worker loop |
| QUEUED | CANCELLED | `CancelTask()` |
| PROCESSING | COMPLETED | worker success |
| PROCESSING | FAILED | worker error |
| PROCESSING | TOO_LARGE | pre-flight check |
| PROCESSING | NO_PROVIDER | no provider found |

Terminal states: `COMPLETED`, `FAILED`, `CANCELLED`, `TOO_LARGE`, `NO_PROVIDER`.

Failed tasks can be re-submitted (creates a new task) — not re-promoted.

---

## 5. Per-Task Provider Routing — Detail

### Current behavior
```
Task.ModelID      → "which model"    (empty = any active model)
Task.ProviderHint → "prefer who"     (empty = no preference)
FindForModel(modelID, hint) → 2-pass discovery
```

### New behavior
```
Task.ProviderName → "exactly who"    (empty = use discovery)
Task.ModelID      → "which model"    (unchanged)
Task.ProviderHint → "prefer who"     (unchanged, lower priority than ProviderName)
```

Resolution order in `FindForModel`:
1. **ProviderName set** → exact lookup by `ProviderName()`. If not alive → `NO_PROVIDER`. If alive but wrong model → `NO_PROVIDER`.
2. **ProviderName empty** → existing 2-pass (hint-prioritized discovery). No behavior change.

This is additive. Existing tasks with empty `ProviderName` route exactly as today.

### Validation
- `CreateDraft` / `SubmitTask`: if `ProviderName` is set, verify it exists in discovered providers (warn, don't block — provider may come online later)
- `PromoteTask` to QUEUED: no validation on provider (same as SubmitTask today)

---

## 6. SQLite Schema Changes

```sql
-- Migration: add backlog/planning columns to tasks
ALTER TABLE tasks ADD COLUMN priority      INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tasks ADD COLUMN provider_name TEXT    NOT NULL DEFAULT '';
ALTER TABLE tasks ADD COLUMN parent_id     TEXT    NOT NULL DEFAULT '';
ALTER TABLE tasks ADD COLUMN tags          TEXT    NOT NULL DEFAULT '[]';
```

Follow existing pattern: idempotent `ALTER TABLE ... ADD COLUMN` with error suppression (column may already exist).

No new tables. `DRAFT`/`BACKLOG` are just status values in the existing `tasks` table.

---

## 7. HTTP API Extensions (Additive)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/tasks/draft` | Create a DRAFT task |
| `POST` | `/api/tasks/{id}/promote` | DRAFT→BACKLOG or BACKLOG→QUEUED |
| `POST` | `/api/tasks/{id}/demote` | QUEUED→BACKLOG or BACKLOG→DRAFT |
| `PATCH` | `/api/tasks/{id}` | Update mutable fields (TaskPatch) |
| `GET` | `/api/tasks?project={path}&status={s}` | Filter by project + status |

Existing endpoints unchanged. `POST /api/tasks` still creates `QUEUED` tasks.

---

## 8. MCP Tool Extensions

| Tool | Description |
|------|-------------|
| `nexus_create_draft` | Capture an idea as DRAFT |
| `nexus_promote_task` | Advance lifecycle state |
| `nexus_list_backlog` | List DRAFT+BACKLOG for a project |

Existing 6 tools unchanged.

---

## 9. Task Breakdown

### Wave 1 — Domain & Ports (no adapter changes)

| # | Title | Description | Role |
|---|-------|-------------|------|
| 1 | Add DRAFT/BACKLOG to TaskStatus | Add constants + update `IsTerminal()` helper if exists | UXArchitect |
| 2 | Add new fields to Task struct | `Priority`, `ProviderName`, `ParentID`, `Tags` | UXArchitect |
| 3 | Add TaskPatch domain type | Pointer-field patch struct for partial updates | UXArchitect |
| 4 | Extend port interfaces | `TaskRepository.GetByProjectAndStatus`, `UpdateTask`; `Orchestrator.CreateDraft`, `PromoteTask`, `DemoteTask`, `ListBacklog`, `UpdateTaskFields` | UXArchitect |

### Wave 2 — Service & Persistence (deps: Wave 1)

| # | Title | Description | Role |
|---|-------|-------------|------|
| 5 | SQLite migration for new columns | `priority`, `provider_name`, `parent_id`, `tags` via ALTER TABLE | Backend |
| 6 | Implement TaskRepository new methods | `GetByProjectAndStatus`, `UpdateTask` in repo_sqlite | Backend |
| 7 | Implement orchestrator backlog methods | `CreateDraft`, `PromoteTask`, `DemoteTask`, `ListBacklog`, `UpdateTaskFields` in OrchestratorService | Backend |
| 8 | Enhance FindForModel for ProviderName | Add `providerName` parameter, exact-match short-circuit | Backend |

### Wave 3 — Adapters (deps: Wave 2)

| # | Title | Description | Role |
|---|-------|-------------|------|
| 9 | HTTP API endpoints for backlog | `POST /draft`, `POST /promote`, `POST /demote`, `PATCH`, query filters | Backend |
| 10 | MCP tools for backlog | `nexus_create_draft`, `nexus_promote_task`, `nexus_list_backlog` | Backend |
| 11 | GUI backlog panel | Project-scoped view with DRAFT/BACKLOG columns, drag-to-promote | Frontend |
| 12 | CLI backlog commands | `nexus-cli draft`, `nexus-cli promote`, `nexus-cli backlog` | Backend |

---

## 10. What This ADR Does NOT Cover

- Plan entity as a first-class domain type (plans remain task trees via `ParentID`)
- Automatic scheduling / priority-based ordering in the worker queue
- Multi-user / auth concerns
- Plan import from `.claude/orchestrator.json` (migration tool — separate ADR)

---

## Appendix: Key Constraints Verified

- [x] Hexagonal: no adapter imports in `internal/core/`
- [x] Additive: existing `SubmitTask`, `GetQueue`, `CancelTask` unchanged
- [x] Worker loop: only `QUEUED` tasks dequeued — DRAFT/BACKLOG invisible to worker
- [x] SQLite: idempotent ALTER TABLE migrations (existing pattern)
- [x] Go 1.24 compatible, no generics abuse, `fmt.Errorf` wrapping
