---
id: TASK-237
title: "VS Code extension: add route visibility and activity log"
role: frontend
planId: PLAN-034
status: done
dependencies: [TASK-236]
createdAt: 2026-03-13T01:30:00.000Z
completedAt: 2026-03-13T01:50:00.000Z
---

## Context
Users can see the daemon and session state, but the extension does not clearly explain whether activity is a direct Copilot session, an MCP session, or a task that was explicitly queued through Nexus. The extension needs one coherent visibility layer.

## Files to Read
- `vscode-extension/src/activityLog.ts`
- `vscode-extension/src/sessionMonitor.ts`
- `vscode-extension/src/statusBar.ts`
- `vscode-extension/src/taskQueueProvider.ts`
- `vscode-extension/src/extension.ts`

## Implementation Steps

1. Create a shared output channel for Nexus activity and expose a command to show it.
2. Log queue submissions, task status transitions, and Copilot session registration/heartbeat failures with clear source prefixes.
3. Update the status bar and queue tree to surface route-aware information without making the primary UI noisy.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] The extension contributes a visible `Nexus: Show Activity Log` command
- [ ] The status bar tooltip differentiates VS Code/Copilot sessions from MCP sessions
- [ ] Queue items show local provenance when they were queued from the extension

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging