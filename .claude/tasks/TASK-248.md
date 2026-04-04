---
id: TASK-248
title: "DevOps: make publish workflow enforce full release gate"
role: devops
planId: PLAN-037
status: done
dependencies: []
createdAt: 2026-03-13T10:17:51.000Z
---

## Context
The publish workflow currently produces artifacts after a materially weaker set of checks than PR CI. This task makes artifact creation depend on the same release-critical validation surfaces that were added to CI.

## Files to Read
- `.github/workflows/ci.yml`
- `.github/workflows/publish.yml`
- `frontend/package.json`
- `vscode-extension/package.json`
- `scripts/e2e-smoke.sh`

## Implementation Steps

1. Add a publish-side full-gate job that mirrors CI's release-critical checks.
2. Make artifact-producing jobs depend on that gate before any build or packaging work starts.
3. Keep the workflow structure maintainable and verify that publish is no longer weaker than CI for release-critical checks.

## Acceptance Criteria
- [x] `go vet ./...` exits 0
- [x] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [x] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [x] `publish.yml` runs vet, gofmt, lint, Go race tests, frontend coverage, extension coverage, and daemon E2E before artifact jobs
- [x] Publish artifact jobs are blocked when the full gate fails

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging

## Result
- Expanded `.github/workflows/publish.yml` with vet, lint, frontend coverage, VS Code extension coverage, daemon E2E, and a `release-gate` barrier before artifact jobs.
- Corrected workflow dependencies so release-gate conditionals read valid `needs.version.outputs.exists` state and artifact builds cannot bypass the gate.