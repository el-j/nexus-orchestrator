---
id: TASK-393
title: Add Playwright task lifecycle flow coverage
role: testing
planId: PLAN-056
status: todo
dependencies: [TASK-392]
createdAt: 2026-03-28T20:30:00Z
---

## Context

The highest-value browser flow is still untested: creating a task, watching it enter the queue, and cancelling it from the UI. This task adds end-to-end coverage for that core path.

## Files to Read

- `frontend/e2e/`
- `frontend/src/views/`
- task lifecycle endpoints used by the UI

## Implementation Steps

1. Add a browser test for submit-to-queue lifecycle.
2. Cover navigation, form submission, live queue visibility, and cancellation.
3. Keep fixtures isolated and deterministic.
4. Fold the test into the shared Playwright harness.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] A Playwright test covers submit → queue visibility → cancel
- [ ] The test passes headless without manual intervention

## Anti-patterns to Avoid

- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
