---
id: TASK-386
title: Fix errcheck lint in nexus-mcp-stdio
role: backend
planId: PLAN-056
status: done
dependencies: []
createdAt: 2026-03-28T20:30:00Z
---

## Context

`make nice` is blocked by unchecked `os.Stdout.Write` calls in `cmd/nexus-mcp-stdio/main.go`. This is the smallest release-blocking task in PLAN-056 and unblocks the rest of the plan by restoring a clean lint path first.

## Files to Read

- `cmd/nexus-mcp-stdio/main.go` — current stdout write logic
- `.claude/plans/PLAN-056.md` — plan rationale and acceptance target

## Implementation Steps

1. Replace direct `os.Stdout.Write` calls with checked `fmt.Fprint` or another errcheck-safe helper.
2. Preserve the current response behavior, including appending a trailing newline when missing.
3. Keep diagnostics on stderr and avoid changing the stdio proxy protocol shape.
4. Verify the command still builds and `go vet` remains clean.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] `cmd/nexus-mcp-stdio/main.go` no longer contains unchecked stdout writes
- [ ] Proxy responses still emit exactly one trailing newline

## Anti-patterns to Avoid

- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
