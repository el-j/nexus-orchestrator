# TASK-339: Rebuild LiveActivityView — unified real-time timeline

**Plan:** PLAN-050 · Wave 4
**Status:** DONE
**Agent:** UI Designer

## Description

Rebuild `frontend/src/views/LiveActivityView.vue` to show a real-time scrolling timeline of ALL AI activities.

Layout:

- Header: "Live Activity" with count of active agents and total activities
- Filter bar: by agent, by project, by activity type (toggles)
- Scrolling timeline (newest first): each entry shows agent badge, activity summary, project tag, relative timestamp, token count if available
- Auto-updates via SSE `ai_activity_new` events (append to top of list)
- Fallback: poll GET /api/activities/timeline every 5s

Use existing composable pattern (create `useActivities.ts` composable).
Style consistent with existing dark theme (Tailwind + PrimeVue).

## Acceptance

- Shows real-time activity timeline
- SSE updates append new items without full refresh
- Filters work
- Empty state: "No AI activity detected — start an AI tool and it will appear here"

## Completed

Rebuilt `LiveActivityView.vue` with SSE-driven real-time chronological timeline, filter bar, `useActivities.ts` composable, and empty-state guidance.
