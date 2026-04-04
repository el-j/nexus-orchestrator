---
id: TASK-473
plan: PLAN-062
status: done
wave: 2
priority: 2
---

# TASK-473: Fix AppSidebar daemon status — always green hardcoded pulse

**Problem:** `frontend/src/components/AppSidebar.vue` renders the daemon status indicator with a hardcoded `bg-emerald-400 animate-pulse` class unconditionally. The indicator is always green regardless of whether the daemon is reachable. This masks connectivity failures from the user entirely.

**Fix:**

1. Create (or reuse) a `useServerHealth.ts` composable that calls `GET /health` on a 10 s interval
2. Expose `connected: Ref<boolean>` and `lastChecked: Ref<Date | null>` from `useServerHealth`
3. Implement initial check on mount (do not wait 10 s for first result)
4. If three consecutive health checks fail, set `connected.value = false`; reset to `true` on any success
5. In `AppSidebar.vue`, import `useServerHealth` and replace the hardcoded class with a computed binding:
   - `connected` → `bg-emerald-400 animate-pulse`
   - `!connected` → `bg-red-500`
6. Add a tooltip (`:title`) to the status dot showing `"Daemon online"` or `"Daemon unreachable"` plus `lastChecked` timestamp
7. Ensure the polling interval is cleared in `onUnmounted` of `AppSidebar.vue`

**Files:**

- `frontend/src/components/AppSidebar.vue`
- `frontend/src/composables/useServerHealth.ts` (new)

**Acceptance criteria:**

- Status dot turns red within 15 s of daemon shutdown
- Status dot turns green within 15 s of daemon restart
- `vue-tsc --noEmit` zero errors
- No memory leak: interval cleared on sidebar unmount
