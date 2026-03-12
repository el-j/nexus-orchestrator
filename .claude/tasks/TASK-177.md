---
id: TASK-177
title: MCP — draft, backlog, promote, update tools
role: mcp
planId: PLAN-024
status: todo
dependencies: [TASK-176]
createdAt: 2026-03-11T22:00:00.000Z
---

## Context
MCP clients (Claude Desktop, VS Code extension) need JSON-RPC 2.0 tools to create drafts, query backlogs, promote items, and update tasks — mirroring the new HTTP endpoints.

## Files to Read
- `internal/adapters/inbound/mcp/server.go` — existing tool registration pattern
- `internal/core/ports/ports.go` — updated Orchestrator interface

## Implementation Steps

1. Register 4 new tools in `registerTools()`:

2. **Tool: `create_draft`**
   - Description: "Create a draft idea for a project without entering the execution queue"
   - Input: `{ projectPath, instruction, targetFile?, contextFiles?, providerName?, modelId?, priority?, tags? }`
   - Required: `["projectPath", "instruction"]`
   - Handler: build `domain.Task`, call `s.orch.CreateDraft(task)`, return `{task_id, status: "DRAFT"}`

3. **Tool: `get_backlog`**
   - Description: "List draft and backlog items for a project, ordered by priority"
   - Input: `{ projectPath }`
   - Required: `["projectPath"]`
   - Handler: call `s.orch.GetBacklog(projectPath)`, return JSON array

4. **Tool: `promote_task`**
   - Description: "Promote a draft/backlog task to the execution queue"
   - Input: `{ id }`
   - Required: `["id"]`
   - Handler: call `s.orch.PromoteTask(id)`, return `{promoted: true}`
   - On ErrNotFound: MCP error -32602

5. **Tool: `update_task`**
   - Description: "Update fields on an existing task (instruction, priority, provider, tags, status)"
   - Input: `{ id, instruction?, priority?, providerName?, modelId?, tags?, status? }`
   - Required: `["id"]`
   - Handler: build partial Task, call `s.orch.UpdateTask(id, updates)`, return updated task

6. Update MCP `initialize` response tool count.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] 4 new MCP tools registered and functional
- [ ] `create_draft` returns task_id without entering queue
- [ ] `promote_task` transitions to QUEUED
- [ ] Existing MCP tools unchanged

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER skip `fmt.Errorf("mcp: operation: %w", err)` error wrapping
- NEVER break existing MCP tool handlers
