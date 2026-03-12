---
id: TASK-171
title: QA integration tests for discovery and tray pipeline
role: qa
planId: PLAN-023
status: todo
dependencies: [TASK-170]
createdAt: 2026-03-11T21:00:00.000Z
---

## Context
All components of PLAN-023 are wired. We need integration tests to validate the full pipeline: system scanner → OrchestratorService → HTTP API → SSE events, plus unit tests for the scanner's probe logic and the log hub.

## Files to Read
- `internal/core/services/orchestrator_test.go` — existing test patterns
- `internal/adapters/outbound/sys_scanner/scanner.go` — scanner implementation
- `internal/adapters/inbound/httpapi/server.go` — HTTP endpoints
- `internal/adapters/inbound/httpapi/log_hub.go` — log hub
- `internal/adapters/inbound/mcp/server.go` — MCP tools

## Implementation Steps

1. **Scanner unit tests** — `internal/adapters/outbound/sys_scanner/scanner_test.go`:
   - Test `Scan()` returns results without crashing (even if no providers are installed)
   - Test that unreachable port probes return gracefully (no panics, no hanging)
   - Test CLI probe with a known binary that exists on CI (e.g., `go` or `git`)
   - Test deduplication logic (same provider found by port AND CLI)
   - Test timeout: scan completes within 10s even if all endpoints are unreachable

2. **Service unit tests** — extend `internal/core/services/orchestrator_test.go`:
   - Test `GetDiscoveredProviders()` returns empty when no scanner configured
   - Test `TriggerScan()` with a mock `SystemScanner` returning 2 providers
   - Test `PromoteProvider()` with a reachable provider → verify it gets registered
   - Test `PromoteProvider()` with non-existent ID → verify `ErrNotFound`
   - Test `PromoteProvider()` with installed (non-reachable) provider → verify error

3. **HTTP API tests** — `internal/adapters/inbound/httpapi/server_test.go` (or new file):
   - Test `GET /api/providers/discovered` returns `200` with JSON array
   - Test `POST /api/providers/discovered/scan` returns `200` with scan results
   - Test `POST /api/providers/promote/{id}` with valid ID → `204`
   - Test `POST /api/providers/promote/{id}` with bad ID → `404`

4. **LogHub unit tests** — `internal/adapters/inbound/httpapi/log_hub_test.go`:
   - Test `Write()` parses log lines into `LogEntry`
   - Test ring buffer caps at 500 entries
   - Test concurrent writes are thread-safe (`-race`)
   - Test stderr tee: output appears on both the hub and original stderr

5. **MCP tool tests** — extend MCP server tests:
   - Test `discover_providers` tool returns JSON array
   - Test `promote_provider` with valid ID returns success
   - Test `promote_provider` with invalid ID returns error

## Acceptance Criteria
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` passes with all new tests
- [ ] Scanner tests complete within 15s
- [ ] No data races detected under `-race`
- [ ] At least 15 new test cases across all test files
- [ ] Mock `SystemScanner` used in service tests (no real network calls in unit tests)

## Anti-patterns to Avoid
- NEVER use real network calls in unit tests — mock the SystemScanner port
- NEVER skip `-race` flag — concurrency bugs must be caught
- NEVER write tests that depend on specific AI tools being installed on the CI machine
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
