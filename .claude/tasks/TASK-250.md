---
id: TASK-250
title: "Verification: validate Wave 1 release-readiness hardening"
role: verify
planId: PLAN-037
status: done
dependencies: [TASK-247, TASK-248, TASK-249]
createdAt: 2026-03-13T10:17:51.000Z
---

## Context
Wave 1 changes affect backend execution, release workflow safety, and provider persistence. This task verifies the full gate and captures explicit evidence that the targeted blockers are closed.

## Files to Read
- `.claude/tasks/TASK-247.md`
- `.claude/tasks/TASK-248.md`
- `.claude/tasks/TASK-249.md`
- `.github/workflows/publish.yml`
- `scripts/e2e-smoke.sh`

## Implementation Steps

1. Run the Go, daemon E2E, frontend coverage, and extension coverage checks required by the release gate.
2. Verify evidence for persisted queued-task recovery and admission behavior, publish-gate parity, and provider-promotion durability.
3. Record the verification outcome in the task file and close PLAN-037 if all acceptance criteria pass.

## Acceptance Criteria
- [x] `go vet ./...` exits 0
- [x] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [x] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [x] `./scripts/e2e-smoke.sh` exits 0
- [x] Frontend and VS Code extension coverage suites exit 0 after the Wave 1 changes

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging

## Result
- Verified focused backend validation with `go vet ./...` and targeted package tests from the repo root.
- Verified full backend and concurrency safety with `CGO_ENABLED=1 go test -race ./...`.
- Verified daemon contract behavior with `./scripts/e2e-smoke.sh` and confirmed 25/25 passing after aligning queued-task promotion and cancel semantics.
- Verified `frontend` and `vscode-extension` coverage suites both pass after the Wave 1 changes.