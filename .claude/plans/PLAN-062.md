# PLAN-062 — Frontend: Pre-Launch Critical Fixes

**Status:** active
**Created:** 2026-04-04
**Author:** copilot

## Summary

Audit-driven critical and high-priority fixes for the Vue 3 frontend identified during the pre-launch review. Covers five categories of production-blocking issues:

- Broken Wails desktop bindings (runtime config calls hit HTTP with no `isWails()` guard)
- Redundant SSE connections (5-6 concurrent EventSources instead of one shared bus)
- Permanent disconnects in composables with no reconnect logic
- UX regressions: hardcoded status indicators, silent failures, `window.confirm` in webview
- Dead code, incomplete UI actions, and missing destructive-operation safety

## Waves

### Wave 1 — Critical: Infrastructure & Connectivity (build-blocking)

| Task     | Description                                                                |
| -------- | -------------------------------------------------------------------------- |
| TASK-470 | Fix wails.ts getRuntimeConfig/updateRuntimeConfig — broken in desktop mode |
| TASK-471 | Fix useGlobalSSE — currently bypassed, 5-6 concurrent SSE connections      |
| TASK-472 | Fix useLogs SSE — no reconnect causes permanent disconnect                 |

### Wave 2 — Critical: Visible UX Regressions

| Task     | Description                                                                       |
| -------- | --------------------------------------------------------------------------------- |
| TASK-473 | Fix AppSidebar daemon status — always green hardcoded pulse                       |
| TASK-474 | Fix BacklogView error handling — no try/catch on refresh                          |
| TASK-475 | Fix SettingsView — hardcoded server addresses and silent token operation failures |
| TASK-476 | Replace window.confirm with component-level confirmation dialogs                  |

### Wave 3 — High: Correctness & Completeness

| Task     | Description                                                        |
| -------- | ------------------------------------------------------------------ |
| TASK-477 | Remove/wire AgentDetailDrawer dead code                            |
| TASK-478 | Fix AIActivityCard — undefined :class binding                      |
| TASK-479 | Fix useProjectFilter — misses completed projects                   |
| TASK-480 | Fix DiscoveredPlansView — display-only cards with no actions       |
| TASK-481 | Fix TaskDetailDrawer — no cancel for PROCESSING tasks              |
| TASK-482 | Replace all console.warn/console.error with structured error state |

## Validation

- `vue-tsc --noEmit` zero errors after all waves
- `pnpm -C frontend vitest run` all pass
- Browser DevTools Network tab shows exactly **one** `EventSource` connection at idle
- Desktop (Wails) mode: SettingsView loads runtime config without 404 errors in Go console
- AppSidebar status dot turns red within 5 s of daemon shutdown
- No `window.confirm` calls remain in production bundles (`grep -r 'window.confirm' frontend/src` returns empty)
- All destructive actions show `AppConfirmDialog` modal before proceeding
- No `console.warn` or `console.error` calls in `frontend/src` outside test files
