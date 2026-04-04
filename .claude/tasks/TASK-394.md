---
id: TASK-394
title: Add Playwright settings and token flow coverage
role: testing
planId: PLAN-056
status: done
dependencies: [TASK-392, TASK-390]
createdAt: 2026-03-28T20:30:00Z
---

## Context

Token UX is security-sensitive and easy to regress visually. This task adds browser coverage for the settings flow that manages tokens and related runtime configuration.

## Files to Read

- `frontend/e2e/`
- `frontend/src/views/SettingsView.vue`

## Implementation Steps

1. Add a browser test for navigating to Settings and managing tokens.
2. Cover generate/rotate/copy or equivalent token actions.
3. Assert live runtime config values are rendered.
4. Keep token assertions safe and deterministic.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] Settings token flow is covered in Playwright
- [ ] The test verifies live config-backed UI state

## Anti-patterns to Avoid

- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
