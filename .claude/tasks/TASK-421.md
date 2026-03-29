---
id: TASK-421
plan: PLAN-057
status: todo
role: frontend
wave: 4
---

# TASK-421: Add "last refreshed" indicator to every view header

Every view that fetches data should show a "Last refreshed: Xm ago" or "⟳ 3s ago" timestamp in the header area. This eliminates the "stale" perception — users can see exactly when data was last fetched.

Implement via a reusable `FreshnessIndicator.vue` component or a `useFreshness()` composable that tracks the last successful fetch timestamp and formats it as relative time.

**Files:** new component or composable, integrate into all view headers
**Depends on:** none
