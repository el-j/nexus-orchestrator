---
id: TASK-256
title: "Extension: Enhance sessionMonitor with auto-claim and progress reporting"
role: backend
planId: PLAN-038
status: todo
dependencies: [TASK-255]
createdAt: 2026-03-13T16:00:00.000Z
---

## Context
The VS Code extension currently only registers a session and sends heartbeats. It doesn't monitor Copilot activity or claim tasks for the active project. This task enhances the extension so it can auto-claim queued tasks for the workspace and report task completion.

## Files to Read
- `vscode-extension/src/sessionMonitor.ts`
- `vscode-extension/src/nexusClient.ts`
- `vscode-extension/src/extension.ts`
- `vscode-extension/src/taskQueueProvider.ts`

## Implementation Steps
1. Add `claimTask(taskId: string, sessionId: string)` method to `NexusClient`:
   - POST `/api/tasks/{id}/claim` with `{ sessionId }`

2. Add `updateTaskStatus(taskId: string, sessionId: string, status: string, logs?: string)` method to `NexusClient`:
   - PUT `/api/tasks/{id}/status` with `{ sessionId, status, logs }`

3. Add `getSessionTasks(sessionId: string)` method to `NexusClient`:
   - GET `/api/ai-sessions/{id}/tasks`

4. In `SessionMonitor`, add a periodic poll (every 30s) after registration:
   - Fetch queued tasks for the workspace project path
   - If any QUEUED tasks match the workspace path + no task is currently claimed → auto-claim the highest priority one
   - Store claimed task ID in monitor state

5. Add `reportTaskComplete(taskId, status, logs)` method that the extension can call from commands or task completion hooks.

6. Update `taskQueueProvider` to show claimed status badge on tasks owned by the local session.

## Acceptance Criteria
- [ ] `NexusClient` has `claimTask`, `updateTaskStatus`, `getSessionTasks` methods
- [ ] Extension auto-claims queued tasks matching workspace project
- [ ] Claimed tasks are visible in the task queue tree view with ownership indicator
- [ ] `npm run compile` and extension tests pass
