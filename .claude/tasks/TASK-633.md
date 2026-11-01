---
id: TASK-633
planId: PLAN-077
title: 'Replace the five `as any` escapes with real types'
role: frontend
status: todo
createdAt: 2026-10-31
---

# TASK-633 — Replace the five `as any` escapes with real types

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

`DiscoveredToolsSection.vue` (2), `AgentsView.vue`, `ProvidersView.vue`,
`docs/src/views/DownloadsView.vue` (navigator.userAgentData). Use union types for the filter options
and a declared `NavigatorUAData` type. Add a check that fails on `as any` outside tests.
