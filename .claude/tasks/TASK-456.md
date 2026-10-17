---
id: TASK-456
plan: PLAN-060
status: closed
closedReason: 'stale: planned before v0.10.0 and never started; the work is tracked in PLAN-073 and PLAN-074 or was superseded'
wave: 2
priority: 2
---

# TASK-456: Refactor AppSidebar to use router

Drop emit + manual activeView ref. Use useRouter().push(path) and useRoute().path for active highlighting.
