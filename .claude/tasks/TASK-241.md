---
id: TASK-241
title: "VS Code extension: add queue and visibility smoke tests"
role: frontend
planId: PLAN-035
status: done
dependencies: [TASK-239]
createdAt: 2026-03-13T02:00:00.000Z
completedAt: 2026-03-13T08:31:20.000Z
---

## Context
The extension now has explicit queueing and route visibility, but it still has no automated safety net. The command/status logic needs a deterministic test suite independent of live Copilot.

## Files to Read
- `vscode-extension/package.json`
- `vscode-extension/src/commands/submitTask.ts`
- `vscode-extension/src/statusBar.ts`
- `vscode-extension/src/taskQueueProvider.ts`

## Implementation Steps

1. Add an extension test runner and coverage configuration.
2. Add tests for the explicit queue workflow, task provenance rendering, and route-aware status bar output.
3. Keep the suite independent of live VS Code services by mocking the `vscode` module.

## Acceptance Criteria
- [x] `go vet ./...` exits 0
- [x] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [x] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [x] `cd vscode-extension && npm run test:coverage` exits 0
- [x] Extension smoke tests cover explicit queueing and route visibility behavior

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging