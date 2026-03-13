---
id: TASK-243
title: "Audit: collect release-readiness evidence"
role: qa
planId: PLAN-036
status: done
dependencies: []
createdAt: 2026-03-13T09:59:36.000Z
completedAt: 2026-03-13T09:59:36.000Z
---

## Context
The 2026-03-13 release-readiness audit depended on concrete verification evidence rather than opinion. This task captures the command outcomes and audit inputs that established the baseline for the final report.

## Files to Read
- `.claude/AUDIT-2026-03-13-release-readiness.md`
- `.github/workflows/ci.yml`
- `.github/workflows/publish.yml`
- `scripts/e2e-smoke.sh`

## Implementation Steps

1. Gather the completed verification evidence for `go vet`, race-tested Go suites, daemon E2E, frontend coverage, extension coverage, and Go coverage output.
2. Confirm the audit references the same verified surfaces that were exercised during the release-gate work.
3. Preserve the evidence trail in `.claude` so later remediation work can point back to a factual baseline.

## Acceptance Criteria
- [x] `go vet ./...` exits 0
- [x] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [x] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [x] Release-gate verification evidence is captured for Go, daemon E2E, frontend coverage, and extension coverage
- [x] The audit record references the same verified surfaces that were exercised on 2026-03-13

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging

## Result
Captured the existing release-gate evidence set and tied it to the audit artifact, including successful `go vet`, Go race tests, daemon E2E, frontend coverage, extension coverage, and Go coverage verification performed on 2026-03-13.