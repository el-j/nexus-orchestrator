---
id: TASK-254
title: "MCP: Add claim_task and update_task_status tools"
role: mcp
planId: PLAN-038
status: todo
dependencies: [TASK-253]
createdAt: 2026-03-13T16:00:00.000Z
---

## Context
External AI agents (Copilot, Claude Desktop) interact with nexusOrchestrator via MCP JSON-RPC 2.0. They currently have no way to claim a queued task or report completion. This task adds two MCP tools that call the new service methods.

## Files to Read
- `internal/adapters/inbound/mcp/server.go` (existing tool patterns: toolSubmitTask, toolRegisterSession)
- `internal/core/ports/ports.go` (Orchestrator interface with ClaimTask, UpdateTaskStatus)

## Implementation Steps
1. Add `claim_task` tool definition to `toolList()`:
   - Parameters: `task_id` (required string), `session_id` (required string)
   - Description: "Claim a queued task for execution by the specified AI session"

2. Implement `toolClaimTask(ctx, args)`:
   - Parse `task_id` and `session_id` from args
   - Call `s.orch.ClaimTask(ctx, taskID, sessionID)`
   - Return task JSON on success

3. Add `update_task_status` tool definition to `toolList()`:
   - Parameters: `task_id` (required string), `session_id` (required string), `status` (required enum: "COMPLETED"/"FAILED"), `logs` (optional string)
   - Description: "Report task completion or failure from the executing AI session"

4. Implement `toolUpdateTaskStatus(ctx, args)`:
   - Parse args, validate status enum
   - Call `s.orch.UpdateTaskStatus(ctx, taskID, sessionID, status, logs)`
   - Return updated task JSON

5. Add tool names to the `switch` dispatch in `handleToolsCall`.

## Acceptance Criteria
- [ ] `claim_task` tool appears in MCP `tools/list` response
- [ ] `update_task_status` tool appears in MCP `tools/list` response
- [ ] `claim_task` returns claimed task JSON on success
- [ ] `update_task_status` returns updated task on success
- [ ] Invalid parameters return proper MCP error responses
- [ ] `go vet ./...` passes
