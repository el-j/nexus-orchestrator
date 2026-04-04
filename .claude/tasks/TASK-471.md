---
id: TASK-471
plan: PLAN-062
status: done
wave: 1
priority: 1
---

# TASK-471: Fix useGlobalSSE — currently bypassed, 5-6 concurrent SSE connections

**Problem:** `frontend/src/composables/useGlobalSSE.ts` is instantiated as a singleton in `App.vue` but every composable that needs real-time data (`useTasks`, `useAISessions`, `useActivities`, `useDiscovery`, `useLogs`) independently creates its own `EventSource`. This produces 5-6 concurrent SSE connections per page load. On reconnect storms (daemon restart, network drop) this multiplies. The global SSE bus exists but nothing subscribes to it.

**Fix:**

1. Audit each composable (`useTasks`, `useAISessions`, `useActivities`, `useDiscovery`, `useLogs`) and identify where they open their own `EventSource` — list the exact lines
2. In `useGlobalSSE.ts`, confirm (or add) typed event routing: `task` event type → `task` handler, `activity` event type → `activity` handler, `session` event type → `session` handler, `discovery` event type → `discovery` handler, `log` event type → `log` handler
3. Export `onEvent(type: string, handler: (data: unknown) => void)` and `offEvent(type: string, handler)` from `useGlobalSSE.ts` for typed subscription
4. In `useTasks.ts`: remove the local `EventSource` construction; subscribe to the global `task` events via `onEvent('task', ...)` in `onMounted`, unsubscribe in `onUnmounted`
5. Repeat step 4 for `useAISessions.ts` (subscribe to `session` events), `useActivities.ts` (subscribe to `activity` events), `useDiscovery.ts` (subscribe to `discovery` events), `useLogs.ts` (subscribe to `log` events)
6. Ensure `useGlobalSSE.ts` singleton is initialized once in `App.vue` before any composable that uses it is mounted
7. Add a `connectionCount` debug log at `EventSource` open time in `useGlobalSSE.ts` — verify only one fires per page load
8. Update `useGlobalSSE.ts` reconnect logic to use exponential backoff (3s → 6s → 12s → 30s cap, reset on successful message)
9. Ensure handler cleanup in `offEvent` correctly removes from the internal Map to prevent memory leaks across component unmounts

**Files:**

- `frontend/src/composables/useGlobalSSE.ts`
- `frontend/src/composables/useTasks.ts`
- `frontend/src/composables/useAISessions.ts`
- `frontend/src/composables/useActivities.ts`
- `frontend/src/composables/useDiscovery.ts`
- `frontend/src/composables/useLogs.ts`
- `frontend/src/App.vue`

**Acceptance criteria:**

- Browser DevTools Network tab shows exactly **one** `EventSource` (SSE) connection at idle
- All real-time views (tasks, sessions, activities, logs) continue to update without polling
- Rapid component mount/unmount does not leak EventSource instances
- `vue-tsc --noEmit` zero errors
