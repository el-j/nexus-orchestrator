---
id: TASK-480
plan: PLAN-062
status: done
wave: 3
priority: 3
---

# TASK-480: Fix DiscoveredPlansView — display-only cards with no actions

**Problem:** `frontend/src/views/DiscoveredPlansView.vue` renders plan and task file cards from the discovery API but the cards have no interactive actions — no click handler, no "Open in editor" button, no "Promote to queue" action, no status badge. The view is read-only in a way that makes it nearly useless for the planning workflow it is supposed to support.

**Fix:**

1. Add a status badge to each card: map the task `status` field (from YAML frontmatter or body) to a colored pill (`todo` → yellow, `in-progress` → blue, `done`/`completed` → green, `blocked` → red)
2. Add an "Open" button to plan cards that calls `router.push({ path: '/plans', query: { file: plan.filePath } })` — or emits to the parent if the view is embedded — to navigate to the detail view
3. Add a "Promote" button to task cards with `status: todo` that calls `POST /api/tasks` (or the MCP `promote_task` equivalent) with the parsed task metadata pre-filled; show a success toast on completion
4. Highlight cards with unfulfilled tasks: if a plan file contains tasks with `status: todo` or `status: in-progress`, add a visual accent (e.g., left border color) to the plan card
5. Add an empty state message when no plans/tasks are discovered: "No plan files found. Create a `.claude/plans/PLAN-NNN.md` file to get started."
6. Clicking a plan card (outside the action buttons) should expand an inline preview of the markdown content (or route to detail)

**Files:**

- `frontend/src/views/DiscoveredPlansView.vue`

**Acceptance criteria:**

- Each card has at least one interactive action (Open or Promote)
- Status badges are visible on all task cards with correct color mapping
- Promoting a task creates an entry in the queue (verified via MissionControlView)
- Empty state renders when discovery returns no files
- `vue-tsc --noEmit` zero errors
