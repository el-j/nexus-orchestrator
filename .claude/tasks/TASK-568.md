---
id: TASK-568
title: CLI expansion — config, plans, logs, brain focused-context, tasks claim/heartbeat
role: cli
planId: PLAN-071
status: todo
dependencies: [TASK-566]
createdAt: 2026-04-15T00:00:00Z
---

## Context

Several major API groups have no CLI surface: runtime config (get/set), plan file discovery (discovered, scan), activity logs streaming, brain focused-context query, and worker-side task operations (claim, status update, heartbeat). These are needed for headless agents and scripted workflows.

## Files to Read

- `internal/adapters/inbound/cli/root.go`
- `internal/adapters/outbound/httpapi_client/client.go`
- `internal/core/domain/task.go`

## Implementation Steps

1. Add `config` command group:
   - `config get` — GET /api/config; print JSON or key=value table
   - `config set --queue-cap <n>` — PUT /api/config with partial update
2. Add `plans` command group:
   - `plans discovered [--project-path <p>]` — GET /api/plans/discovered; table: ID, path, kind, projectPath, isActive
   - `plans scan [--project-path <p>]` — POST /api/plans/discovered/scan
3. Add `logs` command:
   - `logs [--follow] [--n <count>]` — GET /api/logs; if `--follow`, open SSE stream to `/api/events` and print log events until Ctrl-C
4. Add `brain focused-context` subcommand to existing `brain` group:
   - `brain focused-context --project-path <p> --question <q> [--max-tokens <n>]` — POST /api/brain/focused-context
5. Extend existing `tasks` command group:
   - `tasks claim <task-id> --session-id <s>` — POST /api/tasks/{id}/claim
   - `tasks heartbeat <task-id> --session-id <s>` — POST /api/tasks/{id}/heartbeat
   - `tasks update-status <task-id> --session-id <s> --status <COMPLETED|FAILED> [--logs <text>]` — PUT /api/tasks/{id}/status

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `nexus-cli config get` and `config set --queue-cap` work
- [ ] `nexus-cli plans discovered` and `plans scan` work
- [ ] `nexus-cli logs --follow` streams SSE events
- [ ] `nexus-cli brain focused-context` calls the correct endpoint
- [ ] `nexus-cli tasks claim`, `tasks heartbeat`, `tasks update-status` work

## Anti-patterns to Avoid

- NEVER block the process without handling SIGINT (logs --follow must exit cleanly on Ctrl-C)
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
