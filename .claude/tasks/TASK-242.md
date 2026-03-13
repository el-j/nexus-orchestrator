---
id: TASK-242
title: "CI: enforce release-gate E2E and coverage checks"
role: qa
planId: PLAN-035
status: done
dependencies: [TASK-240, TASK-241]
createdAt: 2026-03-13T02:00:00.000Z
completedAt: 2026-03-13T08:31:20.000Z
---

## Context
The repo needs a single release gate that runs daemon E2E, frontend smoke coverage, and extension smoke coverage together instead of relying on manual testing.

## Files to Read
- `.github/workflows/ci.yml`
- `scripts/e2e-smoke.sh`
- `frontend/package.json`
- `vscode-extension/package.json`

## Implementation Steps

1. Add CI jobs or steps for the new frontend and extension smoke coverage suites.
2. Add the deterministic daemon E2E script to CI.
3. Verify the combined gate is stable and documents the intended release checks.

## Acceptance Criteria
- [x] `go vet ./...` exits 0
- [x] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [x] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [x] CI runs the daemon E2E script
- [x] CI runs frontend and extension smoke coverage suites

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging