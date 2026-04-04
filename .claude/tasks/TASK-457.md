---
id: TASK-457
plan: PLAN-060
status: todo
wave: 2
priority: 2
---

# TASK-457: Simplify App.vue to RouterView shell

Remove currentView ref, the v-if chain, and the @view-change handler. Keep only <RouterView />, LogPanel, Toast, ConfirmDialog, ErrorFallback, SSE connect.
