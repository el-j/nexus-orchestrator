---
id: TASK-401
title: Group and filter discovered plans in the frontend
role: frontend
planId: PLAN-056
status: todo
dependencies: [TASK-398, TASK-399, TASK-400]
createdAt: 2026-03-28T20:30:00Z
---

## Context

The current Plans view is still mostly a flat dump, which becomes noisy as discovery broadens. This task adds grouping, filtering, and tool-level cues so the expanded scan data remains understandable.

## Files to Read

- `frontend/src/views/DiscoveredPlansView.vue`
- `frontend/src/composables/useDiscoveredPlans.ts`
- related type definitions

## Implementation Steps

1. Group discovered files by AI tool or workflow family.
2. Add filter controls and badges for file kind, tool, and active state.
3. Preserve the existing project grouping while making the view easier to scan.
4. Add frontend tests around grouping and filter behavior.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0 (or new tests added for this task)
- [ ] DiscoveredPlansView groups files by tool/workflow and supports filtering
- [ ] New UI cues make active and important plan files easier to identify

## Anti-patterns to Avoid

- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/` — goroutine lifecycle belongs in inbound adapters
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use `console.log` — use `log.Printf` for operational logging
