---
id: TASK-573
title: Frontend composable tests — useProjectFilter, useProviderActivity, useProviders, useServerUrl
role: qa
planId: PLAN-071
status: todo
dependencies: [TASK-561, TASK-562]
createdAt: 2026-04-15T00:00:00Z
---

## Context

Second batch of four untested frontend composables: `useProjectFilter`, `useProviderActivity`, `useProviders`, `useServerUrl`. `useServerUrl` is used by almost every view to resolve the daemon address — a regression here would silently break all HTTP calls. `useProviders` drives the providers panel and auto-refresh logic.

## Files to Read

- `frontend/src/composables/useProjectFilter.ts`
- `frontend/src/composables/useProviderActivity.ts`
- `frontend/src/composables/useProviders.ts`
- `frontend/src/composables/useServerUrl.ts`
- Any existing `*.test.ts` in composables for test setup patterns

## Implementation Steps

1. Create `frontend/src/composables/useServerUrl.test.ts`:
   - `test('returns cached URL on second call without re-fetching')`
   - `test('calls getServerAddr when cache is empty')`
   - `test('returns VITE_SERVER_URL if set in dev mode')`
   - Reset module cache between tests with `vi.resetModules()`
2. Create `frontend/src/composables/useProjectFilter.test.ts`:
   - `test('filters tasks by selected project')`
   - `test('returns all tasks when no project selected')`
   - `test('currentProject changes trigger re-filter')`
3. Create `frontend/src/composables/useProviders.test.ts`:
   - `test('loads providers on mount')`
   - `test('promote() calls promoteProvider and refreshes list')`
   - `test('error ref set on load failure')`
   - `test('configuredProviders are sorted active-first')`
4. Create `frontend/src/composables/useProviderActivity.test.ts`:
   - `test('fetches activity entries for a provider')`
   - `test('error ref set on failure')`

## Acceptance Criteria

- [ ] `npm test` passes in `frontend/` with all new tests
- [ ] `vue-tsc --noEmit` exits 0
- [ ] All 4 composable test files exist with ≥ 2 tests each
- [ ] `useServerUrl` cache behaviour is tested (not just first call)
- [ ] `useProviders` sorting is tested (active-first)

## Anti-patterns to Avoid

- NEVER share module-level state between tests without `vi.resetModules()`
- NEVER use real `fetch` — mock via `vi.spyOn(global, 'fetch')`
