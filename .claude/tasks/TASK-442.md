---
id: TASK-442
plan: PLAN-059
status: closed
closedReason: 'stale: planned before v0.10.0 and never started; the work is tracked in PLAN-073 and PLAN-074 or was superseded'
wave: 1
priority: 1
---

# TASK-442: Delete dead DashboardView.vue

**Problem:** DashboardView.vue is a duplicate of MissionControlView. Not imported or registered anywhere. Dead code.

**Fix:** Delete `frontend/src/views/DashboardView.vue`

**Files:** `frontend/src/views/DashboardView.vue`
