---
id: TASK-414
plan: PLAN-057
status: done
role: frontend
wave: 2
---

# TASK-414: MissionControlView submit form

Add a collapsible inline task submission form at the bottom of MissionControlView:

- Instruction textarea (primary)
- Project path, Target file, Provider, Priority dropdowns (secondary, collapsed by default)
- "Submit to Queue" button + "Save as Draft" split button
- Collapse toggle (▾) to hide/show the secondary fields

Reuse `TaskSubmitForm.vue` component or extract its logic. Form should be pinned at viewport bottom.

**Files:** `frontend/src/views/MissionControlView.vue`, possibly `frontend/src/components/TaskSubmitForm.vue`
**Depends on:** TASK-410
