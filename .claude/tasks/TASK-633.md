---
id: TASK-633
planId: PLAN-077
title: 'Replace the five `as any` escapes with real types'
role: frontend
status: done
createdAt: 2026-10-31
---

# TASK-633 — Replace the five `as any` escapes with real types

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

`DiscoveredToolsSection.vue` (2), `AgentsView.vue`, `ProvidersView.vue`,
`docs/src/views/DownloadsView.vue` (navigator.userAgentData). Use union types for the filter options
and a declared `NavigatorUAData` type. Add a check that fails on `as any` outside tests.

Done: the filter options of `DiscoveredToolsSection`, `AgentsView` and `ProvidersView` are typed arrays (`ToolStatusFilter`, `SessionStatusFilter`, `ProviderStatusFilter`); the agent group header uses `kindLabel(kind)`; the docs site declares `NavigatorWithUAData`. The audit's grep missed three `catch (e: any)` blocks in `vscode-extension/src/extension.ts`: they use the new `errorText(err: unknown)` helper. `internal/docscheck/typescript_types_test.go` now fails on `as any`, `: any` or `<any>` in application code of the four TypeScript packages.
