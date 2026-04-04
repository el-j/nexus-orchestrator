---
id: TASK-258
title: "Verify: E2E validation of automatic task tracking"
role: verify
planId: PLAN-038
status: todo
dependencies: [TASK-257]
createdAt: 2026-03-13T16:00:00.000Z
---

## Context
Final verification that the full pipeline works end-to-end: register session → submit task → claim task → update status → verify session has routed task IDs.

## Files to Read
- `scripts/e2e-smoke.sh` (existing E2E patterns)
- All task files from PLAN-038

## Implementation Steps
1. Add claim/status flow to `scripts/e2e-smoke.sh`:
   - Register an AI session via POST /api/ai-sessions
   - Submit a task via POST /api/tasks
   - Claim the task via POST /api/tasks/{id}/claim with sessionId
   - Verify task status is PROCESSING and aiSessionId is set
   - Update task status to COMPLETED via PUT /api/tasks/{id}/status
   - Verify task status is COMPLETED
   - Query session tasks via GET /api/ai-sessions/{id}/tasks
   - Verify the completed task appears in the session's task list

2. Run `go vet ./...` and `CGO_ENABLED=1 go build ./...`
3. Run `CGO_ENABLED=1 go test -race -count=1 ./...`
4. Run the updated E2E smoke test
5. Run frontend test coverage
6. Run extension test coverage

## Acceptance Criteria
- [ ] E2E smoke test passes with claim/status flow
- [ ] `go vet ./...` clean
- [ ] `go test -race` all green
- [ ] Full validation stack passes
- [ ] No regressions in existing tests
