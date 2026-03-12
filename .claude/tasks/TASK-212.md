---
id: TASK-212
title: Fix HTTP API error response format consistency
role: api
planId: PLAN-030
status: todo
dependencies: []
createdAt: 2025-07-25T00:00:00.000Z
---

## Context
Several HTTP API handlers use `http.Error()` with JSON-formatted strings but without setting `Content-Type: application/json`. This sends responses as `text/plain`, breaking client JSON parsing. Affected handlers: `handleGetDiscoveredProviders`, `handleTriggerScan`, `handlePromoteProvider`, `handleDeregisterAISession`. All error responses should use `json.NewEncoder` with proper content type.

## Files to Read
- `internal/adapters/inbound/httpapi/server.go` — lines ~477-506 (discovery/promotion handlers), ~525-551 (AI session handlers)
- `internal/adapters/inbound/httpapi/server_test.go` — existing tests to understand response format expectations

## Implementation Steps
1. Create a small helper: `func writeJSONError(w http.ResponseWriter, msg string, code int)` that sets `Content-Type: application/json`, writes status code, and encodes `{"error": msg}` properly
2. Replace all `http.Error(w, fmt.Sprintf(`{"error":...}`), code)` calls with `writeJSONError(w, msg, code)` in:
   - `handleGetDiscoveredProviders` (~line 479)
   - `handleTriggerScan` (~line 488)
   - `handlePromoteProvider` (~lines 502-505)
   - `handleDeregisterAISession` (~line 551)
3. Audit ALL other handlers for the same pattern — grep for `http.Error.*{.*error` 
4. Add test cases that verify `Content-Type: application/json` header on error responses
5. Verify existing tests still pass with the format change

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] All error responses have `Content-Type: application/json` header
- [ ] All error responses are valid JSON `{"error": "..."}`
- [ ] No `http.Error()` calls with inline JSON strings remain

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER return plain text errors from a JSON API
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
