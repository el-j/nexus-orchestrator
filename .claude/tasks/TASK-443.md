---
id: TASK-443
plan: PLAN-059
status: todo
wave: 2
priority: 2
---

# TASK-443: useDiscoveredPlans.ts — fix memory leak

**Problem:** Line 47 has `setInterval(fetchPlans, 30_000)` but interval ID is never stored and never cleared. No `onUnmounted` cleanup. Intervals accumulate across component mounts.

**Fix:**

1. Store interval ID: `const intervalId = setInterval(fetchPlans, 30_000)`
2. Add `onUnmounted(() => clearInterval(intervalId))` or use `tryOnUnmounted` from VueUse
3. Ensure only one interval per composable instance

**Files:** `frontend/src/composables/useDiscoveredPlans.ts`
