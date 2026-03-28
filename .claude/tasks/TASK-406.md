---
id: TASK-406
title: Update MCP client setup docs + howto tool text for VS Code Streamable HTTP
role: docs
planId: PLAN-056
status: done
dependencies: [TASK-404]
createdAt: 2026-03-28T22:00:00Z
---

## Problem

The user's VS Code `mcp.json` uses:

```json
"Nexus Orchestrator (SSE)": {
  "type": "sse",
  "url": "http://127.0.0.1:63988/sse"
}
```

The SSE transport works but is session-stateful — a daemon restart invalidates all
sessions causing 400 errors and the "terminated" log flood seen in the console. The
Streamable HTTP transport (`POST /mcp`) is stateless and reconnect-safe. VS Code's MCP
extension supports both; the `type: "http"` variant should be the recommended default.

Additionally, the in-product `howto` and `howto_brief` MCP tool text currently lacks any
VS Code-specific setup instructions, so users discover the SSE config by trial and error.

## Changes

### 1. `docs/mcp-integration.md`

Add a new **VS Code Setup** section before the Claude Desktop section:

````markdown
## VS Code Setup (Copilot + MCP extension)

Add to `~/Library/Application Support/Code/User/mcp.json` (macOS) or
`%APPDATA%\Code\User\mcp.json` (Windows):

```json
{
  "servers": {
    "Nexus Orchestrator": {
      "type": "http",
      "url": "http://127.0.0.1:63988/mcp"
    }
  }
}
```
````

> **Why `type: "http"` and not `type: "sse"`?**  
> The Streamable HTTP transport is stateless — each request is independent.
> The legacy SSE transport maintains a persistent connection that is invalidated
> when the daemon restarts, causing 400 errors and reconnect noise in the console.
> Use `type: "sse"` only if your MCP client requires the 2024-11-05 SSE transport
> (e.g. Continue IDE).

```

### 2. `internal/adapters/inbound/mcp/tools.go` — `toolHowto` and `toolHowtoBrief`

Add to the **Connection** section of both howto tools:

```

VS Code (GitHub Copilot / Copilot Chat):
mcp.json → servers → Nexus Orchestrator → { "type": "http", "url": "http://127.0.0.1:63988/mcp" }
Use type:"http" (Streamable HTTP) NOT type:"sse" — avoids 400 errors on daemon restart.

Legacy SSE (Continue IDE, Cursor, older clients):
{ "type": "sse", "url": "http://127.0.0.1:63988/sse" }
Sessions are invalidated on daemon restart — client will auto-reconnect after ~1 ping cycle.

stdio (Claude Desktop, any stdio-only client):
Use nexus-mcp-stdio binary as a subprocess bridge.

````

### 3. `/.well-known/nexus.json` response

Verify (or add) that the `GET /.well-known/nexus.json` discovery doc includes the
recommended transport type in the `mcp` section:

```json
{
  "mcp": {
    "url": "http://127.0.0.1:63988/mcp",
    "transport": "streamable-http",
    "legacy_sse_url": "http://127.0.0.1:63988/sse"
  }
}
````

## Files to Change

| File                                                                  | Change                                                    |
| --------------------------------------------------------------------- | --------------------------------------------------------- |
| `docs/mcp-integration.md`                                             | Add VS Code setup section with `type:"http"` example      |
| `internal/adapters/inbound/mcp/tools.go`                              | Update `howto` + `howto_brief` connection instructions    |
| `internal/adapters/inbound/httpapi/handlers_wellknown.go` (if exists) | Add `legacy_sse_url` field and note recommended transport |

## Acceptance Criteria

- [ ] `docs/mcp-integration.md` has VS Code Streamable HTTP setup as the primary recommendation
- [ ] `howto_brief` tool response mentions `type:"http"` for VS Code and warns against `type:"sse"` for reconnect scenarios
- [ ] `/.well-known/nexus.json` distinguishes between `url` (streamable) and `legacy_sse_url`
- [ ] User can copy-paste the VS Code config and connect without 400 errors across daemon restarts
