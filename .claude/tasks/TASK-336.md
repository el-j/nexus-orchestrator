# TASK-336: HTTP API — activity endpoints + SSE events

**Plan:** PLAN-050 · Wave 3
**Status:** DONE
**Agent:** Senior Developer

## Description

Add HTTP API endpoints for activities:

- `GET /api/activities` — list recent activities, query params: `since` (RFC3339), `agent` (filter by name), `project` (filter by path), `type` (filter by activity type), `limit` (default 50)
- `GET /api/activities/timeline` — chronological cross-agent feed, params: `since`, `limit` (default 100)

Add SSE event type `ai_activity_new` to the existing event stream at `/api/events`.

Wire these into the httpapi router alongside existing routes.

## Acceptance

- Both endpoints return JSON arrays of AIActivity
- Filtering works correctly
- SSE event fires for new activities

## Completed

Added `GET /api/activities` and `GET /api/activities/timeline` HTTP endpoints with filtering and `ai_activity_new` SSE event type.

- Tests for both endpoints
