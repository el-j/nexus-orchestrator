---
id: TASK-551
planId: PLAN-070
title: 'httpapi provider handler tests + activity handler tests'
role: qa
status: todo
createdAt: 2026-04-14T02:00:00Z
---

# TASK-551 — httpapi provider + activity handler tests

## Context

Two large zero-coverage gaps in `internal/adapters/inbound/httpapi/`:

### `handlers_providers.go` — 8 of 12 functions at 0%

`handleProviders`, `handleRegisterProvider`, `handleRemoveProvider`, `handleProviderModels`,
`handleAddProviderConfig`, `handleListProviderConfigs`, `handleUpdateProviderConfig`,
`handleRemoveProviderConfig` — all at 0%. No provider handler test file exists.

### `handlers_activities.go` — 0%

`handleListActivities` and `handleActivityTimeline` both at 0%.

These represent a significant untested surface. Provider management is core user-facing
functionality; a serialisation bug or wrong status code here would go undetected.

## Work Required

### `internal/adapters/inbound/httpapi/providers_handlers_test.go` (new)

Use the same `mockOrchestrator` pattern from `server_test.go`. Add methods to the mock
if missing: `GetProviders`, `RegisterProvider`, `RemoveProvider`, `GetProviderModels`,
`AddProviderConfig`, `ListProviderConfigs`, `UpdateProviderConfig`, `RemoveProviderConfig`.

Test at minimum the happy path for each of the 8 handlers:

1. `TestHandleGetProviders_OK` — 200 + JSON array
2. `TestHandleRegisterProvider_OK` — POST, 201
3. `TestHandleRemoveProvider_OK` — DELETE, 204
4. `TestHandleGetProviderModels_OK` — GET, 200 + models array
5. `TestHandleAddProviderConfig_OK` — POST, 201
6. `TestHandleListProviderConfigs_OK` — GET, 200 + array
7. `TestHandleUpdateProviderConfig_OK` — PUT, 200
8. `TestHandleRemoveProviderConfig_OK` — DELETE, 204

### `internal/adapters/inbound/httpapi/activities_handlers_test.go` (new)

Add mock for `GetActivities` / `GetActivityTimeline` on the orchestrator mock.

Test:

1. `TestHandleListActivities_OK` — GET `?projectPath=x`, 200 + array
2. `TestHandleActivityTimeline_OK` — GET, 200 + timeline response
3. `TestHandleListActivities_MissingProject` — GET without projectPath, 400

## File Targets

- `internal/adapters/inbound/httpapi/providers_handlers_test.go` (new)
- `internal/adapters/inbound/httpapi/activities_handlers_test.go` (new)

## Acceptance Criteria

- `CGO_ENABLED=1 go test -race -count=1 ./internal/adapters/inbound/httpapi/...` green
- At least 8 provider handler tests + 3 activity handler tests
- `handlers_providers.go` coverage ≥ 50%
- `handlers_activities.go` coverage ≥ 60%
