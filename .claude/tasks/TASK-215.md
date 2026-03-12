---
id: TASK-215
title: Fix race conditions in VSCode session monitor
role: backend
planId: PLAN-030
status: todo
dependencies: [TASK-214]
createdAt: 2025-07-25T00:00:00.000Z
---

## Context
Two race conditions in the VS Code extension: (1) `SessionMonitor.heartbeat()` failure with 404 sets `sessionId = undefined` then calls `detectAndRegister()` without awaiting — concurrent heartbeats can create duplicate sessions. (2) `statusBar.ts` uses `Promise.all` for provider/task/session fetches — if any rejects, all fail silently to "offline" status, masking transient errors.

## Files to Read
- `vscode-extension/src/sessionMonitor.ts` — lines ~85-87 (heartbeat failure handling), full class
- `vscode-extension/src/statusBar.ts` — lines ~48-57 (Promise.all error handling)
- `vscode-extension/src/nexusClient.ts` — heartbeat method signature

## Implementation Steps
1. In `sessionMonitor.ts` `heartbeat()`: use an instance-level lock flag (`isReregistering`) to prevent concurrent re-registrations. Only call `detectAndRegister()` if not already in progress
2. Make the re-registration flow `await detectAndRegister()` to ensure completion before next heartbeat cycle
3. Add a debounce or guard so rapid consecutive 404s don't trigger multiple registrations
4. In `statusBar.ts`: replace `Promise.all()` with `Promise.allSettled()` — handle partial failures gracefully. Show "degraded" status if some APIs succeed but others fail
5. For each settled result, check `status === 'fulfilled'` before using the value
6. Reduce heartbeat logging verbosity — use debug-only output or remove "Heartbeat sent" messages

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] Concurrent heartbeat failures produce exactly one re-registration attempt
- [ ] Partial API failures show "degraded" rather than "offline" in status bar
- [ ] No "Heartbeat sent" noise in output channel during normal operation

## Anti-patterns to Avoid
- NEVER use `Promise.all` when partial failures should be tolerated — use `Promise.allSettled`
- NEVER leave race windows in async state mutations
- NEVER log at INFO level for every heartbeat — it's noise
