---
id: TASK-565
title: Wire VS Code dead client methods to commands and add brain command menu bindings
role: vscode
planId: PLAN-071
status: todo
dependencies: [TASK-563, TASK-564]
createdAt: 2026-04-15T00:00:00Z
---

## Context

Five client methods are fully implemented in `nexusClient.ts` but never called from any command or tree provider: `listKnowledge()`, `deleteKnowledge()`, `getProjectContext()`, `getFocusedContext()`, `getFileMap()`. Three brain commands (`nexus.brain.status`, `nexus.brain.init`, `nexus.brain.search`) have no menu bindings — they are command-palette only with zero discoverability. `getDiscoveredAgents()` is called but the result is never surfaced in any tree view.

## Files to Read

- `vscode-extension/src/nexusClient.ts`
- `vscode-extension/src/extension.ts`
- `vscode-extension/package.json`
- `vscode-extension/src/workspaceOrchView.ts`

## Implementation Steps

1. In `package.json` — add `view/item/context` menu entries for `nexus.brain.status`, `nexus.brain.init`, `nexus.brain.search` on the relevant tree-view item types (e.g., when `viewItem == nexusProject`)
2. Add a `nexus.brain.listKnowledge` command (command palette + menu): shows a QuickPick with knowledge entries for the current workspace project, with kind filter options
3. Add a `nexus.brain.getContext` command: calls `getProjectContext()` with the active workspace path and opens the result in a new read-only text document (plain text or markdown)
4. Add a `nexus.brain.getFileMap` command: calls `getFileMap()` and shows the file list in a QuickPick (with option to open a file)
5. Surface `getDiscoveredAgents()` in the `AISessionsTreeProvider` or `WorkspaceOrchView`: add a "Discovered Agents" tree section that renders each discovered agent with its `kind` and `status`
6. Register all new commands in `extension.ts` `activate()` and push disposables to `context.subscriptions`
7. Add entries for all new commands in `package.json` `contributes.commands`

## Acceptance Criteria

- [ ] `npm run compile` exits 0 in `vscode-extension/`
- [ ] `npm test` exits 0 in `vscode-extension/`
- [ ] `nexus.brain.status`, `nexus.brain.init`, `nexus.brain.search` appear in the view/item context menu
- [ ] `nexus.brain.listKnowledge`, `nexus.brain.getContext`, `nexus.brain.getFileMap` are registered commands
- [ ] `getDiscoveredAgents()` result appears in at least one tree view
- [ ] All new commands are in `package.json` contributes.commands

## Anti-patterns to Avoid

- NEVER call API methods from tree data providers synchronously — use async with error surfacing
- NEVER register commands without pushing the disposable to `context.subscriptions`
