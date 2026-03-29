---
id: TASK-424
plan: PLAN-057
status: done
role: frontend
wave: 5
---

# TASK-424: Merge History into Mission Control

Add an expandable "All History" section at the bottom of MissionControlView that shows all terminal tasks (currently in HistoryView). Include the status filter tabs (All/Completed/Failed/Cancelled) and summary badge row from HistoryView.

This allows users to see both active and historical tasks in one view. The section should be collapsed by default and lazy-load data on expand.

**Files:** `frontend/src/views/MissionControlView.vue`
**Depends on:** TASK-410
