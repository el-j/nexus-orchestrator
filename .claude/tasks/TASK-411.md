---
id: TASK-411
plan: PLAN-057
status: done
role: frontend
wave: 2
---

# TASK-411: MissionControlView agents panel

Add a compact "Agents" panel to MissionControlView showing all registered AI sessions:

- Agent name + model (if available)
- Project path (short basename)
- Token count (from activity data)
- Idle time or active indicator
- Colored status dot (green=active, yellow=idle, gray=disconnected)

Uses `useAISessions()` and `useActivities()` for token aggregation. Compact card layout, max 4-6 visible with scroll.

**Files:** `frontend/src/views/MissionControlView.vue`
**Depends on:** TASK-410
