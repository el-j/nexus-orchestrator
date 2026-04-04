---
id: TASK-234
title: "Frontend: Fix BacklogView + HistoryView data sources"
role: frontend
planId: PLAN-033
status: done
dependencies: [TASK-233]
createdAt: 2026-03-13T01:10:00.000Z
completedAt: 2026-03-13T01:50:00.000Z
---

## Context
BacklogView and HistoryView use `useTasks()` which calls `getQueue()` — this only returns QUEUED/PROCESSING tasks.  BacklogView needs DRAFT/BACKLOG tasks (via `getBacklog`), and HistoryView needs COMPLETED/FAILED/CANCELLED (via `getAllTasks`).

## Files to Read
- `frontend/src/views/BacklogView.vue`
- `frontend/src/views/HistoryView.vue`
- `frontend/src/composables/useTasks.ts`
- `frontend/src/types/wails.ts`
- `frontend/src/composables/useProjectState.ts`

## Implementation Steps

1. **wails.ts**: Add `getAllTasks()` wrapper function:
   ```ts
   export async function getAllTasks(): Promise<Task[]> {
     if (isWails()) return window.go!.main!.App!.GetAllTasks()
     const r = await fetch(`${await resolveServerUrl()}/api/tasks/all`)
     return r.json() as Promise<Task[]>
   }
   ```
   Also add `GetAllTasks(): Promise<Task[]>` to the WailsApp interface.

2. **BacklogView.vue**: Replace `useTasks()` with direct `getBacklog()` calls:
   - Import `getBacklog` from wails.ts
   - Use `currentProject` from useProjectState
   - Call `getBacklog(currentProject.value ?? '')` on mount + SSE refresh
   - Handle empty project (pass empty string → backend returns all backlog)

3. **HistoryView.vue**: Replace `useTasks()` with `getAllTasks()`:
   - Import `getAllTasks` from wails.ts
   - Call on mount + SSE refresh
   - Filter locally for COMPLETED/FAILED/CANCELLED statuses
   - Add `?? []` null guard

4. **useTasks.ts**: Remove stale `backlogTasks` computed property (it was always empty). Keep `queuedTasks` since DashboardView uses it correctly.

5. Add SSE listeners in both views for `task_changed` events to auto-refresh.

## Acceptance Criteria
- [ ] `vue-tsc --noEmit` exits 0
- [ ] BacklogView shows DRAFT tasks after creating a draft
- [ ] HistoryView shows COMPLETED/FAILED/CANCELLED tasks
- [ ] DashboardView still works (uses `useTasks` → `getQueue` correctly)
- [ ] Both views auto-refresh on SSE events

## Anti-patterns to Avoid
- NEVER hardcode server URLs — use `resolveServerUrl()`
- NEVER assign Wails array results without `?? []` guard
