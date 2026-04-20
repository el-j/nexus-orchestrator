---
id: TASK-572
title: Frontend composable tests — useBrain, useDaemonHealth, useDiscovery, useLogs
role: qa
planId: PLAN-071
status: todo
dependencies: [TASK-561, TASK-562]
createdAt: 2026-04-15T00:00:00Z
---

## Context

Eight frontend composables have zero tests. This task covers the first four: `useBrain`, `useDaemonHealth`, `useDiscovery`, `useLogs`. `useBrain` is the most critical — it was the source of the Vue ref auto-unwrap bug that crashed BrainView. `useDaemonHealth` drives the connection indicator. All should be tested with mocked fetch.

## Files to Read

- `frontend/src/composables/useBrain.ts`
- `frontend/src/composables/useDaemonHealth.ts`
- `frontend/src/composables/useDiscovery.ts`
- `frontend/src/composables/useLogs.ts`
- Any existing `*.test.ts` in `frontend/src/composables/` for test patterns

## Implementation Steps

1. Create `frontend/src/composables/useBrain.test.ts`:
   - Mock `../types/wails` module with `vi.mock`
   - `test('fetchStatus sets status on success')` — mock `getBrainStatus` → assert `status.value` updated
   - `test('fetchStatus sets error on failure')` — mock throws → assert `error.value` set, `loading.value` false
   - `test('init calls initProject and updates status')`
   - `test('search calls searchKnowledge and updates searchResults')`
   - `test('listEntries calls listKnowledge and updates entries')`
   - `test('deleteEntry removes item from entries and refreshes status')`
2. Create `frontend/src/composables/useDaemonHealth.test.ts`:
   - Mock `fetch` globally with `vi.spyOn`
   - `test('connected becomes true when /api/health returns 200')`
   - `test('connected stays false after FAILURE_THRESHOLD consecutive failures')`
   - `test('connected stays true until threshold exceeded')` — 2 failures should not flip it
3. Create `frontend/src/composables/useDiscovery.test.ts`:
   - `test('loads discovered providers on mount')`
   - `test('scan() triggers POST and refreshes list')`
   - `test('error ref set on fetch failure')`
4. Create `frontend/src/composables/useLogs.test.ts`:
   - `test('fetches logs and populates entries')`
   - `test('error ref set on non-ok response')`

## Acceptance Criteria

- [ ] `npm test` passes in `frontend/` with all new tests
- [ ] `vue-tsc --noEmit` exits 0
- [ ] `useBrain.test.ts` covers fetchStatus, init, search, listEntries, deleteEntry (success + error paths)
- [ ] `useDaemonHealth.test.ts` covers threshold logic
- [ ] All 4 composable test files exist with ≥ 2 tests each

## Anti-patterns to Avoid

- NEVER test implementation internals — test behaviour via the returned reactive refs
- NEVER use `setTimeout` / real timers in tests — use `vi.useFakeTimers()`
