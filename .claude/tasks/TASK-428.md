---
id: TASK-428
plan: PLAN-057
status: done
role: frontend
wave: 6
---

# TASK-428: ProvidersView — show latency, rate-limit badge, circuit-breaker state

Enhance the ProvidersView provider cards to show the new operational metadata from TASK-407/426/427:

- Latency (e.g. "120ms")
- Rate-limited badge (red "Rate Limited" when flag is set)
- Circuit-breaker state ("3 consecutive failures")
- Context window size (e.g. "ctx: 32k")
- "Last checked 15s ago" freshness indicator

**Files:** `frontend/src/views/ProvidersView.vue`, `frontend/src/components/ProviderStatus.vue`
**Depends on:** TASK-407, TASK-426, TASK-427
