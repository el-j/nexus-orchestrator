---
id: TASK-499
plan: PLAN-064
status: done
wave: 1
priority: 1
---

# TASK-499: Add httpapi_client test suite (Go)

## Description

`internal/adapters/outbound/httpapi_client/` is the entire network transport layer for the CLI binary. Every `nexus-cli` command eventually calls through this package. It has zero tests. Silent breakage here would silently fail all CLI users without any test failure catching it during development.

## Checklist

- [ ] Create `internal/adapters/outbound/httpapi_client/client_test.go`
- [ ] Use `httptest.NewServer` with a `http.ServeMux` to simulate nexus HTTP API responses for each endpoint
- [ ] Table-driven tests for `SubmitTask`: 201 Created returns populated task; 400 returns error with message; 500 returns error
- [ ] Table-driven tests for `GetTask`: 200 returns task; 404 returns typed "not found" error; malformed JSON body returns parse error
- [ ] Tests for `CancelTask`: 204 returns nil; 404 returns error
- [ ] Tests for provider config CRUD: `CreateProviderConfig`, `ListProviderConfigs`, `UpdateProviderConfig`, `DeleteProviderConfig` each with 2xx success and error paths
- [ ] Tests for AI session lifecycle: `RegisterSession` (201), `HeartbeatSession` (200), `DeregisterSession` (204)
- [ ] Test that the injected HTTP client is used (not `http.DefaultClient`): configure test server to close connection immediately; use a custom transport that records calls; verify requests reach the test server, not a real host
- [ ] Test `UpdateRuntimeConfig` if the method exists

## Files

- `internal/adapters/outbound/httpapi_client/client_test.go` (create)
- `internal/adapters/outbound/httpapi_client/client.go` (reference)

## Acceptance Criteria

- Minimum 15 table-driven test cases across all methods
- All requests verified against `httptest.Server` - no real network traffic
- Injected client transport verifiably used (transport-level interception test)
- `go test ./internal/adapters/outbound/httpapi_client/...` exits 0 (no CGO required)
