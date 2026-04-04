---
id: TASK-484
plan: PLAN-063
status: done
wave: 1
priority: 2
---

# TASK-484: Fix SessionMonitor — remove auto-claim behaviour

Completed. Renamed `pollAndClaim()` to `refreshQueue()`, removed `claimTask()` call. The extension now only reads queue state for display/logging, never claims tasks. Timer interval changed from 10s to 30s.
