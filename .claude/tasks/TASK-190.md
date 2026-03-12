---
id: TASK-190
title: "GUI: Task History view (completed/failed/cancelled tasks + detail)"
status: todo
priority: medium
role: frontend
dependencies: none
estimated_effort: M 1h
---

## Goal

Add a **Task History** page to the Web GUI so users can browse all completed, failed, and cancelled tasks — and open a detail panel for full output/logs.

## Context

- Current views in `frontend/src/views/`: `DashboardView.vue`, `BacklogView.vue`, `ProvidersView.vue`, `AISessionsView.vue`, `DiscoveryView.vue` — no history view exists
- Existing components usable: `TaskDetailDrawer.vue`, `TaskStatusBadge.vue`, `LogPanel.vue`
- Composable `useTasks.ts` already fetches tasks from `GET /api/tasks` (returns all statuses)
- Sidebar component: `AppSidebar.vue` — needs a new nav entry for History
- The HTTP API returns all tasks from `GET /api/tasks`; filtering by status is done client-side
- `domain.ts` type `TaskStatus` includes `completed | failed | cancelled`

## Scope

### Files to create
- `frontend/src/views/HistoryView.vue` — page component

### Files to modify
- `frontend/src/router/index.ts` (or equivalent) — add `/history` route
- `frontend/src/components/AppSidebar.vue` — add History nav link

## Implementation

1. Create `HistoryView.vue`:
   - Use `useTasks` composable; filter tasks where `status` is `completed`, `failed`, or `cancelled`
   - Display as a table/list: task ID, project (truncated), target file, status badge, completedAt timestamp
   - Clicking a row opens `TaskDetailDrawer.vue` with full output and logs
   - Include a status-filter toolbar (All / Completed / Failed / Cancelled)
   - Empty state message when no history exists
2. Add route `/history` pointing to `HistoryView` in the router.
3. Add "History" entry in `AppSidebar.vue` (after Backlog, before Providers).

## Acceptance Criteria
- [ ] `HistoryView.vue` exists and is registered in the router
- [ ] AppSidebar has a working "History" navigation link
- [ ] Page shows tasks with status `completed`, `failed`, or `cancelled` (not draft/queued/processing)
- [ ] Status filter buttons narrow the list correctly
- [ ] Clicking a task opens `TaskDetailDrawer` with output/logs
- [ ] Empty state renders without errors when there are no history tasks
