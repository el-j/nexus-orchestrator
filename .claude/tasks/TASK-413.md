---
id: TASK-413
plan: PLAN-057
status: todo
role: frontend
wave: 2
---

# TASK-413: MissionControlView recent completions panel

Add a "Recent" panel showing last 10 terminal tasks (COMPLETED, FAILED, CANCELLED):

- Status icon (checkmark/X/circle)
- Task ID + instruction text (truncated)
- Time ago ("23m ago")
- Duration (from TASK-409 DurationMs — "0.8s")
- Token count (if available)

Uses `getAllTasks()` filtered to terminal statuses, sorted by CompletedAt desc, limit 10. Does NOT need real-time updates (static on mount + manual refresh).

**Files:** `frontend/src/views/MissionControlView.vue`
**Depends on:** TASK-409, TASK-410
