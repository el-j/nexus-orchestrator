---
id: TASK-398
title: Extend discovery with Windsurf, Aider, Continue, Copilot, and generic task files
role: backend
planId: PLAN-056
status: in-progress
dependencies: [TASK-386]
createdAt: 2026-03-28T20:30:00Z
---

## Context

The current plan scanner misses several real-world AI workflow artifacts, including Windsurf, Aider, Continue config, Copilot’s canonical `.github` location, and generic task files. This task broadens the discovery catalog so the tool can surface actual workflow assets users already have.

## Files to Read

- `internal/adapters/outbound/sys_scanner/plan_scanner.go`
- `internal/core/domain/discovered_agent.go`
- existing plan discovery tests

## Implementation Steps

1. Add new recognized file kinds and well-known paths for the missing tools.
2. Extend classification to catch generic task config files that should appear in scans.
3. Update tests to cover the newly recognized artifacts.
4. Keep backward compatibility for existing discovery results.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] `.windsurfrules`, `.aider.conf.yml`, `.continue/config.*`, `.github/copilot-instructions.md`, and generic task files are recognized by the scanner
- [ ] Existing plan discovery behavior remains intact for prior file kinds

## Anti-patterns to Avoid

- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
