---
id: TASK-425
plan: PLAN-057
status: todo
role: frontend
wave: 5
---

# TASK-425: Merge Backlog into Mission Control

The Backlog view (DRAFT/BACKLOG tasks) should appear as part of Mission Control's unified task list. DRAFT and BACKLOG tasks are shown in the same list as QUEUED/PROCESSING tasks, distinguished by status badge and with appropriate inline actions (Promote, Dismiss, Edit).

Remove the standalone Backlog sidebar entry after merging. BacklogView.vue can be deleted or kept as an alias.

**Files:** `frontend/src/views/MissionControlView.vue`, `frontend/src/components/AppSidebar.vue`
**Depends on:** TASK-410, TASK-419
