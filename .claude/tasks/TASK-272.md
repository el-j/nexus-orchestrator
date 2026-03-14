# TASK-272 — TaskQueueProvider: show all tasks + daemon offline indicator

**Plan**: PLAN-043  
**Status**: done  
**Role**: frontend

## What

The Task Queue VS Code tree view currently calls `GET /api/tasks` which only returns
QUEUED + PROCESSING tasks. Once a task completes it disappears, and if the daemon is offline
the view silently shows empty — with no feedback to the user.

## Changes

### `vscode-extension/src/nexusClient.ts`
Add:
```ts
async getAllTasks(): Promise<Task[]> {
  return this.get<Task[]>('/api/tasks/all')
}
```

### `vscode-extension/src/taskQueueProvider.ts`
1. Use `getAllTasks()` instead of `getTasks()` in `getChildren()`
2. Sort: QUEUED + PROCESSING first, then all others by `updatedAt` desc
3. On any fetch error, return `[new DaemonOfflineItem()]` instead of `[]`
4. `DaemonOfflineItem` — `ThemeIcon('debug-disconnect')`, label `"Nexus daemon offline"`

## Acceptance
- Submitting and completing a task keeps it visible in the tree (with ✅ icon)
- When daemon is not running, tree shows a single "Nexus daemon offline" node
- QUEUED/PROCESSING tasks appear before completed/failed ones
