---
id: TASK-133
title: VS Code submit task command
role: devops
planId: PLAN-018
status: todo
dependencies: [TASK-132]
createdAt: 2026-03-11T16:10:00.000Z
---

## Context
Implement the core `nexus.submitTask` command that lets users submit a coding task from VS Code to the nexusOrchestrator daemon. This is the primary workflow: select code or describe a task, pick a target file, and submit.

## Files to Read
- `vscode-extension/src/extension.ts`
- `vscode-extension/src/nexusClient.ts`
- `internal/adapters/inbound/httpapi/server.go` (POST /api/tasks shape)

## Implementation Steps
1. Implement `nexus.submitTask` command handler:
   - Show an input box for the task instruction (pre-fill with selected text if any).
   - Auto-detect `projectPath` from the workspace root.
   - Optionally prompt for target file (default: active editor file).
   - Optionally prompt for provider/model (quick pick from `getProviders()`).
   - POST to `/api/tasks` with the payload.
2. Show a progress notification while the task is processing.
3. On completion, offer to open the target file or show a diff.
4. On error, show an error notification with the failure reason.
5. Add a context menu entry "Submit to Nexus" for the editor and explorer.

## Acceptance Criteria
- [ ] User can submit a task from the command palette
- [ ] Task instruction can be typed or pre-filled from selection
- [ ] Progress notification shows while task is in-flight
- [ ] Completed task shows result or opens target file
- [ ] Errors surface clearly

## Anti-patterns to Avoid
- NEVER block the extension host — use async/await throughout
- NEVER expose API keys through the extension
