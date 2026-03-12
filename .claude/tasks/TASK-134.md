---
id: TASK-134
title: VS Code task queue webview
role: devops
planId: PLAN-018
status: todo
dependencies: [TASK-133]
createdAt: 2026-03-11T16:10:00.000Z
---

## Context
Implement a tree view or webview panel in the VS Code sidebar that shows the current task queue from nexusOrchestrator, with task status, provider assignment, and actions (cancel, view result).

## Files to Read
- `vscode-extension/src/extension.ts`
- `vscode-extension/src/nexusClient.ts`
- `internal/adapters/inbound/httpapi/server.go` (GET /api/tasks)

## Implementation Steps
1. Create a `TaskQueueProvider` implementing `vscode.TreeDataProvider` for the sidebar.
2. Register a view container in `package.json` (`contributes.viewsContainers` + `contributes.views`).
3. Each tree item shows: task ID, status badge (emoji/icon), instruction snippet, provider/model.
4. Auto-refresh every 5 seconds (or on SSE events if available).
5. Context menu on task items: "Cancel Task", "View Result", "Copy Task ID".
6. Add a "Refresh" button in the view title bar.

## Acceptance Criteria
- [ ] Task queue tree view appears in the VS Code sidebar
- [ ] Tasks show status, instruction preview, and provider
- [ ] Cancel action works from the context menu
- [ ] View auto-refreshes

## Anti-patterns to Avoid
- NEVER use webview when a tree view suffices — keep it lightweight
- NEVER poll more frequently than every 3 seconds
