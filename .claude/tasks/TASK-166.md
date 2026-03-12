---
id: TASK-166
title: HTTP API endpoints for provider discovery
role: api
planId: PLAN-023
status: todo
dependencies: [TASK-164]
createdAt: 2026-03-11T21:00:00.000Z
---

## Context
The GUI and CLI need HTTP endpoints to query discovered providers, trigger a system scan, and promote a discovered provider to active status. These endpoints consume the `Orchestrator` port methods added in TASK-160 and implemented in TASK-164.

## Files to Read
- `internal/adapters/inbound/httpapi/server.go` — existing chi router, provider endpoints
- `internal/core/ports/ports.go` — `Orchestrator` interface with new methods
- `internal/core/domain/provider.go` — `DiscoveredProvider` type

## Implementation Steps

1. In `internal/adapters/inbound/httpapi/server.go`, add 3 new routes under the existing provider section:
   ```go
   r.Get("/api/providers/discovered", s.handleGetDiscoveredProviders)
   r.Post("/api/providers/discovered/scan", s.handleTriggerScan)
   r.Post("/api/providers/promote/{id}", s.handlePromoteProvider)
   ```

2. Implement `handleGetDiscoveredProviders`:
   - Call `s.orch.GetDiscoveredProviders()`
   - Return `200 OK` with `[]DiscoveredProvider` JSON
   - On error, return `500` with error JSON

3. Implement `handleTriggerScan`:
   - Call `s.orch.TriggerScan(r.Context())`
   - Return `200 OK` with `[]DiscoveredProvider` JSON (fresh scan results)
   - On error, return `500` with error JSON

4. Implement `handlePromoteProvider`:
   - Extract `{id}` from URL path via `chi.URLParam(r, "id")`
   - Call `s.orch.PromoteProvider(r.Context(), id)`
   - On success, return `204 No Content`
   - If `domain.ErrNotFound`, return `404`
   - On other error, return `400` with error JSON

5. Extend the SSE hub to emit a `provider_discovered` event when `TriggerScan` completes (fire after successful scan) with payload `{type: "provider_discovered", count: N}`.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `GET /api/providers/discovered` returns `200` with JSON array
- [ ] `POST /api/providers/discovered/scan` triggers scan and returns results
- [ ] `POST /api/providers/promote/{id}` returns `204` on success, `404` on missing
- [ ] SSE event `provider_discovered` fires after scan

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER expose internal error details in API responses — wrap with user-facing messages
- NEVER use goroutines inside `internal/core/services/`
