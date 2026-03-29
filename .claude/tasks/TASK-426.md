---
id: TASK-426
plan: PLAN-057
status: todo
role: backend
wave: 6
---

# TASK-426: Track provider response latency in discovery service

Instrument `Ping()` calls in the discovery health-check loop to record response time. Store as `LatencyMs int` in the health snapshot. Expose in `ProviderInfo` JSON response.

This gives the frontend data to show "p50 latency: 120ms" on provider cards. Track only the most recent ping latency (no histogram needed for v1).

**Files:** `internal/core/services/discovery.go`, `internal/core/ports/ports.go`
**Depends on:** TASK-407
