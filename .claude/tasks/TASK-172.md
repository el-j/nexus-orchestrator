---
id: TASK-172
title: Domain — add DRAFT/BACKLOG statuses, Priority, ProviderName, Tags fields
role: architecture
planId: PLAN-024
status: todo
dependencies: []
createdAt: 2026-03-11T22:00:00.000Z
---

## Context
nexusOrchestrator's Task entity only has execution-oriented statuses (QUEUED→PROCESSING→terminal). Users need to capture ideas that sit outside the queue until explicitly promoted. Additionally, per-task explicit provider routing requires a `ProviderName` field (exact match, not substring hint), and idea organisation needs `Priority` + `Tags`.

## Files to Read
- `internal/core/domain/task.go` — Task struct, TaskStatus, CommandType

## Implementation Steps

1. In `internal/core/domain/task.go`, add two new `TaskStatus` constants:
   ```go
   StatusDraft   TaskStatus = "DRAFT"
   StatusBacklog TaskStatus = "BACKLOG"
   ```
   These are non-executing states. The worker loop will never dequeue them.

2. Add three new fields to the `Task` struct:
   ```go
   // ProviderName is an explicit provider lock. Non-empty means skip discovery
   // and route directly to the named provider. Empty falls back to ProviderHint/ModelID.
   ProviderName string   `json:"providerName,omitempty"`
   // Priority controls backlog ordering (1=highest). Defaults to 2 (medium).
   Priority     int      `json:"priority,omitempty"`
   // Tags are free-form labels for organising ideas and backlog items.
   Tags         []string `json:"tags,omitempty"`
   ```

3. Add a convenience method on Task:
   ```go
   // IsExecutable returns true if the task is in a state that should enter the queue.
   func (t Task) IsExecutable() bool {
       return t.Status == StatusQueued
   }
   ```

4. Ensure `StatusDraft` and `StatusBacklog` are NOT considered terminal statuses (tasks can transition from them to QUEUED via promotion).

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `StatusDraft` and `StatusBacklog` are valid `TaskStatus` constants
- [ ] `Task` has `ProviderName`, `Priority`, `Tags` fields with correct JSON tags
- [ ] Existing Task statuses and fields are unchanged (additive only)

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER remove or rename existing TaskStatus values — additive only
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
