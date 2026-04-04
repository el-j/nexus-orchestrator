---
id: TASK-501
plan: PLAN-064
status: done
wave: 4
priority: 3
---

# TASK-501: Add execution engine provider failover test

## Description

No test exercises the scenario where the first available LLM provider fails during `Chat()` or `GenerateCode()` and a second provider must be tried. This failover path is critical for reliability in multi-provider production deployments but is entirely untested.

## Checklist

- [ ] Add test to `internal/core/services/execution_engine_test.go` (from TASK-493) or create `execution_engine_failover_test.go`
- [ ] Register two mock `LLMClient` providers: first returns `errors.New("timeout")` on every call; second returns a valid response
- [ ] Submit a task with no provider hint so both are eligible
- [ ] Assert task completes successfully using the second provider's response
- [ ] Assert the first provider's `Chat()` was called exactly once (not retried)
- [ ] Assert the second provider's `Chat()` was called exactly once
- [ ] Add a second sub-test: both providers fail -> task status set to FAILED with aggregated error message

## Files

- `internal/core/services/execution_engine_test.go` (extend) or `execution_engine_failover_test.go` (create)

## Acceptance Criteria

- 2 test cases: single-provider-fail -> fallback succeeds; all-providers-fail -> FAILED status
- Mock call counters verified (first provider called once, not repeatedly)
- `CGO_ENABLED=1 go test -race ./internal/core/services/...` exits 0
