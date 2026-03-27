# TASK-342: Project activity view

**Plan:** PLAN-050 · Wave 4
**Status:** DONE
**Agent:** UI Designer

## Description

Create `frontend/src/views/ProjectActivityView.vue` — shows AI activity grouped by project.

Layout:

- Project cards: each project that has active AI work shows as a card
- Inside each card: list of agents working in that project, with their current activity
- Activity mini-feed per project: last 5 activities
- Click project card → filtered timeline view for that project

Data source: GET /api/activities?project=<path> or group client-side from full timeline.

Add route and sidebar nav item "Projects" (or integrate into existing project selector).

## Acceptance

- Groups activities by project path
- Shows which agents are working in each project
- Click-through to filtered timeline

## Completed

Created `ProjectActivityView.vue` grouped by project with per-project agent list, activity mini-feed, and click-through to filtered timeline.
