---
id: TASK-620
planId: PLAN-075
title: 'Surface WAITING status, waitUntil resume time, Cancel button, ranBy/providerHint, and PromoteResult warnings across UI'
role: coder
wave: 4
status: done
createdAt: 2026-10-19
---

# TASK-620 — Surface WAITING status, waitUntil resume time, Cancel button, ranBy/providerHint, and PromoteResult warnings across UI

1. **`frontend/src/composables/useTasks.ts` & `frontend/src/views/MissionControlView.vue` & `frontend/src/components/TaskDetailDrawer.vue`**:
   - Include `WAITING` tasks in `queuedTasks`, header active counts, and the `"Active"` filter in `MissionControlView.vue`.
   - Render `task.waitUntil` (formatted resume time) in `MissionControlView.vue` and `TaskDetailDrawer.vue` when `task.status === 'WAITING'`.
   - Show the **Cancel Task** button in `TaskDetailDrawer.vue` when `task.status === 'WAITING'`.
   - Display `task.ranBy || task.providerName || task.providerHint || '—'` in the Mission Control provider column.
   - Return `Promise<PromoteResult>` from `useTasks().promote` and surface `res.warning` in `BacklogList.vue` and `MissionControlView.vue` when `res.promoted === false`.
   - Include `TOO_LARGE` and `NO_PROVIDER` in `MissionControlView.vue` and `HistoryView.vue` history filters and status color maps.

Done when: the failing test came first, 100% frontend coverage and `make nice` pass, docs and changelog updated.
