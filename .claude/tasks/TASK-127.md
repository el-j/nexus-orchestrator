---
id: TASK-127
title: Show all providers including unreachable
role: backend
planId: PLAN-018
status: todo
dependencies: []
createdAt: 2026-03-11T16:10:00.000Z
---

## Context
`DiscoveryService.ListProviders()` currently only populates model lists for alive providers. Unreachable providers are included with `Active: false` but users can't tell *which* providers are configured vs missing. We need to always return all registered providers with clear status (active/unreachable/error) and the configured base URL.

## Files to Read
- `internal/core/services/discovery.go`
- `internal/core/ports/ports.go` (ProviderInfo type)
- `frontend/src/components/ProviderStatus.vue`

## Implementation Steps
1. Add a `BaseURL` and `Error` field to `ports.ProviderInfo` (or the relevant struct).
2. In `ListProviders()`, always include every registered client — set `Error` to the ping failure reason when unreachable.
3. When `Active: false`, still show the provider name and configured URL so the user can diagnose connectivity.
4. Update the HTTP API response to include the new fields.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `GET /api/providers` returns all registered providers even when `Ping()` fails
- [ ] Each provider entry includes its configured base URL

## Anti-patterns to Avoid
- NEVER import adapters from core services
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
