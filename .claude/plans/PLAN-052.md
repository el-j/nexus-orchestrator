# PLAN-052: MCP Standards Compliance — SSE/Streamable HTTP/stdio Transports

## Goal

Make the MCP server implementation fully compliant with modern MCP standards so
it works correctly with Continue IDE, Claude Desktop, Cursor, Cline, and any
other MCP client using SSE, Streamable HTTP, or stdio transports.

## Critical Findings

### Current state

- Server only supports HTTP POST to `/mcp` returning `application/json`
- `.continue/mcpServers/new-mcp-server.yaml` references `/sse` endpoint that doesn't exist — **guaranteed failure**
- No legacy SSE transport (2024-11-05 spec: GET `/sse` → `endpoint` event → POST to returned URL)
- No Streamable HTTP transport (2025-03-26+: POST+GET on single `/mcp` endpoint)
- No stdio transport (subprocess mode via stdin/stdout)
- Missing standard MCP methods: `ping`, `resources/list`, `prompts/list`
- No Origin header validation (security requirement per MCP spec)
- No CORS headers for browser-based clients

### What clients expect

| Client         | Transport                         | Config format                        |
| -------------- | --------------------------------- | ------------------------------------ |
| Continue IDE   | `sse`, `stdio`, `streamable-http` | YAML in `.continue/mcpServers/`      |
| Claude Desktop | `stdio`                           | JSON in `claude_desktop_config.json` |
| Cursor         | `stdio`, `sse`                    | JSON in settings                     |
| Cline          | `stdio`, `sse`                    | JSON in settings                     |

## Waves

### Wave 1: Core Protocol Methods (TASK-356)

Add missing standard methods: `ping`, `resources/list`, `prompts/list`

### Wave 2: Legacy SSE Transport (TASK-357)

Add 2024-11-05 SSE transport: GET `/sse` opens EventSource, sends `endpoint` event, POST to that endpoint, responses via SSE `message` events

### Wave 3: Streamable HTTP Transport (TASK-358)

Upgrade `/mcp` endpoint to support Streamable HTTP: POST can return `text/event-stream`, GET opens SSE stream, `Mcp-Session-Id` header support

### Wave 4: Security & Headers (TASK-359)

Origin validation, CORS headers, Content-Type enforcement per spec

### Wave 5: stdio Transport Binary (TASK-360)

`cmd/nexus-mcp-stdio/main.go` — thin binary reading JSON-RPC from stdin, writing to stdout

### Wave 6: Fix Continue Config & Tests (TASK-361, TASK-362)

Fix `.continue/mcpServers/new-mcp-server.yaml`, add comprehensive tests for all transports

### Wave 7: Documentation & Orchestrator Update (TASK-363)

Update copilot-instructions.md, orchestrator.json, report results

## Task IDs

TASK-356, TASK-357, TASK-358, TASK-359, TASK-360, TASK-361, TASK-362, TASK-363
