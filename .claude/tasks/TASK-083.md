---
id: TASK-083
title: "QA: Full test suite + build verification"
role: qa
planId: PLAN-009
status: todo
dependencies: [TASK-077, TASK-082]
createdAt: 2026-03-10T15:00:00.000Z
---

## Context
After all code changes (command-aware routing) and documentation (GitHub Pages), verify the entire codebase compiles, all tests pass with -race, and the docs structure is correct.

## Files to Read
- `internal/core/services/orchestrator_test.go`
- `internal/adapters/outbound/repo_sqlite/repo_test.go`
- `internal/adapters/inbound/mcp/server_test.go`
- `internal/adapters/inbound/httpapi/server_test.go`

## Implementation Steps
1. Run `go vet ./...` — must exit 0.
2. Run `CGO_ENABLED=1 go build ./...` — must exit 0.
3. Run `CGO_ENABLED=1 go test -race -count=1 ./...` — all tests must pass.
4. Verify docs structure:
   - `docs/_config.yml` exists
   - `docs/index.md` exists with content
   - `docs/architecture.md` exists with content
   - `docs/api-reference.md` exists with content
   - `docs/getting-started.md` exists with content
   - `docs/mcp-integration.md` exists with content
   - `.github/workflows/pages.yml` exists
5. Verify new domain types exist: `CommandType`, `CommandPlan`, `CommandExecute`, `CommandAuto`, `ErrNoPlan`.
6. Report total test count and any failures.

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] All 6 docs pages exist with proper content
- [ ] GitHub Actions workflow exists with valid YAML
- [ ] All new command-aware routing tests pass
- [ ] No regressions in existing tests

## Anti-patterns to Avoid
- NEVER modify any source files in this task — QA is read-only
- NEVER skip the -race flag
