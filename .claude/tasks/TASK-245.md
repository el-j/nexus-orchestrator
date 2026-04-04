---
id: TASK-245
title: "Audit: synthesize final release-readiness artifact"
role: planning
planId: PLAN-036
status: done
dependencies: [TASK-244]
createdAt: 2026-03-13T09:59:36.000Z
completedAt: 2026-03-13T09:59:36.000Z
---

## Context
The audit needed a single durable artifact that prioritizes blockers, gaps, stubs, and remediation waves. This task consolidates the evidence and validated findings into the final release-readiness report used for follow-on planning.

## Files to Read
- `.claude/AUDIT-2026-03-13-release-readiness.md`
- `.claude/plans/PLAN-035.md`
- `.claude/tasks/TASK-239.md`
- `.claude/tasks/TASK-242.md`

## Implementation Steps

1. Consolidate the validated findings into the final release-readiness artifact.
2. Preserve the distinction between completed verification work and unresolved ship blockers.
3. Organize the artifact so remediation can be executed in ordered waves without re-running the audit discovery work.

## Acceptance Criteria
- [x] `go vet ./...` exits 0
- [x] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [x] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [x] A durable audit artifact exists with prioritized findings and remediation waves
- [x] The artifact distinguishes completed evidence from remaining release blockers and gaps

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging

## Result
Published the durable release-readiness audit artifact in `.claude`, capturing critical blockers, medium-risk mismatches, stub/TODO inventory, test gaps, and a recommended remediation sequence.