---
id: TASK-494
plan: PLAN-064
status: done
wave: 2
priority: 1
---

# TASK-494: Add useTasks composable tests

## Description

`frontend/src/composables/useTasks.ts` is the primary task mutation surface for the entire GUI. It provides the `tasks` ref, `submitTask`, `cancelTask`, and `promoteTask` mutations consumed by MissionControlView, BacklogView, and HistoryView. It has zero tests. Functional regressions here silently break the entire task lifecycle in the UI.

## Checklist

- [ ] Create `frontend/src/composables/__tests__/useTasks.test.ts`
- [ ] Test initial fetch: `fetchTasks()` called on mount; `tasks.value` populated from mocked API response; `isLoading` set false after resolution
- [ ] Test SSE event: when a `task-updated` SSE event is received, the matching task in `tasks.value` is updated in place without a full re-fetch
- [ ] Test `cancelTask`: correct HTTP DELETE (or Wails call) is triggered with the task ID; task removed from `tasks.value` on success; `error.value` set on 500 response
- [ ] Test `promoteTask`: correct API call made with task ID; task status updated to `QUEUED` in `tasks.value`; error path sets `error.value`
- [ ] Test error path for initial fetch: API returns 500 -> `error.value` is set with message; `tasks.value` remains empty array
- [ ] Mock Wails runtime (`window.go.*`) using `vi.mock` or `vi.fn()` stubs, matching style in `useActivities.test.ts`
- [ ] All tests pass with `pnpm vitest run`

## Files

- `frontend/src/composables/__tests__/useTasks.test.ts` (create)
- `frontend/src/composables/useTasks.ts` (reference)
- `frontend/src/composables/__tests__/useActivities.test.ts` (reference for mock style)

## Acceptance Criteria

- Minimum 5 test cases covering the scenarios above
- No modification to `useTasks.ts` required (test against public API surface)
- `pnpm vitest run` exits 0 with all new tests green
