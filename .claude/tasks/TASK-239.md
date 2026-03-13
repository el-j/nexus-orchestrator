---
id: TASK-239
title: "QA: expand deterministic daemon E2E coverage"
role: qa
planId: PLAN-035
status: done
dependencies: []
createdAt: 2026-03-13T02:00:00.000Z
completedAt: 2026-03-13T08:31:20.000Z
---

## Context
The current smoke script checks only a shallow daemon happy path. The release gate must cover draft/backlog/history, AI sessions, and MCP parity without depending on real external providers.

## Files to Read
- `scripts/e2e-smoke.sh`
- `internal/adapters/inbound/httpapi/server.go`
- `internal/adapters/inbound/mcp/server.go`
- `internal/adapters/inbound/mcp/integration_test.go`

## Implementation Steps

1. Expand the daemon E2E flow to cover deterministic HTTP task lifecycle operations that do not depend on external providers.
2. Add AI session lifecycle checks and MCP tool parity checks for the same lifecycle.
3. Make the script fail fast with useful output and keep it portable for CI runners.

## Acceptance Criteria
- [x] `go vet ./...` exits 0
- [x] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [x] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [x] The daemon E2E script validates draft/backlog/history/all-tasks flows
- [x] The daemon E2E script validates AI session and MCP parity flows

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging