---
id: TASK-068
title: MCP protocol integration test (all 6 tools via JSON-RPC in-process)
role: qa
planId: PLAN-008
status: todo
dependencies: []
createdAt: 2026-03-10T14:00:00.000Z
---

## Context
The existing MCP tests in `internal/adapters/inbound/mcp/server_test.go` use a mock orchestrator and only test surface-level request/response shapes. We need an **integration test** that wires the real `OrchestratorService` (with real SQLite, real fs_writer, mock LLM) through the MCP JSON-RPC 2.0 endpoint. This proves the MCP protocol actually works end-to-end — task submission through MCP completes and the file is written.

## Files to Read
- `internal/adapters/inbound/mcp/server.go`
- `internal/adapters/inbound/mcp/server_test.go`
- `internal/core/services/integration_test.go` (for testStack pattern)
- `internal/core/ports/ports.go`
- `internal/core/domain/task.go`

## Implementation Steps

1. Create `internal/adapters/inbound/mcp/integration_test.go` in package `mcp_test`.
2. Build a `mcpTestStack` similar to `testStack` in `services_test`: temp SQLite DB, mockLLM, real OrchestratorService, wired into `mcp.NewServer`, wrapped in `httptest.NewServer`.
3. Write `TestMCPIntegration_SubmitAndComplete`:
   - Call `tools/call` with `submit_task` (projectPath=tmpDir, targetFile="out.go", instruction="generate foo").
   - Extract the returned task ID from the JSON-RPC result.
   - Poll with `tools/call` → `get_task` until status is `COMPLETED` (timeout 15s).
   - Verify `get_queue` returns an empty or completed queue.
   - Verify the output file was written to disk.
4. Write `TestMCPIntegration_CancelTask`:
   - Submit a task with a slow mockLLM (250ms sleep).
   - Immediately call `cancel_task`.
   - Verify `get_task` shows `CANCELLED` status.
5. Write `TestMCPIntegration_GetProviders`:
   - Call `get_providers`.
   - Verify the mock provider appears with `active: true`.
6. Write `TestMCPIntegration_HealthTool`:
   - Call `tools/call` → `health`.
   - Verify result contains `"ok"`.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] At least 4 new integration tests in `internal/adapters/inbound/mcp/integration_test.go`
- [ ] `TestMCPIntegration_SubmitAndComplete` verifies file output on disk
- [ ] All tests use real SQLite + real OrchestratorService (not mock orchestrator)

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
