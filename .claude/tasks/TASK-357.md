# TASK-357: Add legacy SSE transport (2024-11-05 spec)

**Plan:** PLAN-052 | **Wave:** 2 | **Status:** done

## Description

Implement the 2024-11-05 HTTP+SSE transport for backwards compatibility with
Continue IDE `type: sse` and other clients expecting the legacy pattern:

1. GET `/sse` → opens SSE stream, first event is `endpoint` with POST URL
2. Client POSTs JSON-RPC messages to the endpoint URL
3. Server sends responses as SSE `message` events on the original stream

This is what Continue IDE requests when configured with `type: sse`.

## Implementation details

- Add SSE connection manager tracking client connections
- GET `/sse` handler: write `event: endpoint\ndata: /messages?sessionId=xxx\n\n`
- POST `/messages` handler: process JSON-RPC, send response via SSE stream
- Each SSE connection gets a unique session ID
- Proper cleanup on client disconnect

## Files to create/modify

- `internal/adapters/inbound/mcp/sse.go` — SSE transport implementation
- `internal/adapters/inbound/mcp/server.go` — register `/sse` and `/messages` routes

## Acceptance criteria

- Continue IDE `type: sse` config connects successfully
- Messages round-trip through SSE stream
- Graceful cleanup on disconnect
