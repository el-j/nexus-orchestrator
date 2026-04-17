---
id: TASK-574
title: VS Code extension tests — nexusClient, extension activation, AISessionsTreeProvider
role: qa
planId: PLAN-071
status: todo
dependencies: [TASK-563, TASK-564, TASK-565]
createdAt: 2026-04-15T00:00:00Z
---

## Context

`nexusClient.ts` was recently modified (appears in git status as `M`) and has zero test coverage — the highest-risk untested file in the project. `extension.ts` activation and deactivation paths are also untested. `AISessionsTreeProvider` full class has no tests. After TASK-563 adds Zod validation, tests must verify schemas reject malformed responses.

## Files to Read

- `vscode-extension/src/nexusClient.ts`
- `vscode-extension/src/extension.ts`
- `vscode-extension/src/aiSessionsTreeProvider.ts`
- `vscode-extension/src/schemas.ts` (created by TASK-563)
- Any existing `*.test.ts` in `vscode-extension/src/` for test patterns

## Implementation Steps

1. Create `vscode-extension/src/nexusClient.test.ts`:
   - Mock `node-fetch` or global `fetch` with `vi.mock` / `jest.mock`
   - `test('getQueue returns parsed tasks on 200')` — mock fetch returns valid JSON array
   - `test('getQueue throws on non-200')` — mock fetch returns 503
   - `test('getBrainStatus throws NexusValidationError on malformed response')` — mock returns `{ wrong: true }` — Zod should reject it
   - `test('getActivities returns empty array on null response')`
   - `test('submitTask posts correct body and returns task id')`
   - `test('updateTaskStatus calls correct endpoint with sessionId, status, logs')`
2. Create `vscode-extension/src/extension.test.ts`:
   - Mock `vscode` module
   - `test('activate registers all expected commands')`
   - `test('deactivate disposes all subscriptions without throwing')`
   - `test('onDidChangeConfiguration rebuilds client without creating duplicate pollers')`
3. Create `vscode-extension/src/aiSessionsTreeProvider.test.ts`:
   - `test('getTreeItem returns correct label for session')`
   - `test('getChildren returns empty array when no sessions')`
   - `test('refresh triggers getChildren re-fetch')`

## Acceptance Criteria

- [ ] `npm test` exits 0 in `vscode-extension/`
- [ ] `npm run compile` exits 0
- [ ] `nexusClient.test.ts` covers getQueue, getBrainStatus (Zod rejection), submitTask, updateTaskStatus
- [ ] `extension.test.ts` covers activate, deactivate, config change
- [ ] `aiSessionsTreeProvider.test.ts` covers getTreeItem, getChildren, refresh

## Anti-patterns to Avoid

- NEVER use real network calls in tests — mock all fetch calls
- NEVER test VS Code UI directly — test tree data providers via their return values
