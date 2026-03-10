# TASK-058 — QA: tests for all PLAN-006 additions

**Plan:** PLAN-006  
**Role:** qa  
**Status:** todo  
**Dependencies:** TASK-054, TASK-055, TASK-057

## Goal

Full test coverage for every new behaviour introduced in PLAN-006, using table-driven tests and the existing mock/stub patterns.

## Test suites

### 1. `internal/core/services/discovery_test.go` — add

- `TestDiscovery_RemoveProvider_RemovesExisting` — registers two, removes one, verify `DetectActive` only finds remaining
- `TestDiscovery_RemoveProvider_UnknownName` — returns `false`
- `TestDiscovery_GetClientByName_Found` — returns correct client
- `TestDiscovery_GetClientByName_NotFound` — returns `false`
- `TestDiscovery_RegisterProvider_ConcurrentSafe` — 20 goroutines call `RegisterProvider` concurrently with `-race`

### 2. `internal/core/services/orchestrator_test.go` — add

- `TestOrchestrator_RegisterCloudProvider_UnknownKind` — factory returns error; `RegisterCloudProvider` propagates it
- `TestOrchestrator_RemoveProvider_NotFound` — wraps `domain.ErrNotFound`
- `TestOrchestrator_GetProviderModels_Found` — mock returning `[]string{"a","b"}` is correctly forwarded
- `TestOrchestrator_GetProviderModels_NotFound` — returns wrapped `domain.ErrNotFound`
- `TestOrchestrator_StatusNoProvider_AfterRemove` — submit task with ModelID "x"; remove provider that has it; task reaches `StatusNoProvider`

### 3. `internal/adapters/inbound/httpapi/server_test.go` — add

Existing test file already stubs out `ports.Orchestrator`. Add:

- `TestHTTP_POST_providers_201` — valid `ProviderConfig` JSON → 201 Created
- `TestHTTP_POST_providers_400_missing_name` — body with no `name` → 400
- `TestHTTP_POST_providers_400_missing_kind` — body with no `kind` → 400
- `TestHTTP_DELETE_providers_204` — known name → 204 No Content
- `TestHTTP_DELETE_providers_404` — unknown name (orch returns ErrNotFound) → 404
- `TestHTTP_GET_provider_models_200` — known name → 200 with JSON array
- `TestHTTP_GET_provider_models_404` — unknown name → 404

## Mock updates

The `mockOrchestrator` in `server_test.go` must be updated to implement the three new `ports.Orchestrator` methods:
```go
RegisterCloudProvider(cfg domain.ProviderConfig) error
RemoveProvider(name string) error
GetProviderModels(name string) ([]string, error)
```

The `mockLLMClient` in `discovery_test.go` and `orchestrator_test.go` already implements `ActiveModel()` — no struct changes needed.

## Acceptance

- `CGO_ENABLED=1 go test -race -count=1 ./internal/...` — **all tests pass**, ≥ 30 test cases total
- `go test -race ./internal/...` with `-v` shows each new test name
