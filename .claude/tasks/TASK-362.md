# TASK-362: Add comprehensive MCP transport tests

**Plan:** PLAN-052 | **Wave:** 6 | **Status:** done

## Description

Add tests for all new MCP transport functionality:

- Legacy SSE transport (connect, endpoint event, message exchange)
- Streamable HTTP (GET returns 405, POST works, session ID)
- Missing methods (ping, resources/list, prompts/list)
- Origin validation (allowed, blocked)
- CORS preflight

## Files to create/modify

- `internal/adapters/inbound/mcp/sse_test.go`
- `internal/adapters/inbound/mcp/server_test.go` (add new test cases)

## Acceptance criteria

- All new functionality has test coverage
- Tests pass with `-race`
