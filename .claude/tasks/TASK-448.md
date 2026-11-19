---
id: TASK-448
plan: PLAN-059
status: closed
closedReason: 'stale: planned before v0.10.0 and never started; the work is tracked in PLAN-073 and PLAN-074 or was superseded'
wave: 4
priority: 4
---

# TASK-448: MissionControlView — toast on promote failure

**Problem:** Line 314: `console.warn('Promote failed:', e)` — user never sees the error.

**Fix:** Replace `console.warn` with `toast.add({ severity: 'error', summary: 'Promote Failed', detail: e.message, life: 5000 })`.

**Files:** `frontend/src/views/MissionControlView.vue`
