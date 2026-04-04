---
id: TASK-447
plan: PLAN-059
status: todo
wave: 3
priority: 3
---

# TASK-447: Consolidate DiscoveredProvider type

**Problem:** `DiscoveredProvider` is defined in both `domain.ts` and `discovery.ts`, creating a sync risk if one is updated but not the other.

**Fix:**

1. Keep the canonical definition in `domain.ts`
2. In `discovery.ts`, re-export from domain.ts or remove the duplicate
3. Update all imports to use the canonical source

**Files:** `frontend/src/types/domain.ts`, `frontend/src/types/discovery.ts`
