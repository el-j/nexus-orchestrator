---
id: TASK-387
title: Add bearer token middleware to the MCP server
role: mcp
planId: PLAN-056
status: done
dependencies: [TASK-386]
createdAt: 2026-03-28T20:30:00Z
---

## Context

The MCP server currently trusts only origin checks, which does not protect exposed or tunnelled deployments. This task adds a real bearer-token guard for MCP traffic when `NEXUS_MCP_TOKEN` is configured.

## Files to Read

- `internal/adapters/inbound/mcp/server.go`
- `internal/adapters/inbound/mcp/server_test.go`
- `internal/adapters/inbound/mcp/integration_test.go`

## Implementation Steps

1. Add configurable token auth middleware for MCP requests.
2. Allow unauthenticated access only when the token is unset.
3. Return `401` for missing or invalid tokens without breaking existing local-origin checks.
4. Add protocol-level tests for allowed and denied requests.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] MCP requests are rejected with `401` when `NEXUS_MCP_TOKEN` is set and the bearer token is absent or wrong
- [ ] Existing authenticated or token-disabled MCP flows still pass

## Anti-patterns to Avoid

- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
