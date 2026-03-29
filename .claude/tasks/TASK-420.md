---
id: TASK-420
plan: PLAN-057
status: done
role: frontend
wave: 4
---

# TASK-420: Show task.instruction inline in all task lists

The `Task.Instruction` field is the most important human-readable field but is hidden behind detail drawers in all views. Fix by adding truncated instruction text (max ~80 chars with ellipsis) directly in task rows across:

- `TaskQueue.vue` (Dashboard component)
- `BacklogList.vue`
- `HistoryView.vue` task table rows
- `MissionControlView.vue` unified task list

**Files:** `frontend/src/components/TaskQueue.vue`, `frontend/src/components/BacklogList.vue`, `frontend/src/views/HistoryView.vue`
**Depends on:** none
