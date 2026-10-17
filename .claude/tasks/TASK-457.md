---
id: TASK-457
plan: PLAN-060
status: closed
closedReason: 'stale: planned before v0.10.0 and never started; the work is tracked in PLAN-073 and PLAN-074 or was superseded'
wave: 2
priority: 2
---

# TASK-457: Simplify App.vue to RouterView shell

Remove currentView ref, the v-if chain, and the @view-change handler. Keep only <RouterView />, LogPanel, Toast, ConfirmDialog, ErrorFallback, SSE connect.
