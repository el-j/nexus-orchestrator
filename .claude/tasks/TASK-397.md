---
id: TASK-397
title: Wire Playwright into the release CI path
role: devops
planId: PLAN-056
status: done
dependencies: [TASK-393, TASK-394, TASK-395, TASK-396]
createdAt: 2026-03-28T20:30:00Z
---

## Context

Browser coverage only protects releases if it runs in CI. This task wires the new Playwright suite into the project’s automation and keeps artifacts available for debugging failures.

## Files to Read

- `.github/workflows/`
- `Makefile`
- `frontend/package.json`

## Implementation Steps

1. Add a repeatable E2E make target or script entrypoint.
2. Add a CI job for Playwright with browser install and report artifacts.
3. Ensure the job fits the existing release/test workflow ordering.
4. Document any required environment assumptions in the workflow.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] CI runs the Playwright suite headless
- [ ] Failures preserve enough artifacts to debug browser regressions

## Anti-patterns to Avoid

- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
