---
id: TASK-030
title: New command — .claude/commands/push-to-nexus.md
role: planning
planId: PLAN-002
status: todo
dependencies: [TASK-029]
createdAt: 2026-03-09T13:00:00.000Z
---

## Context

External projects that use the `.claude/commands/` workflow system need a way to submit their tasks to nexusOrchestrator and record the submission so the result can be retrieved later. `push-to-nexus.md` is a Copilot agent command that reads the current project's `orchestrator.json`, finds tasks with a given status (default: `todo`), and submits each one to nexusOrchestrator via `POST /api/tasks` with source writeback fields set.

After this command runs, the source project's `orchestrator.json` is updated with `nexusTaskId` so `sync-from-nexus` can match results.

## Files to Read

- `.claude/commands/new-task.md` — example command file format
- `.claude/commands/orchestrator.md` — example with $ARGUMENTS and Steps
- `.claude/orchestrator.json` — understand the task schema being read
- `internal/adapters/inbound/httpapi/server.go` — POST /api/tasks request shape
- `internal/adapters/inbound/mcp/server.go` — MCP submit_task shape (alternative transport)

## Implementation Steps

1. **Create `.claude/commands/push-to-nexus.md`** with the following content and behaviour:

   - `$ARGUMENTS` = optional task ID (e.g. `TASK-045`) or `all` (default: submit all `todo` tasks)
   - Steps:
     a. Read `.claude/orchestrator.json` to get `activePlanId` and tasks.
     b. Filter tasks: if `$ARGUMENTS` is a specific task ID, push only that one; if `all` or empty, push all tasks with `status: "todo"` in the active plan.
     c. For each task to push:
        - Read its `.claude/tasks/TASK-NNN.md` file to extract the full implementation description as the prompt (use the entire file content).
        - Call `POST http://127.0.0.1:9999/api/tasks` with:
          ```json
          {
            "projectPath": "<absolute path of current workspace>",
            "prompt": "<full TASK-NNN.md content>",
            "sourceProjectPath": "<absolute path of current workspace>",
            "sourceTaskId": "TASK-NNN",
            "sourcePlanId": "<activePlanId>"
          }
          ```
        - On success: record `nexusTaskId` in `.claude/orchestrator.json` under `tasks.TASK-NNN.nexusTaskId`.
        - Update task status to `"pushed"` in orchestrator.json.
     d. Print summary: "Pushed N tasks to nexusOrchestrator. Run /sync-from-nexus to retrieve results."

   Note: Use `NEXUS_ADDR` env var for the base URL (default `http://127.0.0.1:9999`).

2. The command file is a Copilot instruction file (plain Markdown), NOT Go code. The agent (Copilot/Claude) reads it and executes the HTTP calls via tool calls.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0 (command file does not affect Go compilation)
- [ ] `CGO_ENABLED=1 go build ./...` exits 0
- [ ] `.claude/commands/push-to-nexus.md` exists
- [ ] File contains `$ARGUMENTS` handling for single task ID or `all`
- [ ] File documents `sourceProjectPath`, `sourceTaskId`, `sourcePlanId` fields in HTTP call
- [ ] File documents that `nexusTaskId` is recorded back in orchestrator.json
- [ ] File references `NEXUS_ADDR` env var override

## Anti-patterns to Avoid

- NEVER hardcode workspace paths — use `$PWD` or workspace root via tool call
- NEVER push tasks that are already `"done"` or `"pushed"` — check status before submitting
- NEVER submit without recording the returned `nexusTaskId` — it is required for sync-from-nexus
