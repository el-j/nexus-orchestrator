---
id: TASK-156
title: VS Code extension — Copilot session detection and registration
role: cli
planId: PLAN-022
status: todo
dependencies: [TASK-153]
priority: high
estimated_effort: M
createdAt: 2026-03-12T11:00:00.000Z
---

## Goal
Extend the nexus VS Code extension to detect active GitHub Copilot chat activity and register it as an `AISession` with the nexusOrchestrator daemon, turning the extension into the VS Code leg of the universal session discovery system.

## Context
`vscode-extension/src/` currently has:
- `extension.ts` — `activate()`, 7 registered commands, no `vscode.lm.*` API use
- `nexusClient.ts` — HTTP client for `http://127.0.0.1:9999`
- `statusBar.ts` — polling status bar
- No existing session registration logic

**VS Code Language Model API** (available from VS Code 1.90+):
- `vscode.lm.selectChatModels({ vendor: 'copilot' })` — returns array of available chat models
- When non-empty, Copilot is active and available
- `vscode.lm.onDidChangeChatModels` — event fired when available models change

**Target daemon endpoint**: `POST /api/ai-sessions` with body `{ "agentName": "GitHub Copilot", "source": "vscode", "projectPath": "<workspace root>" }`

**TypeScript type** to add to `nexusClient.ts`:
```typescript
export interface AISession { id: string; agentName: string; source: string; status: string; lastActivity: string }
export interface RegisterSessionRequest { agentName: string; source: 'vscode' | 'mcp' | 'http'; projectPath?: string; externalId?: string }
```

## Scope

### Files to modify
- `vscode-extension/src/nexusClient.ts` — add `registerSession(req)` and `deregisterSession(id)` methods
- `vscode-extension/src/extension.ts` — on activate: detect Copilot, register session; on deactivate: call deregister
- `vscode-extension/src/statusBar.ts` — append session count to tooltip

### Files to create
- `vscode-extension/src/sessionMonitor.ts` — encapsulates Copilot detection and registration lifecycle

## Implementation Steps

### 1. nexusClient.ts — add session HTTP methods
- `registerSession(req: RegisterSessionRequest): Promise<AISession>` — POST to `/api/ai-sessions`
- `deregisterSession(id: string): Promise<void>` — DELETE to `/api/ai-sessions/{id}`

### 2. sessionMonitor.ts — session lifecycle manager
Export class `SessionMonitor`:
- Constructor: takes `NexusClient` instance and VS Code `ExtensionContext`
- `start()`: 
  - Query `vscode.lm.selectChatModels({ vendor: 'copilot' })` 
  - If models returned (Copilot available): call `nexusClient.registerSession(...)` with workspace root as `projectPath`
  - Store returned `sessionId` in context `workspaceState`
  - Register `vscode.lm.onDidChangeChatModels` listener: re-check and re-register if Copilot models change
  - Set up a heartbeat `setInterval` (every 60 s): call `registerSession` again with same `externalId` to update `lastActivity`
- `stop()`:
  - Clear the heartbeat interval
  - If `sessionId` stored: call `nexusClient.deregisterSession(sessionId)`
  - Clear `sessionId` from workspaceState
- Handle `isDaemonUnreachable()` gracefully: if daemon is offline, silently skip — do NOT show error toasts for session registration failures (non-critical)

**`externalId`** for heartbeat deduplication: use `vscode.env.machineId + ':' + workspaceFolderUri` as a stable key.

### 3. extension.ts — wire SessionMonitor
- Import `SessionMonitor` from `./sessionMonitor`
- In `activate()`: create `new SessionMonitor(getClient(), context)` and call `monitor.start()`
- Store monitor instance; in `deactivate()` call `monitor.stop()`
- Add command `nexus.showAISessions` (already listed in package.json sidebar design — add here) that calls `vscode.commands.executeCommand('nexus.taskQueue.focus')` as placeholder OR opens the Nexus UI URL `http://127.0.0.1:9999/ui`

### 4. statusBar.ts — session count in tooltip
In the tooltip string, if daemon is reachable, append:
```
\n— AI Sessions: N active
```
Add a `GET /api/ai-sessions` call alongside the existing `Promise.all([getProviders(), getTasks()])` → `Promise.all([getProviders(), getTasks(), listSessions()])` and include the count. Keep the status bar label unchanged — only the tooltip changes.

### 5. package.json — capability declaration
Add to `"activationEvents"`:
- `"onStartupFinished"` (if not already present) so the session monitor starts immediately when VS Code loads

Add to `"contributes.commands"`:
```json
{ "command": "nexus.showAISessions", "title": "Nexus: Show AI Sessions" }
```

## Acceptance Criteria
- [ ] TypeScript compiles without errors (`npm run build` in `vscode-extension/`)
- [ ] `SessionMonitor.start()` calls `POST /api/ai-sessions` when Copilot models are available
- [ ] `SessionMonitor.stop()` calls `DELETE /api/ai-sessions/{id}` on extension deactivation
- [ ] Heartbeat sends registration update every 60 s
- [ ] Daemon offline does NOT produce error toasts
- [ ] Status bar tooltip includes AI session count
- [ ] `nexus.showAISessions` command registered

## Anti-patterns to Avoid
- NEVER use `vscode.window.showErrorMessage` for session registration failures — these must fail silently
- NEVER store the API key or session secret in `globalState` — `workspaceState` only
- NEVER call `vscode.lm.selectChatModels` on every heartbeat (expensive) — call once on start and on `onDidChangeChatModels`
- NEVER block the extension activation with async session registration — fire-and-forget with `.catch(() => {})`
- NEVER hardcode `agent_name` differently than "GitHub Copilot" for the Copilot source
