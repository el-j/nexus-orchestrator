You are a nexusOrchestrator dogfood agent. When invoked, you submit PLAN-002 implementation tasks to the running nexusOrchestrator daemon so that LLMs can implement them.

## Input

`$ARGUMENTS` — optional single task ID (e.g. `TASK-013`) or `all` (default: submit all PLAN-002 todo tasks in wave order)

## Prerequisites Check

Before submitting, verify:
1. `curl -sf http://localhost:9999/api/health` returns `{"status":"ok"}` — daemon must be running
2. `curl -sf http://localhost:9999/api/providers` returns at least one `"active": true` — LLM must be online
3. `cmd/nexus-submit` binary exists at `$PWD/cmd/nexus-submit/` (run `CGO_ENABLED=1 go build ./cmd/nexus-submit/...` if not)

If either check fails: print instructions and stop.

## PLAN-002 Task Submission Map

Submit in this order (sequential per file group to avoid conflicts):

**Wave 1 — orchestrator hardening:**
- TASK-013 → target: `internal/core/services/orchestrator.go`, context: `internal/core/services/orchestrator.go,internal/core/ports/ports.go`
- TASK-014 → target: `internal/core/services/orchestrator.go`, context: `internal/core/services/orchestrator.go,internal/core/domain/task.go`

**Wave 2 — API + CLI:**
- TASK-015 → target: `internal/adapters/inbound/httpapi/server.go`, context: `internal/adapters/inbound/httpapi/server.go,internal/core/ports/ports.go`
- TASK-016 → target: `internal/adapters/inbound/cli/root.go`, context: `internal/adapters/inbound/cli/root.go`

**Wave 3 — writeback domain:**
- TASK-026 → target: `internal/core/domain/task.go`, context: `internal/core/domain/task.go,internal/adapters/outbound/repo_sqlite/repo.go`

**Wave 4 — writeback adapter:**
- TASK-027 → target: `internal/adapters/outbound/fs_writeback/writeback.go`, context: `internal/core/ports/ports.go`
- TASK-028 → target: `internal/core/services/orchestrator.go`, context: `internal/core/services/orchestrator.go`
- TASK-029 → target: `internal/adapters/inbound/httpapi/server.go`, context: `internal/adapters/inbound/httpapi/server.go,internal/adapters/inbound/mcp/server.go`

## Steps

### 1. Verify daemon + provider
Call `GET http://127.0.0.1:9999/api/health`.
If that fails:
```
ERROR: nexus-daemon is not running.
Start it with: CGO_ENABLED=1 go build -o /tmp/nexus-daemon ./cmd/nexus-daemon && NEXUS_DB_PATH=/tmp/nexus-local.db /tmp/nexus-daemon &
```
Stop here if daemon is not running.

### 2. Select tasks to submit
If `$ARGUMENTS` is a specific task ID: submit only that task.
If `$ARGUMENTS` is `all` or empty: submit all tasks in wave order (listed above).
Skip tasks that do not have a corresponding `.claude/tasks/TASK-NNN.md` file.

### 3. Submit each task

For each task in the selected set, use the submission map above to determine `target` and `context`, then:

```bash
CGO_ENABLED=1 go run ./cmd/nexus-submit \
  --task-file .claude/tasks/<TASK-ID>.md \
  --project "$PWD" \
  --target <target> \
  --context <context> \
  --addr http://127.0.0.1:9999
```

Record the returned `task_id`.

### 4. Update orchestrator.json

For each submitted task, update `.claude/orchestrator.json`:
- Set `tasks.<TASK-ID>.status` → `"pushed"`
- Set `tasks.<TASK-ID>.nexusTaskId` → returned task ID
- Set `tasks.<TASK-ID>.pushedAt` → current ISO 8601 timestamp

### 5. Report

Print:
```
=== Submitted N tasks to nexusOrchestrator ===
  TASK-013 → <nexus-uuid>
  TASK-014 → <nexus-uuid>
  ...

Dashboard: http://localhost:9999/ui
API:       http://localhost:9999/api/tasks

Run /sync-from-nexus to pull results back when ready.
```

## Constraints

- NEVER submit two tasks targeting the same Go source file simultaneously
- NEVER mark orchestrator.json tasks as "done" during submission — only "pushed"
- ALWAYS check daemon health before submitting
