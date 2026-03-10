---
id: TASK-073
title: QA verification (full test suite + build + smoke pass)
role: qa
planId: PLAN-008
status: todo
dependencies: [TASK-071, TASK-072]
createdAt: 2026-03-10T14:00:00.000Z
---

## Context
Final verification task for PLAN-008. Run full test suite, verify all new tests pass under `-race`, confirm the E2E smoke script is executable and syntactically valid, and validate the new command file is well-formed.

## Files to Read
- `internal/core/services/integration_test.go`
- `internal/adapters/inbound/mcp/integration_test.go`
- `scripts/e2e-smoke.sh`
- `.claude/commands/execute-via-nexus.md`

## Implementation Steps

1. Run `go vet ./...` — must exit 0.
2. Run `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` — must exit 0.
3. Run `CGO_ENABLED=1 go test -race -count=1 ./...` — all tests must pass.
4. Count total test functions across all `_test.go` files — report the number.
5. Verify `scripts/e2e-smoke.sh` has `+x` permission and passes `bash -n scripts/e2e-smoke.sh` (syntax check).
6. Verify `.claude/commands/execute-via-nexus.md` exists and contains the required sections (Steps, Constraints).
7. Report summary of all PLAN-008 deliverables.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] All new tests from TASK-068, TASK-069, TASK-070 pass under -race
- [ ] `scripts/e2e-smoke.sh` passes syntax check
- [ ] `.claude/commands/execute-via-nexus.md` is well-formed

## Anti-patterns to Avoid
- NEVER modify source files in a QA verification task
- NEVER skip the -race flag
