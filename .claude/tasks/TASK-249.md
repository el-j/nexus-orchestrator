---
id: TASK-249
title: "Backend: persist provider promotion as enabled"
role: backend
planId: PLAN-037
status: done
dependencies: []
createdAt: 2026-03-13T10:17:51.000Z
---

## Context
Promoted providers appear live in-process but are not durably enabled in persisted configuration, so they disappear after restart. This task makes promotion persist an enabled config and adds regression coverage for restart durability.

## Files to Read
- `internal/core/services/orchestrator.go`
- `main.go`
- `cmd/nexus-daemon/main.go`
- `internal/core/services/orchestrator_test.go`

## Implementation Steps

1. Update provider promotion to persist enabled configuration without duplicate live registration.
2. Preserve the current runtime registration path while ensuring promoted providers survive restart-level reload semantics.
3. Add targeted tests for saved `Enabled: true` config and duplicate-registration avoidance.

## Acceptance Criteria
- [x] `go vet ./...` exits 0
- [x] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [x] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [x] Promoted providers are persisted with `Enabled: true`
- [x] Promotion tests cover durable persistence and no accidental duplicate live registration

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging

## Result
- Updated provider promotion so a promoted provider is persisted as `Enabled: true` through the provider config repository while preserving the live registration path.
- Added regression coverage ensuring promotion writes durable config and does not duplicate runtime registration when persistence is available.