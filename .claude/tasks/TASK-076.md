---
id: TASK-076
title: "HTTP API + MCP: Accept command param on submit"
role: api
planId: PLAN-009
status: todo
dependencies: [TASK-074]
createdAt: 2026-03-10T15:00:00.000Z
---

## Context
Both the HTTP API and MCP server must accept and pass through the new `command` field on task submissions. The field should be optional — if omitted, the orchestrator defaults to "auto".

## Files to Read
- `internal/core/domain/task.go` (after TASK-074 changes)
- `internal/adapters/inbound/httpapi/server.go`
- `internal/adapters/inbound/mcp/server.go`
- `internal/adapters/inbound/wailsbind/bind.go`
- `app.go`

## Implementation Steps
1. In `internal/adapters/inbound/httpapi/server.go`:
   - The `Task` struct already gets deserialized from JSON. Since `Command` has `json:"command,omitempty"`, it will naturally be included in POST `/api/tasks` requests. No code change needed here — the JSON binding already handles it.
   - Verify that `handleSubmit` passes the full task to `SubmitTask()` which includes the Command field.
2. In `internal/adapters/inbound/mcp/server.go`:
   - Add `Command string` field to the `submit_task` tool args struct (the `struct{ ProjectPath, TargetFile, Instruction, ContextFiles, Command }` in `handleToolCall`).
   - After building the `domain.Task`, set `task.Command = domain.CommandType(args.Command)` if non-empty.
3. In `app.go`:
   - No change needed — `SubmitTask(task domain.Task)` already passes the full Task struct which now includes Command.
4. In `internal/adapters/inbound/wailsbind/bind.go`:
   - Verify the Wails binding passes through the full Task struct (it should since it delegates to `orchestrator.SubmitTask(task)`).

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] HTTP POST `/api/tasks` with `"command": "plan"` passes it through to orchestrator
- [ ] MCP `submit_task` tool accepts `command` parameter
- [ ] Omitting `command` field works (backward compatible)

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
