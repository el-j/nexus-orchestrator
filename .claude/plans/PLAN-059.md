# PLAN-059 — Comprehensive Codebase Hardening & Gap Closure

**Status:** active
**Created:** 2026-03-29
**Author:** copilot

## Summary

Full audit-driven hardening plan covering all gaps found by 4-agent parallel audit:

- Black screen bug (LiveActivityView not wired)
- Dead/orphaned code cleanup
- Memory leaks in composables
- Type safety fixes
- Stub/placeholder completion
- Error handling standardization (frontend + backend)
- Go backend JSON encoding safety

## Waves

### Wave 1 — Critical Wiring (user-visible bug)

| Task     | Description                                                      |
| -------- | ---------------------------------------------------------------- |
| TASK-441 | Register LiveActivityView + HistoryView + BacklogView in App.vue |
| TASK-442 | Delete dead DashboardView.vue                                    |

### Wave 2 — Memory Leaks & Cleanup

| Task     | Description                                                                   |
| -------- | ----------------------------------------------------------------------------- |
| TASK-443 | useDiscoveredPlans.ts — store interval ID, clear in onUnmounted               |
| TASK-444 | useGlobalSSE.ts — add disconnect export, exponential backoff, handler cleanup |

### Wave 3 — Type Safety & Stubs

| Task     | Description                                                                            |
| -------- | -------------------------------------------------------------------------------------- |
| TASK-445 | wails.ts HeartbeatAISession return type fix (Promise<Error> → Promise<void>)           |
| TASK-446 | ProvidersView handlePromote() — pre-fill form with discovered data, remove console.log |
| TASK-447 | Consolidate DiscoveredProvider type (domain.ts vs discovery.ts)                        |

### Wave 4 — Frontend Error Handling

| Task     | Description                                                             |
| -------- | ----------------------------------------------------------------------- |
| TASK-448 | MissionControlView — replace console.warn with toast on promote fail    |
| TASK-449 | BacklogList + ProviderStatus — add toast on errors                      |
| TASK-450 | Composable error handling — replace silent catches with error surfacing |

### Wave 5 — Go Backend Hardening

| Task     | Description                                                  |
| -------- | ------------------------------------------------------------ |
| TASK-451 | Fix 44 silent `_ = json.NewEncoder(w).Encode(...)` calls     |
| TASK-452 | Tray adapter — document as stub/N/A with TODO                |
| TASK-453 | Replace critical log.Printf anti-patterns with error returns |

## Validation

- `vue-tsc --noEmit` zero errors
- All Vitest tests pass
- `go vet ./...` clean
- `CGO_ENABLED=1 go test -race ./...` all pass
