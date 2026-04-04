---
id: TASK-495
plan: PLAN-064
status: done
wave: 2
priority: 1
---

# TASK-495: Add useGlobalSSE composable tests

## Description

`frontend/src/composables/useGlobalSSE.ts` is the real-time event bus for the entire application. Task updates, activity feeds, and session changes all arrive through it. It has zero tests. A regression here would silently freeze all live-update views without any error surfacing.

## Checklist

- [ ] Create `frontend/src/composables/__tests__/useGlobalSSE.test.ts`
- [ ] Test connect on mount: `EventSource` constructor called with correct URL; `onmessage` handler registered
- [ ] Test event routing: message event with `type: "task"` dispatched to all `task` subscribers; message with `type: "activity"` dispatched to `activity` subscribers only; unrecognised type is silently ignored
- [ ] Test subscriber cleanup: handler added via `onEvent("task", fn)` then removed; after removal, `fn` is not called for subsequent `task` events
- [ ] Test disconnect on unmount: `EventSource.close()` called when component using composable is unmounted; no further events processed after close
- [ ] Test reconnect with exponential backoff: `EventSource.onerror` triggered; reconnect attempted after initial delay; second failure doubles delay; backoff capped at max configured value
- [ ] Mock `window.EventSource` with a `vi.fn()` factory returning a controllable stub
- [ ] All tests pass with `pnpm vitest run`

## Files

- `frontend/src/composables/__tests__/useGlobalSSE.test.ts` (create)
- `frontend/src/composables/useGlobalSSE.ts` (reference)

## Acceptance Criteria

- Minimum 6 test cases covering the scenarios above
- `EventSource` is fully mocked - no real network connections in tests
- `pnpm vitest run` exits 0 with all new tests green
