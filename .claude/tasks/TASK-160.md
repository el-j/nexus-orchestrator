---
id: TASK-160
title: Port interfaces for SystemScanner and discovery orchestration
role: architecture
planId: PLAN-023
status: todo
dependencies: [TASK-159]
createdAt: 2026-03-11T21:00:00.000Z
---

## Context
The system-wide provider discovery needs a clean port interface so the scanning logic (outbound adapter) is decoupled from the core services. The `Orchestrator` port must be extended with methods for the GUI/HTTP/MCP to query discovered providers, trigger scans, and promote a discovered provider to an active backend.

## Files to Read
- `internal/core/ports/ports.go` — existing port interfaces
- `internal/core/domain/provider.go` — `DiscoveredProvider` type from TASK-159

## Implementation Steps

1. In `internal/core/ports/ports.go`, add the `SystemScanner` outbound port:
   ```go
   // SystemScanner scans the local system for AI providers/agents.
   type SystemScanner interface {
       Scan(ctx context.Context) ([]domain.DiscoveredProvider, error)
   }
   ```

2. Extend the `Orchestrator` interface with 3 new methods:
   ```go
   GetDiscoveredProviders() ([]domain.DiscoveredProvider, error)
   TriggerScan(ctx context.Context) ([]domain.DiscoveredProvider, error)
   PromoteProvider(ctx context.Context, discoveredID string) error
   ```

3. Add stub implementations in `internal/core/services/orchestrator.go`:
   - `GetDiscoveredProviders()` → returns empty slice (scanner not yet wired)
   - `TriggerScan(ctx)` → returns empty slice
   - `PromoteProvider(ctx, id)` → returns `fmt.Errorf("orchestrator: promote provider: scanner not configured")`

4. Add a `WithSystemScanner(s ports.SystemScanner)` setter on `OrchestratorService` (same pattern as `WithProviderConfigRepo`).

5. Update all existing `Orchestrator` implementors (test mocks, CLI remote client) with the 3 new methods so compilation doesn't break.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `SystemScanner` interface exists in `ports.go` with `Scan(ctx) ([]DiscoveredProvider, error)`
- [ ] `Orchestrator` interface includes `GetDiscoveredProviders`, `TriggerScan`, `PromoteProvider`
- [ ] All existing test mocks compile with new methods

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER add concrete types to port interfaces — only use domain types and standard library types
