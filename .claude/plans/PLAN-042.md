# PLAN-042 — "Here I Am" — AI Self-Discovery & How-To Beacon

**Status**: completed  
**Created**: 2026-03-14

## Context

When nexusOrchestrator starts, AIs connecting to it have no ambient knowledge of what it does, what tools it exposes, or how to collaborate with it. The user wants two things:

1. **A discovery beacon** — a well-known URL at `/.well-known/nexus.json` that any tool, agent, or client can GET immediately after connection to learn the server's identity, capabilities, and endpoints. Follows the pattern of `/.well-known/openid-configuration`.

2. **A howto endpoint** — `GET /api/howto` returns a rich, machine-readable JSON guide covering: what the orchestrator does, all HTTP endpoints, AI collaboration workflows (worker / planner / orchestrator roles), concrete curl examples. This is the "system prompt" for any AI that wants to work with nexus.

3. **MCP `howto` tool** — same guide delivered in-protocol. When an AI connects via MCP and calls `tools/list`, it sees `howto`. Calling it returns the full guide as text so the AI understands the system without needing HTTP access.

4. **MCP `initialize` instructions** — the `initialize` response embeds a short `instructions` hint in `serverInfo` pointing AIs to call `howto` first, following Anthropic Claude's MCP server guidance.

5. **Startup banner** — daemon and desktop startup logs print a clear "ready" block listing all endpoints, including the howto URL, so humans and AIs in logs also find the entry point.

## Port Reference (NEXUS T9 keypad: 63987)

| Service    | Port  | Mnemonic |
|------------|-------|----------|
| HTTP API   | 63987 | NEXUS    |
| MCP server | 63988 | NEXUS+1  |
| Vite dev   | 63989 | NEXUS+2  |

## Task Breakdown

| Task     | Title                                      | Role      |
|----------|--------------------------------------------|-----------|
| TASK-259 | HTTP howto endpoint + well-known beacon    | backend   |
| TASK-260 | MCP howto tool + enriched initialize       | backend   |
| TASK-261 | Tests: howto, well-known, MCP tool         | qa        |
| TASK-262 | Startup banner + copilot-instructions sync | backend   |

## Design Decisions

### `GET /api/howto` content structure

```json
{
  "name": "nexusOrchestrator",
  "version": "1.0.0",
  "description": "...",
  "quick_start": "...",
  "connection": {
    "http_api": "http://127.0.0.1:63987",
    "mcp_endpoint": "Use /.well-known/nexus.json for the MCP address",
    "dashboard": "http://127.0.0.1:63987/ui",
    "discovery": "http://127.0.0.1:63987/.well-known/nexus.json"
  },
  "ai_workflows": {
    "as_worker":       ["register session", "get queue", "claim task", "update status", "heartbeat", "deregister"],
    "as_planner":      ["create draft", "review backlog", "promote task", "update task"],
    "as_orchestrator": ["submit task", "monitor queue", "check task", "cancel task", "watch events"]
  },
  "http_endpoints": [{ "method": "POST", "path": "/api/tasks", "description": "..." }, ...],
  "examples": [{ "description": "...", "request": "curl ..." }, ...]
}
```

### `GET /.well-known/nexus.json` content structure

```json
{
  "schema_version": "1",
  "name": "nexusOrchestrator",
  "description": "Multi-LLM AI task orchestration server",
  "api": {
    "base_url": "http://HOST/api",
    "howto":    "http://HOST/api/howto",
    "health":   "http://HOST/api/health",
    "events":   "http://HOST/api/events"
  },
  "mcp": {
    "protocol": "json-rpc-2.0",
    "version":  "2024-11-05",
    "note":     "MCP server runs on a separate port — default :63988"
  },
  "capabilities": ["task-queue", "ai-session-registry", "provider-discovery", "llm-dispatch", "sse-events", "mcp-tools"]
}
```

### MCP `initialize` instructions field

The `serverInfo` in the `initialize` response gets an `instructions` string:

```
"You are connected to nexusOrchestrator — a multi-LLM AI task orchestration server. 
Call the 'howto' tool first to receive a complete integration guide. 
Use 'register_session' to identify yourself, 'get_queue' to see available tasks, 
and 'claim_task' to start working."
```

Claude Desktop, Cursor, and other MCP clients surface this as a system-level hint.

### Implementation files

- `internal/adapters/inbound/httpapi/howto.go` — new file: `buildHowToDoc()`, `handleHowto()`, `handleWellKnownNexus()`
- `internal/adapters/inbound/httpapi/server.go` — register two new routes
- `internal/adapters/inbound/mcp/server.go` — add `howto` to `toolList()`, add `case "howto"` dispatch, enrich `serverInfo`
- `main.go` + `cmd/nexus-daemon/main.go` — enhanced ready banner
- `internal/adapters/inbound/httpapi/howto_test.go` — new test file
