---
id: TASK-066
title: "HTTP API: fix cancel error codes + sanitise error responses + CSP headers"
role: api
planId: PLAN-007
status: todo
dependencies: [TASK-061]
createdAt: 2026-03-10T10:00:00.000Z
---

## Context
`handleCancelTask()` returns 404 for ALL errors, not just NotFound. Error messages leak internal details (raw sqlite errors). Dashboard lacks CSP and X-Frame-Options headers. Error handling in cancel is inconsistent.

## Files to Read
- `internal/adapters/inbound/httpapi/server.go`
- `internal/adapters/inbound/httpapi/dashboard.go`
- `internal/adapters/inbound/httpapi/server_test.go`

## Implementation Steps
1. In `handleCancelTask()`, check `errors.Is(err, domain.ErrNotFound)` — return 404 only for NotFound, return 500 for other errors.
2. In all error responses, replace `err.Error()` with generic safe messages for 500 errors. Only return specific error text for 4xx (validation) errors. Log the real error with `log.Printf`.
3. Add a middleware function that sets security headers on all responses: `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`. On the `/ui` dashboard handler specifically, add `Content-Security-Policy: default-src 'self'; style-src 'unsafe-inline'; script-src 'unsafe-inline'`.
4. Update `handleRegisterProvider` to return 409 Conflict (not ErrNotFound check which is wrong) when a duplicate provider is registered.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] Cancel of unknown task returns 404; cancel error from repo returns 500
- [ ] Error responses do not contain "sqlite:" prefixed messages
- [ ] Dashboard response has CSP and X-Frame-Options headers

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
