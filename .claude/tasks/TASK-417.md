---
id: TASK-417
plan: PLAN-057
status: todo
role: frontend
wave: 3
---

# TASK-417: Delete AISessionsView and AIAgentsView

Delete `AISessionsView.vue` and `AIAgentsView.vue`. Remove their sidebar entries and routes. Wire the new `AgentsView.vue` (TASK-416) as the single "Agents" sidebar entry.

**Files:** DELETE `frontend/src/views/AISessionsView.vue`, DELETE `frontend/src/views/AIAgentsView.vue`, update `AppSidebar.vue`, `App.vue`
**Depends on:** TASK-416
