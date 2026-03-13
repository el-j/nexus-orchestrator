---
id: TASK-232
title: Build verification — go test + vue-tsc + local Wails build
role: verify
planId: PLAN-032
status: todo
dependencies: [TASK-230, TASK-231]
createdAt: 2026-03-13T00:30:00.000Z
---

## Context
After fixing both the Go nil-slice issue and the frontend null-coalescing guards, the full build pipeline must be verified to confirm the app no longer crashes on startup with an empty database.

## Files to Read
- `.claude/plans/PLAN-032.md`
- `Makefile` (for build-gui target)

## Implementation Steps
1. Run `go vet ./...` — must exit 0
2. Run `CGO_ENABLED=1 go test -race -count=1 ./...` — must exit 0
3. Run `cd frontend && npx vue-tsc --noEmit` — must exit 0
4. Run `cd frontend && npx vite build` (or `make build-gui`) — must exit 0
5. Verify the Go repo methods: grep for `var tasks []domain` or `var sessions []domain` or `var configs []domain` — should find ZERO matches (all replaced with `tasks := []domain.Task{}` etc.)
6. Verify the frontend guards: grep for `await getQueue()` without `?? []` — should find ZERO unguarded patterns

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `npx vue-tsc --noEmit` exits 0
- [ ] No `var tasks []domain.Task` declarations remain (all use `tasks := []domain.Task{}`)
- [ ] No unguarded Wails array assignments remain (all use `?? []`)
- [ ] Built app starts without crash on empty database

## Anti-patterns to Avoid
- NEVER skip the type check — TypeScript errors in templates cause runtime crashes
- NEVER skip the go test — data race detection catches shared-state bugs
