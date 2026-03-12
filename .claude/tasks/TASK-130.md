---
id: TASK-130
title: Provider CRUD HTTP API + Wails bindings
role: api
planId: PLAN-018
status: todo
dependencies: [TASK-129]
createdAt: 2026-03-11T16:10:00.000Z
---

## Context
Expose the persisted provider configs via HTTP API endpoints and Wails bindings so the GUI can add, edit, and remove providers at runtime.

## Files to Read
- `internal/adapters/inbound/httpapi/server.go`
- `internal/core/services/orchestrator.go`
- `internal/core/ports/ports.go`
- `app.go`

## Implementation Steps
1. Add orchestrator methods: `AddProvider(cfg ProviderConfig)`, `UpdateProvider(cfg ProviderConfig)`, `RemoveProvider(id string)`, `ListProviderConfigs()`.
2. Add HTTP endpoints: `POST /api/providers/config`, `PUT /api/providers/config/{id}`, `DELETE /api/providers/config/{id}`, `GET /api/providers/config`.
3. On `AddProvider`: persist to SQLite, instantiate the appropriate LLM adapter, register it with DiscoveryService.
4. On `RemoveProvider`: deregister from DiscoveryService, delete from SQLite.
5. Add corresponding Wails binding methods in `app.go`.
6. Mask API keys in list responses (show only last 4 chars).

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `POST /api/providers/config` with a valid config registers a new provider
- [ ] `DELETE /api/providers/config/{id}` removes it and it disappears from `GET /api/providers`
- [ ] API key is masked in list responses

## Anti-patterns to Avoid
- NEVER import adapters from core services
- NEVER expose full API keys in HTTP responses
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
