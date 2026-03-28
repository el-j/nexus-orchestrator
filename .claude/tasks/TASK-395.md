---
id: TASK-395
title: Add Playwright backlog promotion flow coverage
role: testing
planId: PLAN-056
status: todo
dependencies: [TASK-392]
createdAt: 2026-03-28T20:30:00Z
---

## Context

The backlog-to-queue flow is central to planning but still only covered below the browser layer. This task adds end-to-end coverage for creating a draft, promoting it, and seeing the no-provider warning path.

## Files to Read

- `frontend/e2e/`
- backlog and dashboard views involved in promotion

## Implementation Steps

1. Add a browser test for draft creation and promotion.
2. Assert the no-provider warning and resulting task visibility.
3. Keep the test isolated from real provider dependencies.
4. Reuse shared fixtures from the Playwright harness.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] A Playwright test covers draft → promote → warning visibility
- [ ] The flow passes without requiring a real LLM provider

## Anti-patterns to Avoid

- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
