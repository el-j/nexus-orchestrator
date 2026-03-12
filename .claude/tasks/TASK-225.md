---
id: TASK-225
title: Fix discovery silent error + configurable constants
role: backend
planId: PLAN-030
status: todo
dependencies: [TASK-209]
createdAt: 2025-07-25T00:00:00.000Z
---

## Context
Three issues in discovery and orchestrator services: (1) `cachedHealth()` in `discovery.go` silently discards `GetAvailableModels()` error — initialization failures are invisible. (2) `maxRetries = 3`, `maxResponseTokens = 512`, `cleanupInterval = 2*time.Minute`, and `staleThreshold = 5*time.Minute` are all hardcoded constants with no way to configure them. (3) `context.Background()` is used in `runSessionCleanup()` instead of a derived context, bypassing graceful shutdown.

## Files to Read
- `internal/core/services/discovery.go` — line ~240 (`cachedHealth`, `GetAvailableModels` error discarded)
- `internal/core/services/orchestrator.go` — line ~30 (`maxRetries`), ~961 (`maxResponseTokens`), ~838-839 (cleanup intervals)
- `internal/core/services/orchestrator.go` — line ~859 (`context.Background()` in cleanup loop)

## Implementation Steps
1. In `discovery.go` `cachedHealth()`: check `GetAvailableModels()` error — if error, log with `log.Printf("discovery: get models from %s: %v", name, err)` and leave models empty (don't set error models)
2. In `orchestrator.go`: promote `maxRetries`, `maxResponseTokens`, `cleanupInterval`, `staleThreshold` to builder-pattern options:
   - `WithMaxRetries(n int) Option`
   - `WithMaxResponseTokens(n int) Option`
   - `WithCleanupInterval(d time.Duration) Option`
   - `WithStaleThreshold(d time.Duration) Option`
   - Keep current values as defaults
3. In `runSessionCleanup()`: replace `context.Background()` with a context derived from the service's stop channel. Use `context.WithCancel` and cancel it when `stopCh` is closed, or accept a context parameter
4. Add test cases for custom option values — verify they override defaults
5. Document the new options in the function's godoc

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `GetAvailableModels` errors are logged, not silently discarded
- [ ] `maxRetries`, `maxResponseTokens`, cleanup intervals are configurable via builder options
- [ ] Session cleanup respects graceful shutdown (derived context, not Background)
- [ ] Existing behavior unchanged when using defaults

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use `context.Background()` in goroutines that should respect shutdown
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
