---
id: TASK-084
title: "Verify: Pages build + complete pipeline check"
role: verify
planId: PLAN-009
status: todo
dependencies: [TASK-083]
createdAt: 2026-03-10T15:00:00.000Z
---

## Context
Final verification that everything works end-to-end: the Go codebase compiles and passes all tests, the documentation site has proper structure, and the command-aware routing feature is fully functional.

## Files to Read
- `.claude/orchestrator.json`
- `docs/_config.yml`
- `docs/index.md`
- `internal/core/domain/task.go`
- `internal/core/services/orchestrator.go`

## Implementation Steps
1. Run full build: `CGO_ENABLED=1 go build ./...`
2. Run full test suite: `CGO_ENABLED=1 go test -race -count=1 ./...`
3. Verify docs directory has all expected files using `find docs/ -name '*.md' | sort`
4. Verify `_config.yml` has correct theme and title
5. Validate the workflow YAML: check `.github/workflows/pages.yml` exists and has correct structure
6. Verify command-aware routing:
   - Check `domain.CommandType` type exists in task.go
   - Check `domain.ErrNoPlan` sentinel exists
   - Check orchestrator handles CommandExecute with plan validation
7. Update `.claude/orchestrator.json`:
   - Mark PLAN-009 as "completed" with `completedAt`
   - Set `activePlanId` to null
   - Update notes

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] All docs pages pass structure validation
- [ ] Pages workflow YAML is valid
- [ ] PLAN-009 marked completed in orchestrator.json

## Anti-patterns to Avoid
- NEVER modify Go source files in verification
- NEVER skip the -race flag
