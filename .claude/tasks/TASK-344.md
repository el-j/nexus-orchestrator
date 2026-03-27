# TASK-344: VS Code extension — live activity tree

**Plan:** PLAN-050 · Wave 5
**Status:** DONE
**Agent:** Senior Developer

## Description

Update `vscode-extension/src/workspaceOrchView.ts` to show live activities from the daemon when reachable.

When daemon is reachable:

- Add "Live Activity" tree group at the top (above plan history)
- Show recent activities for the current workspace's project path
- Each activity as a tree item: icon by type, summary text, relative timestamp
- Refresh every 10s or on SSE event

When daemon not reachable:

- Show "Daemon offline — start with: make dev" message
- Fall through to existing orchestrator.json display

Replace the "No orchestrator.json found" empty state with more useful information: either live activity (if daemon up) or instructions for getting started.

## Acceptance

- Live activities shown when daemon reachable
- Graceful fallback when offline
- Refresh works
- No "No orchestrator.json" when daemon is providing live data

## Completed

Updated `WorkspaceOrchView.ts` to show a "Live Activity" tree group from the daemon when reachable, with graceful offline fallback.
