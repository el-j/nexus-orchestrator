---
id: TASK-400
title: Improve markdown discovery heuristics
role: backend
planId: PLAN-056
status: in-progress
dependencies: [TASK-398]
createdAt: 2026-03-28T20:30:00Z
---

## Context

Unknown markdown files are currently classified too generically, which makes AI workflow discovery noisy and unhelpful. This task adds structural heuristics so plan-like documents stand out from ordinary docs.

## Files to Read

- `internal/adapters/outbound/sys_scanner/plan_scanner.go`
- any supporting domain types for plan file kinds

## Implementation Steps

1. Add markdown structural heuristics such as YAML frontmatter, checklist density, and heading patterns.
2. Use those signals to improve summaries and classification without overfitting.
3. Add tests for plan-like and non-plan-like markdown examples.
4. Keep the heuristics deterministic and cheap to run.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] Plan-like markdown is classified more accurately than generic README-style content
- [ ] Heuristic changes are covered by focused scanner tests

## Anti-patterns to Avoid

- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
