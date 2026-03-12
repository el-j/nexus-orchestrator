---
id: TASK-169
title: MCP tools for provider discovery
role: mcp
planId: PLAN-023
status: todo
dependencies: [TASK-166]
createdAt: 2026-03-11T21:00:00.000Z
---

## Context
External MCP clients (Claude Desktop, VS Code extension, etc.) need to query and interact with the system-wide provider discovery via MCP JSON-RPC 2.0. Two new tools: `discover_providers` (trigger scan + return results) and `promote_provider` (make a discovered provider active).

## Files to Read
- `internal/adapters/inbound/mcp/server.go` — existing MCP server, tool registration pattern
- `internal/core/ports/ports.go` — `Orchestrator` interface with discovery methods
- `internal/core/domain/provider.go` — `DiscoveredProvider` type

## Implementation Steps

1. In `internal/adapters/inbound/mcp/server.go`, register 2 new tools in `registerTools()`:

2. **Tool: `discover_providers`**
   - Description: "Scan the local system for installed AI providers/agents and return discovered results"
   - Input schema: empty object `{}` (no parameters needed)
   - Handler: call `s.orch.TriggerScan(ctx)`, return JSON array of `DiscoveredProvider`
   - On error, return MCP error response

3. **Tool: `promote_provider`**
   - Description: "Promote a discovered provider to an active LLM backend"
   - Input schema: `{ "id": { "type": "string", "description": "ID of the discovered provider to promote" } }`
   - Required: `["id"]`
   - Handler: call `s.orch.PromoteProvider(ctx, id)`
   - On success, return `{"promoted": true, "id": "..."}`
   - On `domain.ErrNotFound`, return MCP error with code -32602 (invalid params)

4. Update the MCP `initialize` response to include the 2 new tools in the capabilities list (total: 8 tools).

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `discover_providers` tool returns JSON array of discovered providers
- [ ] `promote_provider` tool promotes a reachable provider and returns success
- [ ] MCP `initialize` response lists 8 tools total
- [ ] Error responses use correct JSON-RPC 2.0 error codes

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER break existing MCP tool handlers — only add new ones
- NEVER use goroutines inside `internal/core/services/`
