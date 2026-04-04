---
id: TASK-504
plan: PLAN-064
status: done
wave: 4
priority: 3
---

# TASK-504: E2E smoke test hardening

## Description

`scripts/e2e-smoke.sh` and `internal/e2e/` tests currently verify basic HTTP response codes but do not inspect SSE event content, do not test auth token enforcement end-to-end, do not test concurrent multi-task submission, and have no coverage of provider failover at the system level. These gaps mean a significant class of integration regressions could pass smoke tests undetected.

## Checklist

- [ ] Add SSE event content verification to `internal/e2e/` or `scripts/e2e-smoke.sh`: subscribe to SSE stream; submit a task; assert at least one `task` event received containing the submitted task ID; assert event JSON is well-formed
- [ ] Add concurrent submission test: submit 3 tasks in parallel via goroutines (or background curl); assert all 3 appear in `GET /api/tasks` with unique IDs; assert no task ID collisions
- [ ] Add auth token enforcement test: if auth is configured, send a request with invalid/missing token; assert 401 or 403 response; assert no task created
- [ ] Add a placeholder test for real-LLM provider failover with a `t.Skip("requires live LLM providers")` marker and a comment documenting the expected behaviour and how to run manually
- [ ] Ensure all new e2e tests are guarded by a build tag `//go:build e2e` so they do not run in unit test CI by default
- [ ] Update `scripts/e2e-smoke.sh` header comment to document new test coverage

## Files

- `internal/e2e/smoke_test.go` (extend) or `internal/e2e/sse_test.go` (create)
- `scripts/e2e-smoke.sh` (extend)

## Acceptance Criteria

- SSE event content verified in at least one automated test
- Concurrent 3-task submission test present and non-flaky
- Real-LLM skip marker present with explanatory comment
- `CGO_ENABLED=1 go test -tags e2e ./internal/e2e/...` passes against a running daemon
