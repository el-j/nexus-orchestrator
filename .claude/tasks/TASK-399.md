---
id: TASK-399
title: Add recursive discovery for instruction and prompt files
role: backend
planId: PLAN-056
status: in-progress
dependencies: [TASK-398]
createdAt: 2026-03-28T20:30:00Z
---

## Context

Instruction and prompt files often live below the repo root, especially in `.github/` and tool-specific subfolders. This task expands scanning depth so those files are surfaced instead of silently missed.

## Files to Read

- `internal/adapters/outbound/sys_scanner/plan_scanner.go`
- existing discovery tests

## Implementation Steps

1. Add a bounded recursive walk for instruction and prompt file patterns.
2. Include `.github/` and other relevant subdirectories without exploding scan cost.
3. Deduplicate results across overlapping scan roots.
4. Add tests for nested discovery paths.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] Nested `*.instructions.md` and `*.prompt.md` files are discovered within the intended scan depth
- [ ] Duplicate entries are not produced when the same file is reachable from multiple roots

## Anti-patterns to Avoid

- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
