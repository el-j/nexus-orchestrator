---
id: TASK-244
title: "Audit: cross-check findings against repo state"
role: verify
planId: PLAN-036
status: done
dependencies: [TASK-243]
createdAt: 2026-03-13T09:59:36.000Z
completedAt: 2026-03-13T09:59:36.000Z
---

## Context
The audit findings needed to be validated against the live repository state so the report did not drift from reality. This task cross-checked the findings, TODOs, contract mismatches, and orchestration metadata against current source and workflow files.

## Files to Read
- `.claude/AUDIT-2026-03-13-release-readiness.md`
- `.claude/orchestrator.json`
- `internal/core/services/orchestrator.go`
- `internal/adapters/outbound/repo_sqlite/repo.go`
- `app.go`

## Implementation Steps

1. Re-read the key runtime, workflow, and orchestration files that the audit claims depend on.
2. Confirm the verified blockers, stubs, and mismatches remain accurate in the repository snapshot used for the audit.
3. Record any orchestration-state inconsistency that must be corrected as part of the backfill itself.

## Acceptance Criteria
- [x] `go vet ./...` exits 0
- [x] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [x] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [x] The audit findings are cross-checked against the live repo state used during the audit session
- [x] The stale `activePlanId` inconsistency is identified explicitly for correction

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging

## Result
Validated the audit findings against the live repo snapshot and confirmed the registry inconsistency where a completed plan remained active, making that metadata correction part of the tracked audit closeout.