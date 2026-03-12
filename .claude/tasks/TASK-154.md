---
id: TASK-154
title: MCP — register_session and get_ai_sessions tools
role: mcp
planId: PLAN-022
status: todo
dependencies: [TASK-153]
priority: high
estimated_effort: M
createdAt: 2026-03-12T11:00:00.000Z
---

## Goal
Add two new tools to the MCP JSON-RPC 2.0 server — `register_session` and `get_ai_sessions` — so that external AI clients (Claude Desktop, Cursor, any MCP-capable AI tool) can announce themselves to nexusOrchestrator and query the active session list.

## Context
`internal/adapters/inbound/mcp/server.go` currently has 6 tools implemented using a dispatcher pattern (a `switch` on tool name). The server depends only on `ports.Orchestrator` — it never imports services directly. All new tools must follow the identical pattern:

**Existing tool pattern (submit_task excerpt):**
```json
{
  "name": "submit_task",
  "description": "...",
  "inputSchema": { "type": "object", "properties": {...}, "required": [...] }
}
```
Handler reads from `params` map, calls `s.orch.SomeMethod(...)`, returns JSON result.

**New MCP source constant**: when `register_session` is called via MCP, the `Source` field must be set to `domain.SessionSourceMCP` ("mcp").

## Scope

### Files to modify
- `internal/adapters/inbound/mcp/server.go` — add 2 tools to the tools list and 2 case branches in the dispatcher

### Files to modify (tests)
- `internal/adapters/inbound/mcp/server_test.go` — add table-driven test cases for both new tools

## Implementation Steps

### 1. Add tools to the `listTools` response
In the `tools` slice returned by the `tools/list` method, append two new tool descriptors:

**`register_session`**:
```json
{
  "name": "register_session",
  "description": "Announce this AI agent session to nexusOrchestrator for visualisation and orchestration. Call this once when the agent starts, and periodically (heartbeat) to update last_activity.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "agent_name": { "type": "string", "description": "Human-readable name of this AI agent (e.g. 'Claude Desktop', 'GitHub Copilot')" },
      "project_path": { "type": "string", "description": "Absolute path of the project this agent is working on (optional)" },
      "external_id": { "type": "string", "description": "Caller-provided correlation token for deduplication (optional)" }
    },
    "required": ["agent_name"]
  }
}
```

**`get_ai_sessions`**:
```json
{
  "name": "get_ai_sessions",
  "description": "Return the list of all known external AI agent sessions registered with this nexusOrchestrator instance.",
  "inputSchema": { "type": "object", "properties": {} }
}
```

### 2. Add dispatcher cases

**`register_session` handler**:
- Extract `agent_name` (required), `project_path` (optional string), `external_id` (optional string) from `params`
- Validate `agent_name` non-empty → return JSON-RPC error code -32602 (InvalidParams) with message "agent_name is required"
- Build `domain.AISession{AgentName: ..., Source: domain.SessionSourceMCP, ProjectPath: ..., ExternalID: ...}`
- Call `s.orch.RegisterAISession(r.Context(), session)`
- Return `{"session_id": session.ID, "status": "registered"}`

**`get_ai_sessions` handler**:
- Call `s.orch.ListAISessions(r.Context())`
- Return the sessions slice serialised as JSON (return `[]` if nil)

### 3. Tests (server_test.go)
Add test cases using `httptest.NewServer(mcpServer)`:
- `register_session` with missing `agent_name` → error -32602
- `register_session` valid → response has `session_id`
- `get_ai_sessions` → response is JSON array

Mock orchestrator for MCP tests must implement the two new methods (add stub to existing test mock).

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./internal/adapters/inbound/mcp/...` exits 0
- [ ] `register_session` tool appears in `tools/list` response
- [ ] `get_ai_sessions` tool appears in `tools/list` response
- [ ] `register_session` with missing `agent_name` returns JSON-RPC error -32602
- [ ] `register_session` valid call returns `{"session_id": "..."}` in `result`
- [ ] Tool count in `tools/list` response is 8 (was 6, +2 new)

## Anti-patterns to Avoid
- NEVER import domain types directly in MCP handler logic — only through `ports.Orchestrator` method calls
- NEVER skip input validation — MCP clients send arbitrary JSON
- NEVER return HTTP 200 with an error in the JSON body — use JSON-RPC error response format
- NEVER add new tool schemas that are inconsistent with the existing tool schema style
