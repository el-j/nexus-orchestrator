---
id: TASK-238
title: "Verification: build Go, frontend views, and VS Code extension surfaces"
role: qa
planId: PLAN-034
status: done
dependencies: [TASK-236, TASK-237]
createdAt: 2026-03-13T01:30:00.000Z
completedAt: 2026-03-13T01:50:00.000Z
---

## Context
The backlog/history fix and the new VS Code extension routing work touch multiple surfaces. Verification needs to cover the Go build, the Wails/Vue frontend, and the extension bundle.

## Files to Read
- `frontend/src/views/BacklogView.vue`
- `frontend/src/views/HistoryView.vue`
- `vscode-extension/package.json`
- `vscode-extension/src/extension.ts`

## Implementation Steps

1. Run Go vet/build/test for the repository.
2. Run frontend type-checking for the Wails frontend.
3. Build the VS Code extension bundle and confirm the new commands are wired.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] `cd frontend && npx vue-tsc --noEmit` exits 0
- [ ] `cd vscode-extension && npm run build` exits 0

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging