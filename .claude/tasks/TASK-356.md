# TASK-356: Add missing standard MCP protocol methods

**Plan:** PLAN-052 | **Wave:** 1 | **Status:** done

## Description

Add handlers for standard MCP methods that clients expect:

- `ping` → returns `{}` result
- `resources/list` → returns `{"resources": []}`
- `prompts/list` → returns `{"prompts": []}`

These are required by MCP spec — clients may probe them during capability discovery.

## Files to modify

- `internal/adapters/inbound/mcp/server.go` — add cases in `handleRPC` switch

## Acceptance criteria

- `ping` returns `{"jsonrpc":"2.0","id":...,"result":{}}`
- `resources/list` returns empty resources array
- `prompts/list` returns empty prompts array
- Tests pass
