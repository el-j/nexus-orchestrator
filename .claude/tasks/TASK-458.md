---
id: TASK-458
plan: PLAN-060
status: closed
closedReason: 'stale: planned before v0.10.0 and never started; the work is tracked in PLAN-073 and PLAN-074 or was superseded'
wave: 3
priority: 3
---

# TASK-458: ProjectActivityView — replace emit with router.push

Replace `emit('navigate', 'live-activity')` with `useRouter().push('/projects/live-activity')`. Remove defineEmits.
