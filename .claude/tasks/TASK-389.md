---
id: TASK-389
title: Add runtime config storage and /api/config endpoints
role: backend
planId: PLAN-056
status: done
dependencies: [TASK-386]
createdAt: 2026-03-28T20:30:00Z
---

## Context

Settings needed by the UI are still hardcoded or env-only, which prevents token management and accurate runtime display. This task introduces persisted runtime config plus HTTP read/write access for the settings screen.

## Files to Read

- `internal/core/ports/ports.go`
- `internal/core/services/orchestrator.go`
- `internal/adapters/outbound/repo_sqlite/`
- `internal/adapters/inbound/httpapi/server.go`

## Implementation Steps

1. Add a runtime-config persistence port and SQLite adapter.
2. Store queue-cap and token-related settings in a durable location.
3. Expose `GET /api/config` and `PUT /api/config` with safe response shapes.
4. Add backend and endpoint tests for persistence and masking behavior.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] Runtime config is persisted and reloadable across service instances
- [ ] `GET /api/config` returns the queue cap and token state needed by the UI

## Anti-patterns to Avoid

- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
