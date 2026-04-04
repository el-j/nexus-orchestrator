---
id: TASK-240
title: "Frontend: add backlog/history smoke tests with coverage"
role: frontend
planId: PLAN-035
status: done
dependencies: [TASK-239]
createdAt: 2026-03-13T02:00:00.000Z
completedAt: 2026-03-13T08:31:20.000Z
---

## Context
The recent backlog/history regressions were frontend data-source mistakes and empty-array handling issues. Those paths need a dedicated smoke suite with coverage.

## Files to Read
- `frontend/package.json`
- `frontend/src/views/BacklogView.vue`
- `frontend/src/views/HistoryView.vue`
- `frontend/src/types/wails.ts`

## Implementation Steps

1. Add a frontend test runner and coverage configuration suitable for Vue 3.
2. Add smoke tests for BacklogView and HistoryView with mocked Wails/HTTP data.
3. Verify empty-array handling, correct fetch calls, and visible task rendering.

## Acceptance Criteria
- [x] `go vet ./...` exits 0
- [x] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [x] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [x] `cd frontend && npm run test:coverage` exits 0
- [x] BacklogView and HistoryView smoke tests cover the fixed data-source behavior

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging