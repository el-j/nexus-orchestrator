---
id: TASK-407
plan: PLAN-057
status: todo
role: backend
wave: 1
---

# TASK-407: Enrich ProviderInfo with operational metadata

Extend the `ProviderInfo` struct in `internal/core/ports/ports.go` to include:

- `ContextLimit int` — from `LLMClient.ContextLimit()` (already exists on all adapters)
- `TimeoutSec int` — the HTTP client timeout configured per adapter
- `LastChecked time.Time` — when the discovery service last probed this provider
- `ConsecutiveFails int` — circuit-breaker state from discovery service

Populate these in `discovery.go` `ListProviders()`. This gives the frontend the data it needs to show provider health age, context windows, and circuit-breaker state.

**Files:** `internal/core/ports/ports.go`, `internal/core/services/discovery.go`
**Depends on:** none
