---
id: TASK-031
title: New command — .claude/commands/sync-from-nexus.md
role: planning
planId: PLAN-002
status: todo
dependencies: [TASK-030]
createdAt: 2026-03-09T13:00:00.000Z
---

## Context

After external tasks have been pushed to nexusOrchestrator (via push-to-nexus), the `sync-from-nexus` command retrieves their results and updates the source project's local state. It polls `GET /api/tasks?sourceProjectPath=<path>` to find completed tasks, then:
1. Marks them `"done"` in the local `.claude/orchestrator.json`
2. Appends the LLM output to the local `.claude/tasks/TASK-NNN.md` under `## Nexus Output`

Note: `fs_writeback` (TASK-027/028) does this automatically on the nexusOrchestrator side when the daemon can reach the source filesystem. `sync-from-nexus` is the MANUAL/PULL alternative — useful when the daemon runs on a remote machine that cannot write to the source project.

## Files to Read

- `.claude/commands/push-to-nexus.md` — (created in TASK-030) to understand the push side
- `internal/adapters/inbound/httpapi/server.go` — `GET /api/tasks` route and `GetBySourceProject` query param
- `.claude/orchestrator.json` — task schema to understand what fields to update

## Implementation Steps

1. **Create `.claude/commands/sync-from-nexus.md`** with the following content and behaviour:

   - `$ARGUMENTS` = optional planId filter (default: sync all tasks for current project)
   - Steps:
     a. Determine current workspace path.
     b. Call `GET http://127.0.0.1:9999/api/tasks?sourceProjectPath=<encoded-workspace-path>` to find all tasks submitted from this project.
     c. Filter response: tasks with `status: "completed"` or `status: "failed"` that have a matching `sourceTaskId` in the local orchestrator.json.
     d. For each matched remote task:
        - Read `GET /api/tasks/<nexusTaskId>` to get full task data.
        - Read local `.claude/tasks/<sourceTaskId>.md`.
        - Append section:
          ```markdown
          ## Nexus Output
          <!-- Synced from nexusOrchestrator on <timestamp> — NexusTaskID: <id> — Status: completed -->

          <task output from nexus — the LLM response/file write content>
          ```
        - Update local `.claude/orchestrator.json`: set `tasks.<sourceTaskId>.status = "done"`, `completedAt = <timestamp>`, `nexusTaskId = <id>`.
     e. Print summary: "Synced N tasks. N failed. Run /execute-task to act on results."

   Note: If `fs_writeback` has already updated the file, the command checks for an existing `## Nexus Output` section and skips re-appending.

2. The command file is a Copilot/Claude instruction file — NOT Go code.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./...` exits 0
- [ ] `.claude/commands/sync-from-nexus.md` exists
- [ ] File uses `GET /api/tasks?sourceProjectPath=` endpoint
- [ ] File documents duplicate-detection (skip if `## Nexus Output` already present)
- [ ] File updates both `.md` task file AND `orchestrator.json` on successful sync
- [ ] File handles `status: "failed"` tasks (marks as `"failed"` locally, not `"done"`)

## Anti-patterns to Avoid

- NEVER overwrite task file content — only APPEND the Nexus Output section
- NEVER mark tasks "done" without first verifying status is "completed" on nexus side
- NEVER use raw output from nexus without wrapping in the structured section header
