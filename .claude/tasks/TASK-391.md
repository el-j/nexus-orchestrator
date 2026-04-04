---
id: TASK-391
title: Read live queue cap and config in SettingsView
role: frontend
planId: PLAN-056
status: done
dependencies: [TASK-389]
createdAt: 2026-03-28T20:30:00Z
---

## Context

The queue cap shown in Settings is hardcoded to `50`, which misleads users running with a different runtime value. This task switches the screen to live config data from the server.

## Files to Read

- `frontend/src/views/SettingsView.vue`
- `frontend/src/composables/`
- `internal/adapters/inbound/httpapi/server.go`

## Implementation Steps

1. Fetch runtime config from `/api/config`.
2. Replace hardcoded queue-cap and related settings labels with live values.
3. Handle loading, empty, and error states gracefully in the settings UI.
4. Add frontend tests around config loading and display.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] SettingsView displays the live queue cap returned by `/api/config`
- [ ] No hardcoded queue-cap text remains in the settings flow

## Anti-patterns to Avoid

- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
