---
id: TASK-430
plan: PLAN-057
status: done
role: testing
wave: 7
---

# TASK-430: Vitest — AgentsView shows merged sessions + discovered agents

Write `frontend/src/test/AgentsView.spec.ts`:

- Renders registered sessions section with disconnect buttons
- Renders discovered agents section with detection method badges
- "Delegate All Active" button visible when sessions exist
- Purge disconnected button works
- Activity tab shows timeline when expanded (if TASK-423 is done)

**Files:** `frontend/src/test/AgentsView.spec.ts` (NEW)
**Depends on:** TASK-416
