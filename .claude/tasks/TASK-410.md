---
id: TASK-410
plan: PLAN-057
status: todo
role: frontend
wave: 2
---

# TASK-410: Build MissionControlView — status bar + unified task list

Create `frontend/src/views/MissionControlView.vue` as the new primary dashboard. Must include:

1. **Status bar**: Active agents count, queued tasks count, providers down count — single horizontal strip
2. **Unified task list**: ALL non-terminal tasks (DRAFT, QUEUED, PROCESSING) shown in one scrollable list sorted by status then priority. Each row shows: status badge, task ID, `instruction` text (truncated), provider/model, project, elapsed time. Inline actions: Cancel, Promote (for DRAFT), Edit.

Uses `useTasks()` with SSE for real-time updates. Replace `DashboardView` as the default landing page.

**Files:** `frontend/src/views/MissionControlView.vue` (NEW)
**Depends on:** TASK-407, TASK-409
