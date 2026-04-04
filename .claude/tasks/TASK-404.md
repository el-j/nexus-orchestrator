---
id: TASK-404
title: Fix SSE session 400 on daemon restart — add reconnect resilience + ping keepalive
role: backend
planId: PLAN-056
status: done
dependencies: []
createdAt: 2026-03-28T22:00:00Z
---

## Problem

Observed in VS Code MCP extension logs:

```
Error reading SSE stream: TypeError: terminated          ← daemon restarted, TCP closed
Stopping server Nexus Orchestrator (SSE)                 ← client detected disconnect
Starting server Nexus Orchestrator (SSE)                 ← client reconnects
400 status sending message to /sse, will attempt to fall back to legacy SSE
Failed to parse message: ""                              ← empty 400 body confuses client
```

**Root cause (two bugs):**

1. **Post-restart session loss.** The SSE session map lives in memory. When the daemon restarts,
   all `sseSession` entries are gone. The MCP client (Continue IDE / VS Code extension) caches
   the `sessionId` from the `endpoint` event and immediately POSTs to
   `/messages?sessionId=<old-id>`. With no session, `handleSSEMessage` returns `400 "session not found"`.
   The HTTP body is the string `"session not found\n"`, but the client receives an empty 400
   (because the Origin check or SSE reconnect race means the request is rejected before the handler
   runs). The `Failed to parse message: ""` confirms an empty body is arriving.

2. **No SSE ping/keepalive.** SSE connections over HTTP/1.1 have no built-in heartbeat.
   Without periodic `:` comment events, cloud proxies and VS Code's fetch implementation
   close idle long-poll connections after ~60s, causing spurious `terminated` errors.

## Current Code Location

- `internal/adapters/inbound/mcp/sse.go` — `handleSSE`, `handleSSEMessage`, `sseManager`
- `internal/adapters/inbound/mcp/server.go` — `StartMCPServer` (ReadTimeout set to 15s — kills idle SSE!)

## Fix Strategy

### Fix A — Remove ReadTimeout from SSE server (critical)

`StartMCPServer` currently sets `ReadTimeout: 15 * time.Second` which terminates any SSE
connection sitting idle for 15 seconds. This must be 0 (disabled) for the SSE path.
Pattern: use `http.TimeoutHandler` only for the `/mcp` POST endpoint, not for `/sse`.

### Fix B — Send SSE `:ping` comment every 15s

Inside `handleSSE`, start a ticker after the `endpoint` event is sent:

```go
ticker := time.NewTicker(15 * time.Second)
defer ticker.Stop()
for {
    select {
    case <-ticker.C:
        fmt.Fprintf(w, ": ping\n\n")
        flusher.Flush()
    case <-r.Context().Done():
        return
    case <-session.done:
        return
    }
}
```

SSE comment lines (starting with `:`) are valid per RFC and ignored by clients — they exist
solely to keep the TCP connection alive and prevent proxy timeouts.

### Fix C — Return structured error JSON on 400 (not plain text)

`handleSSEMessage` currently calls `http.Error(w, "session not found", http.StatusNotFound)`.
VS Code MCP extension's fallback logic expects any non-2xx response to carry an empty body
(it discards the body). However an empty body trips the `Failed to parse message: ""` log.
Fix: return `204 No Content` (no body) instead of 400/404 when a session is not found,
allowing the client to reconnect cleanly rather than treating it as a parse error.
**Or**: return a valid JSON-RPC error in the body with `Content-Type: application/json` so
the client can parse it instead of erroring.

### Fix D — Update VS Code `mcp.json` docs to prefer Streamable HTTP

The user's VS Code `mcp.json` uses `type: "sse"` pointing to `/sse`. The Streamable HTTP
transport (`POST /mcp`) is stateless and reconnect-safe — no session map, no 400 on restart.
The docs and `howto` tool text should recommend:

```json
{
  "Nexus Orchestrator": {
    "type": "http",
    "url": "http://127.0.0.1:63988/mcp"
  }
}
```

Legacy SSE stays for Continue IDE and older clients that require it.

## Files to Change

| File                                      | Change                                                                                                                            |
| ----------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `internal/adapters/inbound/mcp/server.go` | Remove `ReadTimeout: 15s` from `StartMCPServer` (already 0 in comment but coded otherwise); wrap `/mcp` handler with timeout only |
| `internal/adapters/inbound/mcp/sse.go`    | Add 15s ping ticker to `handleSSE`; fix `handleSSEMessage` to return JSON-RPC error or 204 on missing session                     |
| `docs/mcp-integration.md`                 | Add VS Code `mcp.json` example with `type: "http"` pointing to `/mcp`                                                             |
| `internal/adapters/inbound/mcp/tools.go`  | Update `howto` and `howto_brief` text with corrected VS Code setup                                                                |

## Acceptance Criteria

- [ ] `go build ./...` exits 0
- [ ] `CGO_ENABLED=1 go test -race ./internal/adapters/inbound/mcp/...` exits 0
- [ ] SSE connections no longer receive `ReadTimeout` termination (verified by holding an SSE connection open 15+ s in test)
- [ ] `GET /sse` sends `: ping` events at ~15s intervals (verified in new test)
- [ ] `POST /messages?sessionId=nonexistent` returns either `204 No Content` or a parseable JSON-RPC error — NOT an empty 400
- [ ] `docs/mcp-integration.md` includes VS Code Streamable HTTP example
