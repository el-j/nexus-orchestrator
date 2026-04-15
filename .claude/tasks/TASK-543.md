---
id: TASK-543
planId: PLAN-070
title: 'VS Code: add missing brain nexusClient methods, commands, package.json entries, fix searchKnowledge'
role: vscode
status: todo
createdAt: 2026-04-14T02:00:00Z
---

# TASK-543 — VS Code extension brain completeness

## Context

The VS Code extension (`vscode-extension/`) covers only 5 of 9 brain HTTP endpoints and registers
only one brain command. Confirmed gaps:

1. **`nexusClient.ts` missing 4 brain methods**: `initProject`, `listKnowledge`,
   `deleteKnowledge`, `getFileMap` have no HTTP client implementations.

2. **`searchKnowledge()` return type mismatch**: The method is typed to return `ContextSection[]`
   but the server wraps the response: `{"results": [...]}`. The client must unwrap the outer object
   before returning the array.

3. **`extension.ts` missing commands**: Only `nexus.brain.ingest` is registered. Missing:
   `nexus.brain.status`, `nexus.brain.init`, `nexus.brain.search`.

4. **`package.json` missing command declarations**: Only `nexus.brain.ingest` in
   `contributes.commands`. All other brain commands are unregistered.

5. **Status bar quick-pick** (`nexus.statusBarAction`): no brain actions in the pick list.

## Work Required

### `vscode-extension/src/nexusClient.ts`

Add 4 new methods following the existing `ingestKnowledge` pattern:

- `initProject(projectPath: string, claudeMDPath?: string): Promise<BrainStatus>`
  → `POST /api/brain/init`
- `listKnowledge(projectPath: string, kind?: string): Promise<ProjectKnowledge[]>`
  → `GET /api/brain/knowledge?projectPath=...&kind=...`
- `deleteKnowledge(id: string): Promise<void>`
  → `DELETE /api/brain/knowledge/{id}`
- `getFileMap(projectPath: string, focusArea?: string): Promise<string[]>`
  → `GET /api/brain/file-map?projectPath=...&focusArea=...` (unwrap `filePaths` array)

Fix `searchKnowledge`: unwrap `data.results` instead of returning `data` directly.

Add TypeScript types if not already present: `ProjectKnowledge` interface matching
`domain.ProjectKnowledge` Go struct.

### `vscode-extension/src/extension.ts`

Register 3 new commands:

- `nexus.brain.status` — prompts for project path, calls `getClient().getBrainStatus()`, shows status in info message
- `nexus.brain.init` — shows open folder dialog or prompts for path, calls `initProject()`, shows result
- `nexus.brain.search` — prompts for project + query, calls `searchKnowledge()`, shows results in quick-pick

Add brain entries to the `nexus.statusBarAction` quick-pick list.

### `vscode-extension/package.json`

Add `contributes.commands` entries for `nexus.brain.status`, `nexus.brain.init`,
`nexus.brain.search` with appropriate titles and icons (`$(search)`, `$(rocket)`, `$(database)`).

## File Targets

- `vscode-extension/src/nexusClient.ts`
- `vscode-extension/src/extension.ts`
- `vscode-extension/package.json`

## Acceptance Criteria

- `cd vscode-extension && npm run compile` — 0 errors
- `cd vscode-extension && npm test` — all tests pass
- `searchKnowledge` correctly returns `ContextSection[]` (not the `{results:[]}` wrapper)
- All 4 new client methods implemented with correct HTTP verbs and URL patterns
