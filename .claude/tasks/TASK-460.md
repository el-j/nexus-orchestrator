---
id: TASK-460
plan: PLAN-061
status: done
wave: 1
priority: 1
---

# TASK-460: Fix httpapi_client — replace http.DefaultClient bypasses

**Problem:** `internal/adapters/outbound/httpapi_client/client.go` has approximately 20 methods that call `http.DefaultClient.Do()`, `http.Get()`, or `http.Post()` directly instead of routing through the injected `r.client` field. This means any custom timeout, TLS config, or transport set by callers is silently ignored. Affected methods include: `AddProviderConfig`, `UpdateProviderConfig`, `RemoveProviderConfig`, `ListProviderConfigs`, `GetDiscoveredProviders`, `TriggerScan`, `PromoteProvider`, `CreateDraft`, `GetBacklog`, `PromoteTask`, `CancelTask`, `UpdateRuntimeConfig`, and all AI session lifecycle methods.

**Fix:** Replace every direct `http.DefaultClient.Do()` / `http.Get()` / `http.Post()` call with the `http.NewRequest` + `r.client.Do(req)` pattern already used by `GetTask` and `SubmitTask`. Convert all `map[string]interface{}` request bodies to proper typed request structs.

**Files:**

- `internal/adapters/outbound/httpapi_client/client.go`
- `internal/adapters/outbound/httpapi_client/client_test.go` (new)

## Checklist

- [x] Audit every method in `client.go` and list each one that calls `http.DefaultClient`, `http.Get`, or `http.Post` directly
- [x] Define typed request struct for each affected endpoint (e.g. `addProviderConfigRequest`, `createDraftRequest`, `registerSessionRequest`) in the same file or a new `types.go`
- [x] Rewrite each affected method using `http.NewRequest(method, url, body)` + `r.client.Do(req)` matching the pattern of the existing `GetTask` method
- [x] Ensure JSON marshalling uses `json.Marshal` + `bytes.NewReader` and sets `Content-Type: application/json`
- [x] Replace `map[string]interface{}` in AI session methods (`RegisterSession`, `HeartbeatSession`, `DeregisterSession`) with typed structs
- [x] Remove any unused imports after the refactor (`net/http` direct call helpers)
- [x] Add `httptest.NewServer`-based unit tests in `client_test.go` covering: `AddProviderConfig`, `RemoveProviderConfig`, `CreateDraft`, `PromoteTask`, `RegisterSession`
- [x] Verify `CGO_ENABLED=1 go test -race ./internal/adapters/outbound/httpapi_client/...` passes

## Acceptance Criteria

- Zero calls to `http.DefaultClient`, `http.Get`, or `http.Post` remain in `client.go`
- All request bodies use typed structs (no `map[string]interface{}`)
- New `client_test.go` with at least 8 table-driven test cases all pass
- `go vet ./internal/adapters/outbound/httpapi_client/...` clean
