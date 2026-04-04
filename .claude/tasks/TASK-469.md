---
id: TASK-469
plan: PLAN-061
status: done
wave: 5
priority: 5
---

# TASK-469: Add httpapi_client test suite

**Problem:** `internal/adapters/outbound/httpapi_client/` has zero tests. This is the entire transport layer used by the CLI binary to communicate with the daemon. Any regression in request construction, URL building, header handling, or response parsing goes completely undetected until a user reports a broken CLI command.

**Fix:** Implement a comprehensive table-driven test suite using `httptest.NewServer`, mirroring the pattern used in `internal/adapters/inbound/httpapi/server_test.go`. Cover all major method groups: task lifecycle, provider config CRUD, AI session lifecycle, and backlog/draft operations.

**Files:**

- `internal/adapters/outbound/httpapi_client/client_test.go` (new — may already exist after TASK-460; extend it)
- Reference: `internal/adapters/inbound/httpapi/server_test.go`

## Checklist

- [ ] Review `internal/adapters/inbound/httpapi/server_test.go` for the `httptest.NewServer` + table-driven pattern to replicate
- [ ] Bootstrap: write a helper `newTestServer(t, handler http.HandlerFunc) *Client` that starts an `httptest.NewServer`, injects its URL into a `Client`, and registers cleanup with `t.Cleanup`
- [ ] Task lifecycle tests: `SubmitTask` → 201 response; `GetTask` → 200 with body; `CancelTask` → 200; `GetQueue` → list response
- [ ] Provider config tests: `AddProviderConfig` → 201; `UpdateProviderConfig` → 200; `RemoveProviderConfig` → 204; `ListProviderConfigs` → 200 with list
- [ ] AI session tests: `RegisterSession` → 201; `HeartbeatSession` → 200; `DeregisterSession` → 200; `GetAISessions` → 200 with list
- [ ] Backlog/draft tests: `CreateDraft` → 201; `GetBacklog` → 200 with list; `PromoteTask` → 200
- [ ] Error path tests: server returns `404` → client returns recognisable error; server returns `500` → client surfaces the error
- [ ] Run `go test -v ./internal/adapters/outbound/httpapi_client/...` and confirm all cases pass

## Acceptance Criteria

- At least 20 test cases covering all major client method groups
- Both success and error paths tested
- `httptest.NewServer` used throughout (no real network calls)
- `go test -race ./internal/adapters/outbound/httpapi_client/...` passes clean
