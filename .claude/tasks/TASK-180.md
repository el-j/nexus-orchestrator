---
id: TASK-180
title: Frontend — BacklogView, BacklogList, ProjectSelector components
role: frontend
planId: PLAN-024
status: todo
dependencies: [TASK-179]
createdAt: 2026-03-11T22:00:00.000Z
---

## Context
Users need a dedicated backlog view per project to manage their idea pipeline. A project selector in the sidebar filters all views. Backlog items can be promoted to queue, edited, or deleted.

## Files to Read
- `frontend/src/components/AppSidebar.vue` — nav items
- `frontend/src/components/TaskQueue.vue` — list component pattern
- `frontend/src/components/TaskDetailDrawer.vue` — detail panel pattern
- `frontend/src/views/DashboardView.vue` — view wrapper pattern
- `frontend/src/composables/useProjectFilter.ts` — from TASK-179

## Implementation Steps

1. Create `frontend/src/components/ProjectSelector.vue`:
   - Dropdown showing basename of each project path (full path in tooltip)
   - "All Projects" default option
   - Placed in AppSidebar above nav items, below logo
   - Consumes `useProjectFilter()` to set/get `currentProject`
   - Violet highlight when a project is selected

2. Create `frontend/src/components/BacklogList.vue`:
   - Props: `items: Task[]`, `loading: boolean`
   - Card list (same pattern as TaskQueue.vue cards)
   - Each card: priority badge (1=red, 2=amber, 3+=slate), instruction snippet, tags as small pills, provider chip ("Auto" when empty), relative timestamp
   - Hover actions: Promote (arrow-up), Edit (pencil), Delete (trash)
   - Promote calls `useTasks().promoteTask(id)` → item moves to TaskQueue
   - Sort: priority ASC then creation date ASC
   - Empty state: "No backlog items. Use the form below to draft ideas."

3. Create `frontend/src/views/BacklogView.vue`:
   - Header: "Backlog" + counts: "N drafts · M backlog" for current project
   - Body: `<BacklogList :items="backlogTasks" />`
   - Footer: Submit form in draft mode (save-as-draft default action)

4. Create `frontend/src/components/ProjectSummaryBar.vue`:
   - Horizontal row of stat cards: Active, Queued, Backlog, Completed (24h), Failed (24h)
   - Backlog card is clickable → navigates to BacklogView
   - Only visible when a project is selected (hidden in "All Projects" mode)

5. Update `frontend/src/components/AppSidebar.vue`:
   - Add `<ProjectSelector />` above nav items
   - Add "Backlog" nav item with inbox icon between "Task Queue" and "Providers"
   - Add badge showing DRAFT+BACKLOG count for current project

6. Update `frontend/src/views/DashboardView.vue`:
   - Add `<ProjectSummaryBar />` when project is selected
   - Filter TaskQueue by current project

7. Add route `/backlog` → `BacklogView` in Vue router.

## Acceptance Criteria
- [ ] ProjectSelector dropdown shows unique project paths from all tasks
- [ ] Selecting a project filters all views (TaskQueue, Backlog, etc.)
- [ ] BacklogList shows DRAFT+BACKLOG items with priority ordering
- [ ] "Promote" button moves item from backlog to queue
- [ ] ProjectSummaryBar shows correct counts
- [ ] `/backlog` route loads BacklogView

## Anti-patterns to Avoid
- NEVER derive project list from anything other than task data (no separate registration)
- NEVER implement drag-and-drop in v1 — priority numbers control ordering
- NEVER duplicate provider/model fetching — reuse `useProviders()`
