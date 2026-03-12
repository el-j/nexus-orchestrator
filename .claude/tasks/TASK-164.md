---
id: TASK-164
title: OrchestratorService discovery and promote methods
role: backend
planId: PLAN-023
status: todo
dependencies: [TASK-160, TASK-161]
createdAt: 2026-03-11T21:00:00.000Z
---

## Context
The `OrchestratorService` stubs from TASK-160 need real implementations now that the `SystemScanner` adapter exists. The service must cache the latest scan results, expose them via the port methods, and enable "promoting" a discovered provider into the active `DiscoveryService` (and optionally persisting it as a `ProviderConfig`).

## Files to Read
- `internal/core/services/orchestrator.go` — stub methods from TASK-160
- `internal/core/services/discovery.go` — `DiscoveryService` and `RegisterProvider`
- `internal/core/ports/ports.go` — `SystemScanner`, `Orchestrator` interfaces
- `internal/core/domain/provider.go` — `DiscoveredProvider`, `ProviderConfig`

## Implementation Steps

1. Add a `scanner ports.SystemScanner` field and `lastScan []domain.DiscoveredProvider` cache (protected by `sync.RWMutex`) to `OrchestratorService`.

2. Implement `WithSystemScanner(s ports.SystemScanner)` setter:
   ```go
   func (o *OrchestratorService) WithSystemScanner(s ports.SystemScanner) {
       o.mu.Lock()
       defer o.mu.Unlock()
       o.scanner = s
   }
   ```

3. Implement `GetDiscoveredProviders()`:
   - Return `o.lastScan` under read lock
   - If `lastScan` is nil and scanner is configured, trigger a scan (non-blocking, return empty for now)

4. Implement `TriggerScan(ctx)`:
   - If `o.scanner == nil`, return `fmt.Errorf("orchestrator: trigger scan: scanner not configured")`
   - Call `o.scanner.Scan(ctx)`, store results in `o.lastScan` under write lock
   - Return results

5. Implement `PromoteProvider(ctx, discoveredID)`:
   - Find the discovered provider by ID in `o.lastScan`
   - If not found, return `domain.ErrNotFound`
   - If `Status != DiscoveryStatusReachable`, return error (can only promote API-reachable providers)
   - Map `DiscoveredProvider` → `domain.ProviderConfig` (kind, baseURL, name)
   - Call `o.RegisterCloudProvider(cfg)` to add to the live `DiscoveryService`
   - If `o.providerConfigRepo != nil`, also persist via `o.AddProviderConfig(ctx, cfg)`

6. Ensure all error returns use `fmt.Errorf("orchestrator: ...: %w", err)` wrapping.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `GetDiscoveredProviders()` returns cached scan results
- [ ] `TriggerScan()` invokes the scanner and updates cache
- [ ] `PromoteProvider()` converts a reachable discovered provider into an active backend
- [ ] Thread-safety: all access to `lastScan` is mutex-protected

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — scanning is invoked by inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER auto-register discovered providers without explicit user action (promote is opt-in)
