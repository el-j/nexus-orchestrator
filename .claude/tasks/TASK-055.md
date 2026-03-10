# TASK-055 — HTTP API: provider management endpoints

**Plan:** PLAN-006  
**Role:** api  
**Status:** todo  
**Dependencies:** TASK-053

## Goal

Expose the three new `ports.Orchestrator` methods as REST endpoints and enrich the existing `GET /api/providers` response.

## Changes in `internal/adapters/inbound/httpapi/server.go`

### Router additions

```
POST   /api/providers              → handleRegisterProvider
DELETE /api/providers/{name}       → handleRemoveProvider
GET    /api/providers/{name}/models → handleProviderModels
```

### `handleRegisterProvider`

```
POST /api/providers
Content-Type: application/json
Body: { "name": "...", "kind": "...", "baseUrl": "...", "apiKey": "...", "model": "..." }
```

- Decode `domain.ProviderConfig` from body; validation: `name` and `kind` must be non-empty
- Call `s.orch.RegisterCloudProvider(cfg)`
- `201 Created` with `{"name": cfg.Name}` on success
- `400` for decode/validation error
- `409 Conflict` if provider name already registered (detect by error string)
- `422 Unprocessable Entity` for unknown kind

### `handleRemoveProvider`

```
DELETE /api/providers/{name}
```

- URL-decode `{name}` (chi param)
- Call `s.orch.RemoveProvider(name)`
- `204 No Content` on success
- `404 Not Found` if `errors.Is(err, domain.ErrNotFound)`

### `handleProviderModels`

```
GET /api/providers/{name}/models
```

- Call `s.orch.GetProviderModels(name)`
- `200 OK` with JSON array of model strings
- `404 Not Found` if `errors.Is(err, domain.ErrNotFound)`

### Existing `handleProviders` — no change required

The `GET /api/providers` already returns `[]ports.ProviderInfo` which now includes `ActiveModel` and `Models` fields via `ListProviders()`.

## Acceptance

- `go vet ./internal/adapters/inbound/httpapi/...` passes
- `CGO_ENABLED=1 go test -race ./internal/adapters/inbound/httpapi/...` >= prior pass count
