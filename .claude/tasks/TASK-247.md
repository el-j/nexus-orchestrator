---
id: TASK-247
title: "Backend: unify queued-task authority and admission path"
role: backend
planId: PLAN-037
status: done
dependencies: []
createdAt: 2026-03-13T10:17:51.000Z
---

## Context
Queued-task execution currently depends on both persisted state and an in-memory queue, and multiple public paths can bypass the same admission rules. This task makes persisted `QUEUED` state authoritative and ensures every transition into `QUEUED` follows one validated path.

## Files to Read
- `internal/core/services/orchestrator.go`
- `internal/core/ports/ports.go`
- `internal/adapters/outbound/repo_sqlite/repo.go`
- `internal/core/services/orchestrator_test.go`
- `internal/core/services/orchestrator_hardening_test.go`

## Implementation Steps

1. Add repository support for claiming the next persisted queued task and for guarded status transitions.
2. Refactor orchestration logic so persisted `QUEUED` tasks drive worker processing and restart recovery, not the in-memory queue slice.
3. Centralize transition-to-`QUEUED` validation for `SubmitTask` and `PromoteTask`, and stop `UpdateTask` from bypassing that path.
4. Add tests for restart recovery of persisted queued work, post-restart cancellation, queue-cap enforcement through promotion, and rejection of `UpdateTask` forcing `QUEUED`.

## Acceptance Criteria
- [x] `go vet ./...` exits 0
- [x] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [x] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [x] Persisted `QUEUED` tasks are executable after restart without reconstructing an in-memory queue
- [x] `SubmitTask` and `PromoteTask` enforce the same admission rules for transitions into `QUEUED`

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging

## Result
- Added `ClaimNextQueued` and `UpdateStatusIfCurrent` to the task repository contract and SQLite implementation so persisted queued rows are now the execution source of truth.
- Refactored orchestration to claim work from storage, centralize queue admission for submit/promote, reject direct `QUEUED` mutation through `UpdateTask`, and recover queued/processing work correctly on restart.
- Added regression coverage for persisted queued cancellation, startup recovery, guarded queue admission, and deterministic pending-queue semantics under `-race`.