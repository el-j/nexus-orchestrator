---
id: TASK-235
title: "Build verification: go test + vue-tsc + manual flow"
role: qa
planId: PLAN-033
status: done
dependencies: [TASK-233, TASK-234]
createdAt: 2026-03-13T01:10:00.000Z
completedAt: 2026-03-13T01:50:00.000Z
---

## Context
After fixing the data source mismatch, verify the full stack compiles, tests pass, and the draft→backlog flow works end-to-end.

## Verification Steps
1. `go vet ./...` — exits 0
2. `CGO_ENABLED=1 go test -race -count=1 ./...` — all green
3. `cd frontend && npx vue-tsc --noEmit` — exits 0
4. grep for remaining `useTasks` imports in BacklogView/HistoryView — should be 0
5. grep for `getQueue` usage in BacklogView/HistoryView — should be 0

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go test -race` passes
- [ ] `vue-tsc --noEmit` exits 0
- [ ] No remaining `useTasks` imports in BacklogView or HistoryView
