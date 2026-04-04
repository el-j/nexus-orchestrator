---
id: TASK-214
title: Fix VSCode ext memory leak + hardcoded URLs
role: backend
planId: PLAN-030
status: todo
dependencies: []
createdAt: 2025-07-25T00:00:00.000Z
---

## Context
Two issues in the VS Code extension: (1) When daemon URL changes via `onDidChangeConfiguration`, a new `NexusStatusBar` + polling instance is created and pushed to subscriptions, but the old instance is never disposed — causing accumulating pollers and duplicate API calls. (2) `EventSource` URLs in `useTasks.ts` and `useAISessions.ts` are hardcoded to `http://127.0.0.1:63987/api/events` instead of reading from configuration.

## Files to Read
- `vscode-extension/src/extension.ts` — lines ~62-69 (onDidChangeConfiguration handler)
- `vscode-extension/src/statusBar.ts` — polling lifecycle, dispose pattern
- `frontend/src/composables/useTasks.ts` — line ~59 (hardcoded EventSource URL)
- `frontend/src/composables/useAISessions.ts` — line ~36 (hardcoded EventSource URL)
- `vscode-extension/src/nexusClient.ts` — base URL configuration

## Implementation Steps
1. In `extension.ts` `onDidChangeConfiguration`: dispose the existing status bar instance before creating a new one. Call `statusBar.dispose()` or `statusBar.stopPolling()` before reassignment
2. Track status bar in a module-level variable rather than pushing each new instance to subscriptions array
3. In `useTasks.ts`: replace hardcoded `http://127.0.0.1:63987` with a configurable base URL from Wails runtime or environment. Consider a shared `useConfig()` composable that reads from Wails binding
4. In `useAISessions.ts`: same fix — use configurable base URL for EventSource
5. Add `onUnmounted` cleanup in both composables to close EventSource connections
6. Verify that status bar polling stops cleanly when extension deactivates

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] Changing daemon URL in VS Code settings creates only one poller (old one disposed)
- [ ] EventSource URLs are derived from configuration, not hardcoded
- [ ] EventSource connections are closed on component unmount

## Anti-patterns to Avoid
- NEVER leave event listeners or intervals without cleanup/disposal
- NEVER hardcode server URLs — always derive from configuration
- NEVER use `any` type in TypeScript — define proper interfaces
