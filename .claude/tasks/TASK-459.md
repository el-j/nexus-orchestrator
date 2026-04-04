---
id: TASK-459
plan: PLAN-060
status: todo
wave: 4
priority: 4
---

# TASK-459: Fix tests for router-aware components

Any spec files that mount AppSidebar or App.vue need a router mock. MissionControlView tests should be unaffected (no router usage). Verify all 25+ tests still pass.
