---
id: TASK-074
title: "Domain: Add CommandType (plan/execute/auto) to Task"
role: architecture
planId: PLAN-009
status: todo
dependencies: []
createdAt: 2026-03-10T15:00:00.000Z
---

## Context
The orchestrator needs to support command-aware task routing. Tasks should carry a `CommandType` that indicates whether the task is for planning (creating plans, task documents, orchestration) or execution (implementing code changes). When no command is specified, it defaults to "auto" which lets the orchestrator decide.

## Files to Read
- `internal/core/domain/task.go`
- `internal/core/domain/session.go`
- `internal/core/domain/provider.go`
- `internal/core/ports/ports.go`

## Implementation Steps
1. In `internal/core/domain/task.go`, add a new `CommandType` string type with constants:
   - `CommandPlan CommandType = "plan"` — task is for planning/orchestration work
   - `CommandExecute CommandType = "execute"` — task is for code implementation
   - `CommandAuto CommandType = "auto"` — orchestrator decides (default)
2. Add a `String()` method on `CommandType` (same pattern as `TaskStatus`).
3. Add a `IsValid()` method on `CommandType` that returns true for the three valid values and empty string (treated as auto).
4. Add a `Command CommandType` field to the `Task` struct with JSON tag `json:"command,omitempty"`.
5. Add `ErrNoPlan` sentinel error: `var ErrNoPlan = errors.New("no plan exists; planning required before execution")`.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `CommandType` type with `CommandPlan`, `CommandExecute`, `CommandAuto` constants exists
- [ ] `Task.Command` field exists with proper JSON tag
- [ ] `ErrNoPlan` sentinel error exists
- [ ] `IsValid()` method accepts "", "plan", "execute", "auto" as valid

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
