---
id: TASK-310
title: Add MCP tests for 9 new tools + update tool-count assertion
role: testing
planId: PLAN-047
status: todo
dependencies: [TASK-305]
createdAt: 2026-03-25T00:00:00.000Z
---

## Context

After TASK-305 adds 9 new MCP tools (total: 29), the test file `internal/adapters/inbound/mcp/server_test.go` needs:

1. The tool-count assertion updated from 20 → 29
2. At least one test per new tool verifying: correct dispatch, required-field validation, and error propagation

## Files to Read

- `internal/adapters/inbound/mcp/server_test.go` — existing test patterns (mockOrch, MCP JSON-RPC request format, `TestMCP_ToolsList_Returns20Tools`)
- `internal/adapters/inbound/mcp/server.go` — each new tool handler (after TASK-305)

## Implementation Steps

1. Update `TestMCP_ToolsList_Returns20Tools` → `TestMCP_ToolsList_Returns29Tools`, change assertion from `20` to `29`.

2. For each new tool, add a test function:

```go
func TestMCP_ListProviderConfigs(t *testing.T) {
    orch := &mockOrch{}
    srv := newServer(t, orch)
    // POST MCP request: {"method":"tools/call","params":{"name":"list_provider_configs","arguments":{}}}
    // Assert 200, result contains provider config array
}
```

3. For tools with required fields, also test the missing-field error:

```go
func TestMCP_AddProviderConfig_MissingKind(t *testing.T) {
    // Call with no "kind" field → assert error response
}
```

4. Add mock methods to `mockOrch` for any new Orchestrator methods called by the new tools (e.g., `ListProviderConfigs`, `AddProviderConfig`, etc.) — they likely already exist on the mock from the interface enforcement, but verify.

## Acceptance Criteria

- [ ] `go vet ./internal/adapters/inbound/mcp/...` clean
- [ ] `go test ./internal/adapters/inbound/mcp/... -race -count=1` all pass
- [ ] Tool count assertion reflects 29
- [ ] Each of the 9 new tools has at least a success test
- [ ] MCP coverage improves from 56% to ≥70%
