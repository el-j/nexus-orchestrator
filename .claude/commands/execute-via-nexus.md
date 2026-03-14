You are a nexusOrchestrator **remote execution orchestrator**. When invoked, you push an entire plan's tasks to a running nexusOrchestrator daemon in dependency-wave order, poll for completion, and report results — a fully automated self-dogfood cycle.

## Input

`$ARGUMENTS` — one of:
- A specific plan ID like `PLAN-008` — execute that plan
- `active` or empty — use `activePlanId` from orchestrator.json

## Steps

### 1. Read local orchestrator state
- Read `.claude/orchestrator.json` to get `activePlanId`, the plans map, and the tasks map.
- Determine current workspace absolute path (use `$PWD` or the workspace root tool).
- Resolve the nexusOrchestrator base URL: `${NEXUS_ADDR:-http://127.0.0.1:63987}`.

### 2. Resolve target plan
- If `$ARGUMENTS` is a specific plan ID (e.g. `PLAN-008`): use it. Verify it exists in `plans` — abort if not found.
- If `$ARGUMENTS` is `active` or empty: use `activePlanId`. Abort if `activePlanId` is null or empty.
- Collect all task IDs from `plans.<planId>.taskIds`.

### 3. Verify daemon is running

a. Health check:
   ```
   GET <NEXUS_ADDR>/api/health
   ```
   If this fails (connection refused, non-200):
   ```
   ERROR: nexusOrchestrator daemon not reachable at <NEXUS_ADDR>.
   Start it with one of:
     wails dev                              # desktop GUI with hot-reload
     go run ./cmd/nexus-daemon/...          # headless daemon
   Then re-run this command.
   ```
   Stop execution.

b. Provider check:
   ```
   GET <NEXUS_ADDR>/api/providers
   ```
   Verify the response contains at least one provider with available models. If no providers:
   ```
   ERROR: No LLM providers available. Start LM Studio or Ollama first.
   ```
   Stop execution.

### 4. Build dependency graph and waves
- From the resolved plan, collect all tasks with `status: "todo"` in `orchestrator.json`.
- Skip tasks with status `"done"`, `"pushed"`, `"in-progress"`, or `"failed"`.
- If no `"todo"` tasks remain: print `"No todo tasks in <planId>. Plan may already be complete."` and stop.
- Build waves using topological sort on `tasks.<id>.dependencies`:
  - **Wave 1**: tasks whose dependencies are all `"done"` (or have no dependencies).
  - **Wave N**: tasks whose dependencies are all `"done"` or were in waves 1..N-1.
- Tasks whose dependencies include any `"failed"` task are **blocked** — exclude them and report at the end.

### 5. Process waves sequentially

For each wave (starting from wave 1):

Print: `\n=== Wave <N> (<count> tasks) ===\n`

#### 5a. Push all tasks in the wave

For each task ID in the wave (process sequentially):

1. Read the task file at `.claude/tasks/<TASK-ID>.md` — use the ENTIRE file content as the instruction.

2. Submit via HTTP:
   ```
   POST <NEXUS_ADDR>/api/tasks
   Content-Type: application/json

   {
     "projectPath":       "<absolute workspace path>",
     "prompt":            "<full content of TASK-ID.md>",
     "sourceProjectPath": "<absolute workspace path>",
     "sourceTaskId":      "<TASK-ID>",
     "sourcePlanId":      "<planId>"
   }
   ```

3. On HTTP 201 response: update `.claude/orchestrator.json`:
   - `tasks.<TASK-ID>.status` → `"pushed"`
   - `tasks.<TASK-ID>.nexusTaskId` → returned `id` field
   - `tasks.<TASK-ID>.pushedAt` → current ISO 8601 timestamp
   - `updatedAt` → current timestamp

4. On push error: mark `tasks.<TASK-ID>.status` → `"failed"`, print error, and **stop the entire plan** (do not proceed to next task or wave).

#### 5b. Poll for wave completion

Poll every 5 seconds until all tasks in the wave are resolved or timeout is reached.

For each pushed task in the wave:
```
GET <NEXUS_ADDR>/api/tasks/<nexusTaskId>
```

Interpret the response `status` field:
- `"completed"` → task is done
- `"failed"` → task failed
- `"queued"` or `"processing"` → still running

**Timeout**: 5 minutes per wave. Start the timer after all tasks in the wave are pushed.

Print progress every 30 seconds:
```
  ⏳ Waiting... <completed>/<total> done, <elapsed>s elapsed
```

#### 5c. Handle wave results

**All tasks completed:**
- For each task: update `.claude/orchestrator.json`:
  - `tasks.<TASK-ID>.status` → `"done"`
  - `tasks.<TASK-ID>.completedAt` → current ISO 8601 timestamp
  - `updatedAt` → current timestamp
- Print: `  ✓ Wave <N> complete: all <count> tasks done.`
- Proceed to next wave.

**Any task failed:**
- Mark failed tasks as `"failed"` in `orchestrator.json` with `failedAt` timestamp.
- Mark completed tasks as `"done"`.
- Print:
  ```
  ✗ Wave <N> failed:
    ✓ TASK-NNN → done
    ✗ TASK-MMM → failed
  Stopping — downstream waves depend on failed tasks.
  ```
- **Stop execution** — do not proceed to remaining waves.

**Timeout reached:**
- Mark completed tasks as `"done"`.
- Leave timed-out tasks as `"pushed"` (they may still complete in the daemon).
- Print:
  ```
  ⏰ Wave <N> timed out after 5 minutes:
    ✓ TASK-NNN → done
    ⏳ TASK-MMM → still pushed (may complete later)
  Run /sync-from-nexus to check for late completions.
  ```
- **Stop execution** — do not proceed to remaining waves.

### 6. Summary

After all waves complete (or execution stops), print:

```
╔══════════════════════════════════════════════════╗
║          nexusOrchestrator Execution Report      ║
╠══════════════════════════════════════════════════╣
  Plan: <planId>
  Result: COMPLETE | PARTIAL | FAILED

  Task Summary:
  ┌──────────┬──────────┬──────────────────────────┐
  │ Task ID  │ Status   │ Nexus Task ID            │
  ├──────────┼──────────┼──────────────────────────┤
  │ TASK-068 │ ✓ done   │ abc-123-def              │
  │ TASK-069 │ ✗ failed │ def-456-ghi              │
  │ TASK-070 │ ⏭ skip   │ —                        │
  └──────────┴──────────┴──────────────────────────┘

  Done:    N tasks
  Failed:  N tasks
  Pushed:  N tasks (still running)
  Blocked: N tasks (upstream failed)
  Skipped: N tasks (already done)
╚══════════════════════════════════════════════════╝
```

If plan is fully complete:
- Set `plans.<planId>.status` → `"completed"` and `completedAt` → current timestamp in `orchestrator.json`.
- Print: `Plan <planId> marked as completed.`

If partial or failed:
- Print: `Run /sync-from-nexus to check for late results, or fix failed tasks and re-run.`

## Constraints
- **NEVER** modify Go source files, task `.md` files, or any file outside `.claude/orchestrator.json`.
- **NEVER** push a task whose dependencies are not all `"done"` — wave ordering enforces this.
- **ALWAYS** update `orchestrator.json` after each state change (push, completion, failure) before proceeding.
- **ALWAYS** leave `orchestrator.json` in a consistent state — if interrupted, pushed tasks stay `"pushed"` and can be synced later with `/sync-from-nexus`.
- **NEVER** retry a failed push or a failed task automatically — report and stop.
- Use `url.PathEscape` or equivalent encoding when placing paths in URL query strings.
