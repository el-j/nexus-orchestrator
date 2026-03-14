# PLAN-043 — Task Visibility & Extension Refresh

**Status**: active  
**Created**: 2026-03-14

## Problem Statement

The VS Code extension was built from source at 17:57 on 2026-03-13, **before** the port migration
(9999 → 63987). The bundled `dist/extension.js` still carries `"http://127.0.0.1:9999"` as the
default base URL. Because VS Code reads the hardcoded TypeScript fallback in the bundle, the
installed VSIX cannot reach the daemon.

Beyond the port issue, three UX gaps were identified:

| # | Symptom | Root cause |
|---|---------|-----------|
| 1 | Task Queue view is empty even when tasks exist | `GET /api/tasks` only returns QUEUED + PROCESSING; completed tasks vanish silently |
| 2 | Task Queue shows empty list when daemon is offline, with no indicator | `getChildren()` silently returns `[]` on fetch error |
| 3 | WorkspaceOrchView shows `.claude/orchestrator.json` tasks only, never live daemon tasks | The view reads local files; it does not query the daemon |
| 4 | Old VSIX hardcoded to port 9999 | Built before 9999→63987 port migration |

## Port Reference (NEXUS T9 keypad: 63987)

| Service    | Port  | Mnemonic |
|------------|-------|----------|
| HTTP API   | 63987 | NEXUS    |
| MCP server | 63988 | NEXUS+1  |
| Vite dev   | 63989 | NEXUS+2  |

## Solution Design

### TASK-272 — Fix TaskQueueProvider: show all tasks + offline indicator

**File**: `vscode-extension/src/taskQueueProvider.ts` + `nexusClient.ts`

- Add `getAllTasks(): Promise<Task[]>` to `NexusClient` using `GET /api/tasks/all`
- Change `TaskQueueProvider.getChildren()` to call `getAllTasks()` (returns every task, all statuses)
- Sort: active tasks first (QUEUED, PROCESSING), then by `updatedAt` desc
- Add group header items (collapsible): "Active (N)" and "Recent (N)" to separate the two bands
- On fetch error, return a single `DaemonOfflineItem` TreeItem with icon `debug-disconnect` and
  label `"Nexus daemon offline – start daemon to connect"` instead of silent `[]`

### TASK-273 — Live daemon tasks per project in WorkspaceOrchView

**Files**: `workspaceOrchView.ts`, `workspaceScanner.ts`, `extension.ts`

- Add optional `NexusClient` to `WorkspaceOrchViewProvider` constructor
- In `getChildren(FolderNode)`, fetch `getAllTasks()` from the client and filter by
  `task.projectPath === folderPath` (exact match + trailing slash normalisation)
- Show a "Live Tasks" section inside each folder node when the client is available:
  - Header: `"Live Tasks (N)"` with icon `server-process`
  - Children: one `LiveTaskItem` per task (status icon + short instruction + status label)
- Falls back gracefully when client is unavailable or daemon is offline

### TASK-274 — Rebuild VSIX + install

- `cd vscode-extension && npm run build` → regenerates `dist/extension.js` with port 63987
- `vsce package --no-dependencies` → rebuilds `build/vscode/nexus-orchestrator.vsix`
- Install new VSIX via `code --install-extension build/vscode/nexus-orchestrator.vsix --force`

### TASK-275 — Start daemon + end-to-end validation

- `CGO_ENABLED=1 go build ./cmd/nexus-daemon/... -o nexus-daemon`
- Run daemon, verify health endpoint, verify `/api/howto`, verify extension connects
- Confirm Task Queue view shows real tasks after submission

## Task Breakdown

| Task     | Title                                       | Role        |
|----------|---------------------------------------------|-------------|
| TASK-272 | TaskQueueProvider: all tasks + offline node | frontend    |
| TASK-273 | WorkspaceOrchView: live daemon tasks        | frontend    |
| TASK-274 | Rebuild VSIX + install                      | build       |
| TASK-275 | Start daemon + end-to-end validation        | qa          |
