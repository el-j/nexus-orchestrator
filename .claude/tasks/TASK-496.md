---
id: TASK-496
plan: PLAN-064
status: done
wave: 2
priority: 1
---

# TASK-496: Add useAISessions composable tests

## Description

`frontend/src/composables/useAISessions.ts` manages the AI session lifecycle surface in the GUI — listing sessions, deregistering them, and sending heartbeats. It has zero tests. Failures here would silently prevent the AgentsView from reflecting live session state.

## Checklist

- [ ] Create `frontend/src/composables/__tests__/useAISessions.test.ts`
- [ ] Test fetch on mount: `sessions.value` populated from mocked API response; `isLoading` transitions false->true->false
- [ ] Test `deregisterSession`: correct Wails/API call made with session ID; session removed from `sessions.value` on success
- [ ] Test heartbeat interval: `vi.useFakeTimers()`; advance clock by heartbeat interval; verify heartbeat call issued once per interval; verify no duplicate calls
- [ ] Test SSE update: receiving a `session-updated` event with a known session ID updates the matching entry in `sessions.value`
- [ ] Test error state on fetch failure: API throws -> `error.value` is non-null; `sessions.value` remains `[]`
- [ ] All tests pass with `pnpm vitest run`

## Files

- `frontend/src/composables/__tests__/useAISessions.test.ts` (create)
- `frontend/src/composables/useAISessions.ts` (reference)

## Acceptance Criteria

- Minimum 5 test cases covering the scenarios above
- Fake timers used for heartbeat interval test (no real timers/sleeps)
- `pnpm vitest run` exits 0 with all new tests green
