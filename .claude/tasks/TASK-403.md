---
id: TASK-403
title: Add periodic background plan scanning
role: backend
planId: PLAN-056
status: done
dependencies: [TASK-398, TASK-399, TASK-400]
createdAt: 2026-03-28T20:30:00Z
---

## Context

Plan discovery is currently refresh-only, so the UI goes stale until a manual scan is triggered. This task adds a bounded background refresh loop so discovery data stays reasonably fresh without user intervention.

## Files to Read

- `internal/core/services/orchestrator.go`
- discovery or scan-related adapters and services
- lifecycle wiring for server startup and shutdown

## Implementation Steps

1. Add a periodic plan-scan refresh on a bounded interval.
2. Keep goroutine lifecycle in the inbound/runtime layer rather than core business logic.
3. Reuse existing scanner functionality and cache updated results safely.
4. Add tests or targeted verification for refresh behavior and shutdown safety.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] Plan discovery refreshes periodically without manual API calls
- [ ] Background scanning shuts down cleanly with the hosting service

## Anti-patterns to Avoid

- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
