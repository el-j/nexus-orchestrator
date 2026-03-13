---
id: TASK-246
title: "Planning: reconcile .claude registry after audit closeout"
role: planning
planId: PLAN-036
status: done
dependencies: [TASK-245]
createdAt: 2026-03-13T09:59:36.000Z
completedAt: 2026-03-13T09:59:36.000Z
---

## Context
The audit identified a metadata-integrity problem inside `.claude` itself. This task closes the loop by recording the audit as completed work and removing the stale pointer to a plan that had already finished.

## Files to Read
- `.claude/orchestrator.json`
- `.claude/plans/PLAN-036.md`
- `.claude/tasks/TASK-243.md`
- `.claude/AUDIT-2026-03-13-release-readiness.md`

## Implementation Steps

1. Register PLAN-036 and TASK-243 through TASK-246 in `.claude/orchestrator.json` as completed retrospective records.
2. Clear the stale `activePlanId` so the registry no longer points at completed work.
3. Advance the counters and update the notes field so the audit closeout is visible in orchestration history.

## Acceptance Criteria
- [x] `go vet ./...` exits 0
- [x] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [x] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [x] `.claude/orchestrator.json` contains PLAN-036 and TASK-243 through TASK-246 as completed records
- [x] `.claude/orchestrator.json` no longer points `activePlanId` at completed PLAN-035

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging

## Result
Reconciled the `.claude` registry so the release-readiness audit now exists as a completed plan with completed task records, and the stale active-plan pointer to PLAN-035 is cleared.