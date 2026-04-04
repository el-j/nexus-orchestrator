---
id: TASK-446
plan: PLAN-059
status: todo
wave: 3
priority: 3
---

# TASK-446: ProvidersView handlePromote() — pre-fill form, remove debug log

**Problem:** Lines 525-530: handlePromote() opens form but doesn't pre-fill with discovered provider data. Has `console.log('Promote discovered provider:', ...)` debug statement.

**Fix:**

1. Remove `console.log('Promote discovered provider:', ...)` debug statement
2. Pre-fill the provider config form with discovered provider's baseUrl, name, kind when opening the promote dialog

**Files:** `frontend/src/views/ProvidersView.vue`
