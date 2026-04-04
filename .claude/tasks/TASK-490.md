---
id: TASK-490
plan: PLAN-063
status: done
wave: 3
priority: 2
---

# TASK-490: Add `sessionMonitor` unit tests

## Problem

`vscode-extension/src/sessionMonitor.ts` is the most stateful and complex file in the extension; it manages registration retries, heartbeat intervals, and deregistration on deactivation. It has zero tests, making the TASK-484 refactor risky and future changes fragile.

## Checklist

- [ ] Create `vscode-extension/src/test/sessionMonitor.test.ts`
- [ ] Mock `nexusClient` with a vitest/jest spy; control which calls succeed or fail
- [ ] Test case: `detectAndRegister()` fails on first attempt then succeeds on second → session ID is stored and heartbeat starts
- [ ] Test case: `detectAndRegister()` fails 3 consecutive times → no session ID stored, no heartbeat timer running
- [ ] Test case: `start()` followed by `stop()` → heartbeat interval is cleared and `deregisterSession()` is called exactly once
- [ ] Test case: heartbeat call fails → a warning is logged, interval continues (does not crash)
- [ ] After TASK-484: test that `startPolling()` does NOT call `claimTask()` under any conditions
- [ ] Test timer management: multiple calls to `start()` without `stop()` do not leak intervals

## Files to change

- `vscode-extension/src/test/sessionMonitor.test.ts` (new)
- `vscode-extension/src/sessionMonitor.ts` (minor refactor to injectable clock/timer if needed for test isolation)

## Acceptance criteria

- [ ] All new tests pass under `npm test`
- [ ] Timer leak test confirms no duplicate intervals on repeated `start()` calls
- [ ] Post-TASK-484: `claimTask` spy is never called during normal monitor lifecycle
