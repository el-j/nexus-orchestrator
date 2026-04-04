---
id: TASK-472
plan: PLAN-062
status: done
wave: 1
priority: 1
---

# TASK-472: Fix useLogs SSE — no reconnect causes permanent disconnect

**Problem:** `frontend/src/composables/useLogs.ts` sets `es.onerror = () => { connected.value = false }` and does nothing else. The first SSE error (daemon restart, network hiccup, HTTP timeout) permanently disconnects the LogPanel. There is no reconnect, no backoff, and no polling fallback. `useActivities.ts` has a working polling fallback pattern that should be replicated here.

**Fix:**

1. Add a `reconnectDelay` ref (starts at 3000 ms) and a `reconnectTimer` ref to `useLogs.ts`
2. In `es.onerror`, close the broken `EventSource`, set `connected.value = false`, schedule a reconnect via `setTimeout` using `reconnectDelay`
3. Implement exponential backoff: double `reconnectDelay` on each failure (cap at 30 000 ms), reset to 3000 ms on first successful `es.onopen`
4. Extract SSE setup into a `connect()` function so it can be called both on mount and on reconnect
5. Add a polling fallback (`setInterval`) at 10 s interval that fires only when `connected.value === false` — fetches logs via `GET /api/logs?limit=50` and merges into the `logs` ref without duplicates
6. Clear the polling interval and any pending `reconnectTimer` in `onUnmounted` (prevent leaks after component teardown)
7. Expose `connected` and an `error` ref from `useLogs` so `LogPanel.vue` can show a "Reconnecting…" state
8. Update `LogPanel.vue` (or equivalent consumer) to render a warning banner when `connected === false`

**Files:**

- `frontend/src/composables/useLogs.ts`
- `frontend/src/components/LogPanel.vue` (or equivalent log consumer component)

**Acceptance criteria:**

- Stopping and restarting the daemon: LogPanel reconnects automatically within 30 s without manual page refresh
- Console shows backoff timing (3s, 6s, 12s…) not a rapid hammer loop
- `onUnmounted` leaves no active timers or open EventSources (verified via DevTools)
- Polling fallback populates logs when SSE is unavailable
