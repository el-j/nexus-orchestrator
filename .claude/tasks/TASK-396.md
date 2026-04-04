---
id: TASK-396
title: Add Playwright providers and discovery flow coverage
role: testing
planId: PLAN-056
status: done
dependencies: [TASK-392, TASK-398, TASK-399, TASK-400]
createdAt: 2026-03-28T20:30:00Z
---

## Context

Plan and provider discovery are visual, multi-step workflows that unit tests cannot fully protect. This task adds browser coverage once the discovery backend extensions land.

## Files to Read

- `frontend/e2e/`
- `frontend/src/views/DiscoveredPlansView.vue`
- providers and discovery endpoints

## Implementation Steps

1. Add a browser test for providers/discovery navigation and scan actions.
2. Assert the expected discovered plan and provider signals render in the UI.
3. Reuse fixtures that include extended discovery file kinds.
4. Keep the flow deterministic in headless runs.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] Playwright covers provider/discovery UI flows
- [ ] Extended discovery results appear in the browser assertions

## Anti-patterns to Avoid

- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
