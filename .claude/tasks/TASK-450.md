---
id: TASK-450
plan: PLAN-059
status: todo
wave: 4
priority: 4
---

# TASK-450: Composable error handling — surface errors properly

**Problem:** Multiple composables silently swallow errors with bare catch blocks or console.warn:

- useActivities.ts: bare catches (lines 38, 44)
- useAISessions.ts: bare catches (lines 43, 46)
- useDiscovery.ts: console.warn only (lines 13, 21)
- useProviderActivity.ts: silent catch (line 36)
- useTasks.ts: console.warn + bare catches (lines 53, 79, 85)

**Fix:** Add an `error` ref to each composable that surfaces the last error. Set it in catch blocks. Consumers can watch/display it. Keep console.warn for debugging but add the error ref for programmatic access.

**Files:** `frontend/src/composables/useActivities.ts`, `useAISessions.ts`, `useDiscovery.ts`, `useProviderActivity.ts`, `useTasks.ts`
