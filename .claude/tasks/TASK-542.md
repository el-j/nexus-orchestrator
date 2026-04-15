---
id: TASK-542
planId: PLAN-070
title: 'Add 4 missing brain MCP tools + fix howto full guide enumeration'
role: mcp
status: todo
createdAt: 2026-04-14T02:00:00Z
---

# TASK-542 — Add missing brain MCP tools

## Context

PLAN-069 added 4 new brain HTTP routes (`init`, `knowledge` CRUD, `file-map`). These have HTTP
handlers and CLI subcommands but **no corresponding MCP tools**. AI agents using MCP cannot:

- Initialise a project's knowledge base (`InitProject`)
- List knowledge entries (`ListKnowledge`)
- Delete stale knowledge entries (`DeleteKnowledge`)
- Retrieve the project file map (`GetFileMap`)

Additionally, the `howto` MCP full guide (`toolHowto`, around line 696 in `tools.go`) enumerates
tools in its body text but does not include any brain tools or brain HTTP routes.

## Work Required

### `internal/adapters/inbound/mcp/brain_tools.go`

Add 4 new tool handler methods:

- `toolInitProject(ctx, args)` — calls `s.brain.InitProject(ctx, projectPath, claudeMDPath)`
- `toolListKnowledge(ctx, args)` — calls `s.brain.ListKnowledge(ctx, projectPath, kind)`
- `toolDeleteKnowledge(ctx, args)` — calls `s.brain.DeleteKnowledge(ctx, id)`
- `toolGetFileMap(ctx, args)` — calls `s.brain.GetFileMap(ctx, projectPath, focusArea)`

All must guard `if s.brain == nil { return callToolResult{}, fmt.Errorf("brain service not configured") }`.

### `internal/adapters/inbound/mcp/tools.go`

1. In `dispatch` switch: add 4 new cases (`"init_project"`, `"list_knowledge"`, `"delete_knowledge"`, `"get_file_map"`).
2. In `toolList()`: add 4 new `toolDef` entries with full JSON schemas (required params, descriptions).
3. In `toolHowto()` (the full guide): update the brain tools enumeration to list all 9 brain tools.
4. Update the `TestMCP_ToolsList_Returns36Tools` test (or rename to `Returns40Tools`) — currently at 36, will be 40 after this task. Check the test file first to confirm the count and test name.

## File Targets

- `internal/adapters/inbound/mcp/brain_tools.go`
- `internal/adapters/inbound/mcp/tools.go`
- `internal/adapters/inbound/mcp/server_test.go` (update tool count assertion)

## Acceptance Criteria

- `go vet ./internal/adapters/inbound/mcp/...` clean
- `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -race -count=1 ./internal/adapters/inbound/mcp/...` green
- `toolList()` returns 40 tools
- AI agents can `init_project`, `list_knowledge`, `delete_knowledge`, `get_file_map` via MCP
