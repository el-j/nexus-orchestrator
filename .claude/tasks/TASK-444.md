---
id: TASK-444
plan: PLAN-059
status: todo
wave: 2
priority: 2
---

# TASK-444: useGlobalSSE.ts — cleanup, disconnect, backoff

**Problem:** Module-level singleton EventSource + handlers Map with no cleanup. No exported disconnect function. Handlers accumulate across mounts. 3s reconnect with no backoff — can hammer server. Parse errors silently ignored.

**Fix:**

1. Export a `disconnect()` function that closes EventSource and clears handlers
2. Add exponential backoff on reconnect (3s → 6s → 12s → max 30s), reset on success
3. Ensure `off()` actually removes handlers correctly (verify Map key cleanup)
4. Log parse errors to console.warn (not silent swallow)

**Files:** `frontend/src/composables/useGlobalSSE.ts`
