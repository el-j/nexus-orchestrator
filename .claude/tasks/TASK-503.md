---
id: TASK-503
plan: PLAN-064
status: done
wave: 4
priority: 3
---

# TASK-503: Add GitHub Action full-flow integration test with mock daemon

## Description

`github-action/__tests__/index.test.ts` never exercises a real nexus API shape. All network calls are either skipped or mocked at too high a level to catch HTTP contract regressions. A proper integration test must stand up a mock HTTP server that speaks the real nexus API response schema and verify the action handles all terminal states.

## Checklist

- [ ] Create `github-action/__tests__/integration.test.ts`
- [ ] Spin up a Node `http.createServer` (or use `msw`) serving the nexus API shape: `POST /api/tasks` -> 201 with task stub; `GET /api/tasks/:id` -> 200 with progressively updated status per poll
- [ ] Test success flow: action submits task, polls until COMPLETED, exits 0, outputs task ID as action output
- [ ] Test timeout flow: mock server never returns COMPLETED within configured timeout; action exits with timeout error and non-zero code
- [ ] Test NO_PROVIDER terminal state: mock server returns task with status `FAILED` and error `"no provider available"`; action exits with descriptive error message
- [ ] Test network error: mock server closes connection on first GET; action retries at least once before failing
- [ ] Verify action output variables are set correctly on success (`task-id`, `result`)
- [ ] Server torn down in `afterAll` to prevent port leaks

## Files

- `github-action/__tests__/integration.test.ts` (create)
- `github-action/src/` (reference for action entry point)

## Acceptance Criteria

- 4 test cases (success, timeout, NO_PROVIDER, network error)
- Mock HTTP server uses real nexus API JSON shapes (not hand-waved stubs)
- `pnpm test` in `github-action/` exits 0 with all new tests green
- No real daemon required to run the tests
