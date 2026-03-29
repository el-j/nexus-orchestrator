---
id: TASK-431
plan: PLAN-057
status: todo
role: testing
wave: 7
---

# TASK-431: Vitest — ProvidersView enriched provider info

Write or extend `frontend/src/test/ProvidersView.spec.ts`:

- Provider cards show context limit value
- Provider cards show latency when available
- Rate-limited badge appears when provider is rate limited
- Circuit-breaker state shows consecutive fail count
- "Last checked Xm ago" freshness indicator renders
- Discovery section renders (formerly separate DiscoveryView)
- Configuration section renders (merged from Settings)

**Files:** `frontend/src/test/ProvidersView.spec.ts` (NEW or extend)
**Depends on:** TASK-415, TASK-418, TASK-428
