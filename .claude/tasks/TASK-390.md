---
id: TASK-390
title: Build token management UX in SettingsView
role: frontend
planId: PLAN-056
status: done
dependencies: [TASK-387, TASK-388, TASK-389]
createdAt: 2026-03-28T20:30:00Z
---

## Context

Even if backend token support exists, users still cannot manage it from the app. This task adds a usable settings flow for creating, rotating, viewing state, and copying access tokens.

## Files to Read

- `frontend/src/views/SettingsView.vue`
- `frontend/src/types/`
- `frontend/src/composables/`

## Implementation Steps

1. Add a token management section for API and MCP access.
2. Surface enabled state, last-updated context, and generate/rotate/copy actions.
3. Wire the view to the runtime config API without leaking raw secrets unnecessarily.
4. Add component or composable tests for the new behaviors.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] SettingsView can generate, rotate, and copy tokens through the runtime config API
- [ ] Raw token handling is limited to explicit user actions

## Anti-patterns to Avoid

- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
