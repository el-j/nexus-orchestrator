---
id: TASK-210
title: Add constructor nil-validation in NewOrchestrator
role: backend
planId: PLAN-030
status: todo
dependencies: []
createdAt: 2025-07-25T00:00:00.000Z
---

## Context
`NewOrchestrator()` in `orchestrator.go` does not validate that required dependencies (`discovery`, `repo`, `writer`) are non-nil. A nil dependency would cause a panic deep in the worker loop rather than failing fast at construction time. Similarly, `PromoteProvider()` assumes `found.BaseURL` is populated without validation.

## Files to Read
- `internal/core/services/orchestrator.go` — lines 55-70 (`NewOrchestrator` constructor)
- `internal/core/services/orchestrator.go` — lines 1005-1015 (`PromoteProvider` field access)
- `internal/core/services/discovery.go` — constructor (if any)

## Implementation Steps
1. Add nil checks for `discovery`, `repo`, and `writer` in `NewOrchestrator()`. Panic with descriptive message: `panic("orchestrator: discovery service is required")` — this is a programming error, not a runtime condition
2. Add validation in `PromoteProvider()` for `found.BaseURL == ""` — return `fmt.Errorf("orchestrator: promote provider: discovered provider has no base URL")`
3. Add nil check for `DiscoveryService` constructor dependencies if applicable
4. Add test case: `TestNewOrchestrator_NilDeps` verifying panic on nil required deps
5. Add test case: `TestPromoteProvider_EmptyBaseURL` verifying error on empty BaseURL

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `NewOrchestrator(nil, repo, writer)` panics with descriptive message
- [ ] `PromoteProvider` with empty BaseURL returns wrapped error
- [ ] New tests pass with `-race` flag

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER return error from constructor for programming errors — use panic
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
