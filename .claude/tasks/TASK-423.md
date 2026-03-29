---
id: TASK-423
plan: PLAN-057
status: todo
role: frontend
wave: 5
---

# TASK-423: Merge Live AI into AgentsView as "Activity" tab

Move the `LiveActivityView` real-time activity timeline into `AgentsView.vue` as a tab or expandable section. When viewing an agent, users should see its recent activity (messages, tool use, file edits) inline without switching to a separate page.

Keep the session-grouped timeline layout from LiveActivityView. Add a tab header: "Sessions | Activity | Discovered".

**Files:** `frontend/src/views/AgentsView.vue`, reference `LiveActivityView.vue`
**Depends on:** TASK-416, TASK-417
