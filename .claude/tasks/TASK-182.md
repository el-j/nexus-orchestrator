---
id: TASK-182
title: CLI — backlog and promote subcommands
role: cli
planId: PLAN-024
status: todo
dependencies: [TASK-176]
createdAt: 2026-03-11T22:00:00.000Z
---

## Context
The CLI client needs subcommands to interact with the new backlog endpoints on the daemon. Users should be able to create drafts, list backlogs, promote tasks, and update tasks from the terminal.

## Files to Read
- `internal/adapters/inbound/cli/root.go` — existing Cobra commands
- `cmd/nexus-cli/main.go` — CLI entry point
- `internal/adapters/inbound/httpapi/server.go` — new endpoints from TASK-176

## Implementation Steps

1. Add `nexus-cli draft` command:
   - Flags: `--project` (required), `--instruction` (required), `--target`, `--context`, `--provider`, `--model`, `--priority` (default 2), `--tags` (comma-separated)
   - POST to `http://127.0.0.1:9999/api/tasks/draft`
   - Print `Draft created: {task_id}`

2. Add `nexus-cli backlog` command:
   - Flags: `--project` (required)
   - GET `http://127.0.0.1:9999/api/tasks/backlog/{projectPath}`
   - Print table: ID | Priority | Status | Instruction (truncated) | Provider | Tags

3. Add `nexus-cli promote` command:
   - Arg: task ID (required)
   - POST to `http://127.0.0.1:9999/api/tasks/{id}/promote`
   - Print `Task {id} promoted to queue`

4. Add `nexus-cli update` command:
   - Arg: task ID (required)
   - Flags: `--instruction`, `--provider`, `--model`, `--priority`, `--tags`, `--status`
   - PUT to `http://127.0.0.1:9999/api/tasks/{id}`
   - Print updated task

5. Update `nexus-cli submit` to accept `--provider` and `--priority` flags.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/...` exits 0
- [ ] `nexus-cli draft --project=/path --instruction="..."` creates a DRAFT task
- [ ] `nexus-cli backlog --project=/path` lists backlog items
- [ ] `nexus-cli promote <id>` transitions to QUEUED
- [ ] `nexus-cli update <id> --priority=1` updates the task

## Anti-patterns to Avoid
- NEVER link core services directly — CLI is a remote HTTP client only
- NEVER skip `fmt.Errorf("cli: operation: %w", err)` error wrapping
