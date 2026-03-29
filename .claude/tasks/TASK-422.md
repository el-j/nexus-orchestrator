---
id: TASK-422
plan: PLAN-057
status: todo
role: frontend
wave: 4
---

# TASK-422: Global SSE connection shared across all views

Currently each view creates its own `EventSource` on mount and disconnects on unmount. Navigating between views drops real-time updates. Fix by creating a global `useGlobalSSE()` composable in App.vue that:

1. Maintains a single `EventSource` to `/api/events`
2. Exposes `on(eventType, callback)` / `off(eventType, callback)` for views to subscribe
3. Reconnects automatically on disconnection
4. Shows connection status indicator in the sidebar or header

All existing composables (`useTasks`, `useAISessions`, `useActivities`, etc.) should switch to using the global SSE instead of per-view connections.

**Files:** `frontend/src/composables/useGlobalSSE.ts` (NEW), update all composables that use SSE
**Depends on:** none
