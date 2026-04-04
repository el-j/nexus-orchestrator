---
id: TASK-449
plan: PLAN-059
status: todo
wave: 4
priority: 4
---

# TASK-449: BacklogList + ProviderStatus — toast on errors

**Problem:** Both components log errors via console.warn. User never sees failures in the UI.

**Fix:**

1. In BacklogList.vue: add `useToast()` and show toast on promote/dismiss/delete errors
2. In ProviderStatus.vue: add `useToast()` and show toast on loadConfigs error

**Files:** `frontend/src/components/BacklogList.vue`, `frontend/src/components/ProviderStatus.vue`
