---
id: TASK-236
title: "VS Code extension: add explicit send-current-context queue workflow"
role: frontend
planId: PLAN-034
status: done
dependencies: []
createdAt: 2026-03-13T01:30:00.000Z
completedAt: 2026-03-13T01:50:00.000Z
---

## Context
The extension already knows how to queue Nexus tasks, but the current command surface does not clearly distinguish explicit queueing from normal Copilot chat. Users need a deliberate queue workflow that reviews context before submission.

## Files to Read
- `vscode-extension/package.json`
- `vscode-extension/src/extension.ts`
- `vscode-extension/src/commands/submitTask.ts`
- `vscode-extension/src/commands/index.ts`

## Implementation Steps

1. Add a new `nexus.sendCurrentContext` command and expose it in the command palette, editor UI, and task queue view actions.
2. Rework the submission flow so users enter an instruction, select context files intentionally, choose a route, confirm the queue action, and then submit.
3. Keep the existing manual submission path available under a clearer label so advanced users still have a direct fallback.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] The extension contributes a visible `Nexus: Send Current Context To Queue` command
- [ ] The queue workflow lets users review context files before submission

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging