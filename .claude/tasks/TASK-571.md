---
id: TASK-571
title: MCP tool tests — provider config CRUD, session lifecycle, and howto tools
role: qa
planId: PLAN-071
status: todo
dependencies: [TASK-555]
createdAt: 2026-04-15T00:00:00Z
---

## Context

Eleven MCP tools have zero test coverage: provider config CRUD (add, update, remove, list), session lifecycle (heartbeat_ai_session, deregister_ai_session, purge_disconnected_sessions), discovered agents, discover_providers, and howto/howto_brief. These tools are the primary interface for AI agents — silent regressions here break agent workflows entirely.

## Files to Read

- `internal/adapters/inbound/mcp/tools.go`
- `internal/adapters/inbound/mcp/tools_test.go` (existing tests — study the harness pattern)
- `internal/adapters/inbound/mcp/server_test.go` (existing mock setup)

## Implementation Steps

1. Extend `tools_test.go` `toolHarnessOrch` mock with stub methods for:
   - `AddProviderConfig`, `UpdateProviderConfig`, `RemoveProviderConfig`, `ListProviderConfigs`
   - `HeartbeatAISession`, `DeregisterAISession`, `PurgeDisconnectedSessions`
   - `GetDiscoveredAgents`
2. Add tests for provider config CRUD tools:
   - `TestToolListProviderConfigs_ReturnsAll` — happy path
   - `TestToolAddProviderConfig_RequiresKindAndName` — missing fields → `-32602`
   - `TestToolUpdateProviderConfig_RequiresID` — missing id → `-32602`
   - `TestToolRemoveProviderConfig_ForwardsID`
3. Add tests for session lifecycle tools:
   - `TestToolHeartbeatAISession_RequiresSessionID`
   - `TestToolDeregisterAISession_RequiresSessionID`
   - `TestToolPurgeDisconnectedSessions_ReturnsCount`
4. Add tests for discovery and howto tools:
   - `TestToolGetDiscoveredAgents_ReturnsAgents`
   - `TestToolHowto_ContainsRegisterSessionStep` — response text contains "register_session"
   - `TestToolHowtoBrief_ContainsKeywords` — response contains "claim_task"

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./internal/adapters/inbound/mcp/...` exits 0
- [ ] All 11 previously-untested tools now have at least one test
- [ ] Missing-argument cases return `-32602` invalid params (not panic, not 200 with empty data)
- [ ] `toolHowtoBrief` test verifies key workflow keywords are present in the response

## Anti-patterns to Avoid

- NEVER test implementation details — test behaviour through the JSON-RPC interface
- NEVER use `time.Sleep` in tests
