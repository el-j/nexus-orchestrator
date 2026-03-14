---
id: TASK-061
title: "Security: bind to 127.0.0.1 + HTTP request body size limits"
role: api
planId: PLAN-007
status: todo
dependencies: []
createdAt: 2026-03-10T10:00:00.000Z
---

## Context
HTTP API (`:63987`) and MCP server (`:63988`) bind to `0.0.0.0` by default, exposing unauthenticated endpoints to the entire network. Also, no request body size limits exist — `json.NewDecoder(r.Body)` reads unbounded payloads, enabling DoS.

## Files to Read
- `internal/adapters/inbound/httpapi/server.go`
- `internal/adapters/inbound/mcp/server.go`
- `main.go`
- `cmd/nexus-daemon/main.go`

## Implementation Steps
1. Change default listen addresses from `:63987` to `127.0.0.1:63987` and `:63988` to `127.0.0.1:63988` in both `main.go` and `cmd/nexus-daemon/main.go` (env-var fallback unchanged — user can still override).
2. In `httpapi/server.go`, add a middleware (or per-handler wrapper) that wraps `r.Body` with `http.MaxBytesReader(w, r.Body, 1<<20)` (1 MB limit) for all non-SSE POST endpoints.
3. In `mcp/server.go`, apply the same 1 MB body limit on the `/mcp` POST handler.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] Default daemon listens on `127.0.0.1:63987`, not `0.0.0.0:63987`
- [ ] Default MCP listens on `127.0.0.1:63988`, not `0.0.0.0:63988`
- [ ] POST body exceeding 1 MB returns 413 or 400 error

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
