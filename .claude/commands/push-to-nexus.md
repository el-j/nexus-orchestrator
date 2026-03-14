You are a nexusOrchestrator push agent. When invoked, you submit tasks from the current project to a running nexusOrchestrator daemon so they can be processed by LLMs and written back automatically.

## Input

`$ARGUMENTS` — one of:
- A specific task ID like `TASK-045` — push only that task
- `all` or empty — push all tasks with `status: "todo"` in the active plan

## Steps

### 1. Read local orchestrator state
- Read `.claude/orchestrator.json` to get `activePlanId` and the tasks map.
- Determine current workspace absolute path (use `$PWD` or the workspace root tool).
- Resolve the nexusOrchestrator base URL: `${NEXUS_ADDR:-http://127.0.0.1:63987}`.

### 2. Select tasks to push
- If `$ARGUMENTS` is a specific task ID: select only that task. Verify it has `status: "todo"` — abort with error if already `"done"` or `"pushed"`.
- If `$ARGUMENTS` is `all` or empty: select all tasks in `plans.<activePlanId>.taskIds` where `tasks.<id>.status == "todo"`.
- If no eligible tasks found: print "No todo tasks to push." and stop.

### 3. Push each task to nexusOrchestrator

For each selected task ID (process sequentially — each push depends on the previous recording):

a. Read the task file at `.claude/tasks/<TASK-ID>.md` — use the ENTIRE file content as the prompt.

b. Submit via HTTP:
   ```
   POST <NEXUS_ADDR>/api/tasks
   Content-Type: application/json

   {
     "projectPath":       "<absolute workspace path>",
     "prompt":            "<full content of TASK-ID.md>",
     "sourceProjectPath": "<absolute workspace path>",
     "sourceTaskId":      "<TASK-ID>",
     "sourcePlanId":      "<activePlanId>"
   }
   ```

c. On HTTP 201 response: record in `.claude/orchestrator.json`:
   - `tasks.<TASK-ID>.status` → `"pushed"`
   - `tasks.<TASK-ID>.nexusTaskId` → returned `id` field
   - `tasks.<TASK-ID>.pushedAt` → current ISO 8601 timestamp
   - `updatedAt` → current timestamp

d. On error: print `"ERROR: Failed to push <TASK-ID>: <http status> <message>"` and continue to next task (do not abort entire batch).

### 4. Summary
Print:
```
Pushed N tasks to nexusOrchestrator:
  ✓ TASK-NNN → nexus-task-id-xxx
  ✗ TASK-MMM → ERROR: connection refused

Run /sync-from-nexus to pull results when ready.
nexusOrchestrator dashboard: http://127.0.0.1:63987
```

## Constraints
- NEVER push a task that already has `status: "pushed"`, `"done"`, or `"failed"`.
- NEVER modify task `.md` files during push — only update `orchestrator.json`.
- ALWAYS record `nexusTaskId` before moving to the next task.
- Use `url.PathEscape` or equivalent encoding when placing paths in URL query strings.
