---
id: TASK-305
title: Add 9 missing MCP tools — provider config CRUD, session deregister+heartbeat+purge, discovered-agents, delegate
role: mcp
planId: PLAN-047
status: todo
dependencies: []
createdAt: 2026-03-25T00:00:00.000Z
---

## Context

The 2026-03-25 audit found 9 `ports.Orchestrator` methods with no MCP tool. The MCP server is the primary interface for AI agents — these gaps mean agents cannot manage providers, sessions, or delegation via MCP.

Missing tools (current tool count: 20):

| Method                    | Tool name to add            |
| ------------------------- | --------------------------- |
| ListProviderConfigs       | list_provider_configs       |
| AddProviderConfig         | add_provider_config         |
| UpdateProviderConfig      | update_provider_config      |
| RemoveProviderConfig      | remove_provider_config      |
| DeregisterAISession       | deregister_ai_session       |
| HeartbeatAISession        | heartbeat_ai_session        |
| PurgeDisconnectedSessions | purge_disconnected_sessions |
| GetDiscoveredAgents       | get_discovered_agents       |
| DelegateToNexus           | delegate_to_nexus           |

Legacy methods (`RegisterCloudProvider`, `RemoveProvider`, `GetProviderModels`) are intentionally omitted — they are superseded by the config CRUD family.

## Files to Read

- `internal/adapters/inbound/mcp/server.go` — full file; study `toolList()`, `handleToolCall()`, and patterns like `toolClaimTask`, `toolRegisterSession`, `toolHeartbeatTask`
- `internal/core/ports/ports.go` — method signatures for the 9 methods

## Implementation Steps

For each tool, follow the exact same pattern as existing tools:

1. **Add a case to `handleToolCall()`** switch.
2. **Add a `tool*` handler method** on `*Server` that:
   - Parses `json.RawMessage` arguments with a local anonymous struct
   - Validates required fields (return `mcpError` if missing)
   - Calls the corresponding `s.orch.*` method
   - Returns `textResult(jsonMarshal(result))` or `textResult(`{"ok":true}`)` on success
3. **Add a `toolDef` entry to `toolList()`** with description and inputSchema.

### Tool specs

**list_provider_configs** — no required args; returns JSON array of ProviderConfig
**add_provider_config** — required: `kind` (string), `name` (string); optional: `base_url`, `api_key`, `enabled` (bool); returns saved ProviderConfig JSON
**update_provider_config** — required: `id` (string), `name` (string), `kind` (string); optional: same as add; returns updated ProviderConfig JSON
**remove_provider_config** — required: `id` (string); returns `{"ok":true}`
**deregister_ai_session** — required: `session_id` (string); returns `{"ok":true}`
**heartbeat_ai_session** — required: `session_id` (string); returns `{"ok":true}`
**purge_disconnected_sessions** — no required args; returns `{"purged": N}`
**get_discovered_agents** — no required args; returns JSON array of DiscoveredAgent
**delegate_to_nexus** — required: `session_id` (string); returns `{"instruction": "..."}`

## Acceptance Criteria

- [ ] `go vet ./internal/adapters/inbound/mcp/...` clean
- [ ] `go build ./cmd/nexus-daemon/...` exits 0
- [ ] `go test ./internal/adapters/inbound/mcp/... -race -count=1` all pass
- [ ] `toolList()` returns 29 tools (20 existing + 9 new)
- [ ] Tool count assertion in server_test.go updated to 29
