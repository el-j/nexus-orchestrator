---
id: TASK-229
title: Verify full CI green locally
role: verify
planId: PLAN-031
status: done
dependencies: [TASK-226, TASK-227, TASK-228]
createdAt: 2026-03-12T17:00:00.000Z
---

## Context
After fixing the duplicate package declaration, applying go fmt, and hardening the CI YAML, this task provides a full local verification pass mirroring exactly what CI runs. It is the final gate before pushing the `feature/next-level` branch.

## Files to Read
- `.github/workflows/ci.yml` (to mirror exact CI commands)
- `internal/adapters/outbound/llm_openaicompat/adapter_test.go` (confirm fix)

## Implementation Steps
1. Run `go vet ./...` — must exit 0.
2. Run `gofmt -l ./...` — must produce no output.
3. Run `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-submit/... 2>&1` — must exit 0.
4. Run `CGO_ENABLED=1 go build ./cmd/nexus-daemon/... 2>&1` — must exit 0.
5. Run `CGO_ENABLED=1 go test -race -count=1 ./... 2>&1` — must exit 0 with no FAIL lines.
6. Confirm `internal/adapters/outbound/llm_openaicompat` appears in test output with `ok` status.
7. Report a pass/fail summary for each check.

## Acceptance Criteria
- All 5 commands above exit 0.
- `gofmt -l ./...` produces no output.
- `go test` output contains `ok  nexus-orchestrator/internal/adapters/outbound/llm_openaicompat` with no `FAIL` lines.
- No data races detected under `-race`.
