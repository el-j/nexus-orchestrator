# TASK-273 — WorkspaceOrchView: live daemon tasks per project

**Plan**: PLAN-043  
**Status**: done  
**Role**: frontend

## What

The Workspace Agents view reads `.claude/orchestrator.json` from each workspace folder and
shows plan/task history. It never shows live daemon tasks (QUEUED, PROCESSING, recently
completed) for each project. Users see "done jobs" from the JSON but not currently-running
daemon tasks.

## Changes

### `vscode-extension/src/nexusClient.ts`
Already has `getAllTasks()` after TASK-272. No additional changes needed here.

### `vscode-extension/src/workspaceOrchView.ts`
1. Accept optional `NexusClient` in `WorkspaceOrchViewProvider` constructor
2. Add `LiveTasksGroupNode` (collapsible header: `"Live Tasks (N)"`, icon `server-process`)
3. Add `LiveTaskItem` leaf (status icon + truncated instruction + status description)
4. In `getChildren(FolderNode)`:
   - Fetch `client.getAllTasks()` (async) filtered by `projectPath` matching the folder
   - If ≥1 tasks, push `LiveTasksGroupNode` before plan nodes
   - Sort live tasks: active first (QUEUED/PROCESSING), then recent (last 5 non-active)
5. `getChildren(LiveTasksGroupNode)` → return `LiveTaskItem[]`
6. Gracefully returns empty when client unavailable or fetch fails (no throw)

### `vscode-extension/src/extension.ts`
Pass `getClient()` to `WorkspaceOrchViewProvider` constructor.

## path normalisation
`folderPath` (from workspace folder) may or may not have trailing slash;
`task.projectPath` (from daemon) is `filepath.Clean`'d in Go (no trailing slash, forward slashes).
Normalise both sides: `path.normalize(p).replace(/\/$/, '')` before compare.

## Acceptance  
- Opening a workspace in which the daemon has tasks shows "Live Tasks" section per folder
- Active (QUEUED/PROCESSING) are visible immediately without waiting for completion
- Offline / no-tasks state shows no extra node (clean fallback)
