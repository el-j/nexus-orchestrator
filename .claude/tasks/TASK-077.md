---
id: TASK-077
title: "Tests: command-aware routing + validation"
role: qa
planId: PLAN-009
status: todo
dependencies: [TASK-075, TASK-076]
createdAt: 2026-03-10T15:00:00.000Z
---

## Context
Comprehensive tests are needed for the new command-aware routing feature: domain type validation, orchestrator service logic (ErrNoPlan enforcement), HTTP API passthrough, and MCP tool parameter handling.

## Files to Read
- `internal/core/domain/task.go`
- `internal/core/services/orchestrator.go`
- `internal/core/services/orchestrator_test.go`
- `internal/core/services/integration_test.go`
- `internal/adapters/inbound/httpapi/server_test.go`
- `internal/adapters/inbound/mcp/server_test.go`
- `internal/adapters/outbound/repo_sqlite/repo.go`
- `internal/adapters/outbound/repo_sqlite/repo_test.go`

## Implementation Steps
1. In `internal/core/services/orchestrator_test.go`, add:
   - `TestSubmitTask_CommandPlan_Succeeds` — submit with CommandPlan, verify accepted
   - `TestSubmitTask_CommandExecute_NoPlan_ReturnsErrNoPlan` — submit with CommandExecute and no prior plan tasks, verify wraps `domain.ErrNoPlan`
   - `TestSubmitTask_CommandExecute_WithPlan_Succeeds` — first submit a CommandPlan task, complete it, then submit CommandExecute, verify accepted
   - `TestSubmitTask_InvalidCommand_ReturnsError` — submit with Command="bogus", verify error
   - `TestSubmitTask_EmptyCommand_DefaultsToAuto` — submit with no Command, verify task.Command is "auto" after retrieval
2. In `internal/adapters/outbound/repo_sqlite/repo_test.go`, add:
   - `TestGetByProjectPath` — save multiple tasks to different projects, verify filtering
   - `TestGetByProjectPath_Empty` — verify returns empty slice for unknown project
3. In `internal/adapters/inbound/mcp/server_test.go` or `mcp/integration_test.go`, add:
   - `TestMCP_SubmitTask_WithCommand` — submit via MCP with command="plan", verify task has CommandPlan
4. Run full test suite: `CGO_ENABLED=1 go test -race -count=1 ./...`

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 with all new tests passing
- [ ] At least 5 new test cases for command-aware routing
- [ ] ErrNoPlan scenario is explicitly tested
- [ ] Default command behavior (empty → auto) is tested

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
