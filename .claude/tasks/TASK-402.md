---
id: TASK-402
title: Enrich MCP discovered plan responses
role: mcp
planId: PLAN-056
status: todo
dependencies: [TASK-398, TASK-399, TASK-400]
createdAt: 2026-03-28T20:30:00Z
---

## Context

The MCP `get_discovered_plans` tool still returns only the base file list, which limits downstream agent workflows. This task enriches the response with higher-level scan metadata so clients can reason about discovery results more intelligently.

## Files to Read

- `internal/adapters/inbound/mcp/tools.go`
- `internal/adapters/inbound/mcp/tools_test.go`
- plan discovery service and domain types

## Implementation Steps

1. Add richer scan metadata to the MCP discovered-plan response.
2. Include fields such as detected tools, active counts, and scan roots where appropriate.
3. Keep the response backward compatible for existing clients.
4. Extend tool tests to cover the new shape.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] `get_discovered_plans` returns enriched metadata alongside the discovered files
- [ ] Existing MCP clients continue to parse the response safely

## Anti-patterns to Avoid

- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
