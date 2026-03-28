---
id: TASK-392
title: Set up Playwright E2E infrastructure
role: testing
planId: PLAN-056
status: done
dependencies: [TASK-390, TASK-391]
createdAt: 2026-03-28T20:30:00Z
---

## Context

The frontend has unit and component coverage but no browser-level end-to-end safety net. This task establishes Playwright, fixtures, scripts, and a stable test entry point for full workflows.

## Files to Read

- `frontend/package.json`
- `frontend/`
- `Makefile`
- existing frontend test configuration files

## Implementation Steps

1. Add Playwright dependencies and a baseline config.
2. Create a deterministic test bootstrap for the frontend plus backend dependencies.
3. Add scripts and make targets for local and CI execution.
4. Prove the harness works with at least one smoke scenario.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] `frontend` has a working Playwright config and runnable test command
- [ ] The baseline browser harness runs headless and deterministic locally

## Anti-patterns to Avoid

- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
