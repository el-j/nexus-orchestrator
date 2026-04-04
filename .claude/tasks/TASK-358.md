# TASK-358: Add Streamable HTTP transport support

**Plan:** PLAN-052 | **Wave:** 3 | **Status:** done

## Description

Upgrade the `/mcp` endpoint to support the Streamable HTTP transport (2025-03-26+):

1. POST `/mcp` — accepts JSON-RPC, can return `application/json` OR `text/event-stream`
2. GET `/mcp` — opens SSE stream for server-initiated messages (return 405 if not supported)
3. Session management via `Mcp-Session-Id` header on initialize response
4. Handle `Accept` header: if client includes `text/event-stream`, prefer SSE

For simplicity, start with JSON responses (current behavior is already compliant
for the basic case) and add GET returning 405.

## Files to modify

- `internal/adapters/inbound/mcp/server.go` — handle GET method on `/mcp`, Accept header

## Acceptance criteria

- POST `/mcp` continues working as before
- GET `/mcp` returns 405 Method Not Allowed
- Initialize response includes `Mcp-Session-Id` header
- Continue IDE `type: streamable-http` config works
