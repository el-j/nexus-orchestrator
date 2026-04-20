---
id: TASK-564
title: Fix VS Code memory leaks and unsafe error handling
role: vscode
planId: PLAN-071
status: todo
dependencies: [TASK-556]
createdAt: 2026-04-15T00:00:00Z
---

## Context

Four memory leaks: (1) `buildClient()` in `extension.ts` creates a second `NexusStatusBar` + starts a second poller before the first promise resolves — two pollers race on startup. (2) `onDidChangeConfiguration` re-creates StatusBar + pushes another `startPolling()` disposable on every config change, accumulating entries. (3) `nexus.reconnect` accumulates non-disposable poller entries on repeated runs. (4) `sessionMonitor.ts` heartbeat and claim timers are raw `setInterval` handles not placed in `context.subscriptions`. Five unsafe error patterns: `(err as Error)` casts in `delegateToNexus.ts`, unawaited `.then()` chains, and unguarded `executeCommand` calls.

## Files to Read

- `vscode-extension/src/extension.ts`
- `vscode-extension/src/sessionMonitor.ts`
- `vscode-extension/src/commands/delegateToNexus.ts`
- `vscode-extension/src/workspaceOrchView.ts`

## Implementation Steps

1. `extension.ts` — fix double-poller race: make `buildClient()` synchronous or use a mutex flag. Ensure only one `NexusStatusBar` instance exists at any time; dispose the old one before creating a new one.
2. `extension.ts` — in `onDidChangeConfiguration`: dispose the previous statusBar before creating a new one; do NOT push new `startPolling()` disposables — instead call `statusBar.updateUrl(newUrl)` or equivalent on the existing instance.
3. `extension.ts` — in `nexus.reconnect` handler: same fix — reuse or properly dispose the existing statusBar rather than accumulating new ones in `context.subscriptions`.
4. `sessionMonitor.ts` — convert `heartbeatTimer` and `claimTimer` from raw `ReturnType<typeof setInterval>` to `vscode.Disposable` wrappers and push them into the `stop()` cleanup path; ensure `deactivate()` calls `monitor.stop()`.
5. `delegateToNexus.ts:57` — replace `(err as Error).message` with `err instanceof Error ? err.message : String(err)`
6. `delegateToNexus.ts:76,96` — `await` the `vscode.window.showInformationMessage()` calls; wrap in try-catch
7. `delegateToNexus.ts:119` — `await vscode.commands.executeCommand(...)` and catch the result
8. `workspaceOrchView.ts:251` — replace floating `.then()/.catch(() => {})` with `async/await` and proper error surface (log via `outputChannel`)

## Acceptance Criteria

- [ ] `npm run compile` exits 0 in `vscode-extension/`
- [ ] `npm test` exits 0 in `vscode-extension/`
- [ ] No second poller is created during the async `buildClient()` window
- [ ] `onDidChangeConfiguration` does not accumulate StatusBar instances in subscriptions
- [ ] `sessionMonitor` timers are cleaned up via `stop()`
- [ ] All `(err as Error)` casts replaced with `instanceof Error` guards

## Anti-patterns to Avoid

- NEVER push the same disposable type to `context.subscriptions` more than once without disposing the previous one
- NEVER use `console.log` in the extension — use the VS Code output channel
