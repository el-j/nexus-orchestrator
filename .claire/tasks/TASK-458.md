---
id: TASK-458
plan: PLAN-060
status: todo
wave: 3
priority: 3
---

# TASK-458: ProjectActivityView — replace emit with router.push

Replace `emit('navigate', 'live-activity')` with `useRouter().push('/projects/live-activity')`. Remove defineEmits.
