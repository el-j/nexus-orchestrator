---
id: TASK-388
title: Add bearer token middleware to the HTTP API server
role: api
planId: PLAN-056
status: in-progress
dependencies: [TASK-386]
createdAt: 2026-03-28T20:30:00Z
---

## Context

The HTTP API exposes task, provider, and session operations without real authentication. This task adds an opt-in bearer token gate so remote or forwarded access is not effectively public.

## Files to Read

- `internal/adapters/inbound/httpapi/server.go`
- `internal/adapters/inbound/httpapi/server_test.go`

## Implementation Steps

1. Add a token-auth middleware keyed off `NEXUS_API_TOKEN`.
2. Enforce the bearer token for API routes while preserving local UI behavior when the token is unset.
3. Keep health and discovery behavior intentional and covered by tests.
4. Add request tests for allowed and denied calls.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] HTTP API requests are rejected with `401` when `NEXUS_API_TOKEN` is set and no valid bearer token is supplied
- [ ] Token-disabled local usage still works unchanged

## Anti-patterns to Avoid

- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
