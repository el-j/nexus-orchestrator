---
id: TASK-311
title: Split orchestrator.go god object into task_service.go, provider_service.go, session_service.go, execution_engine.go
role: architecture
planId: PLAN-047
status: todo
dependencies: [TASK-301]
createdAt: 2026-03-25T00:00:00.000Z
---

## Context

`internal/core/services/orchestrator.go` is a 1,525-line god object with 53 methods across 5 distinct responsibility domains, all sharing a single `OrchestratorService` struct and `mu sync.Mutex`. This makes the file hard to navigate and the lock contention unmeasurable.

The split keeps the SAME struct (`OrchestratorService`) and SAME package — it only moves method definitions to separate files within `internal/core/services/`. No package boundary changes, no API changes.

## Proposed File Split

| File                  | Methods                                                                                                                                                                                                          |
| --------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `orchestrator.go`     | struct definition, `NewOrchestrator`, `Start`, `Stop`, `emit`, shared helpers                                                                                                                                    |
| `task_service.go`     | SubmitTask, GetTask, GetQueue, GetAllTasks, CancelTask, CreateDraft, GetBacklog, PromoteTask, UpdateTask, ClaimTask, UpdateTaskStatus, HeartbeatTask, validateQueueAdmission, recoverStuckTasks, requeueForRetry |
| `provider_service.go` | RegisterCloudProvider, RemoveProvider, AddProviderConfig, UpdateProviderConfig, RemoveProviderConfig, ListProviderConfigs, GetProviderModels, GetDiscoveredProviders, TriggerScan, PromoteProvider               |
| `session_service.go`  | RegisterAISession, TerminateAISession, HeartbeatAISession, ListAISessions, DeregisterAISession, PurgeDisconnectedSessions, runSessionCleanup, GetDiscoveredAgents, DelegateToNexus, delegationInstruction        |
| `execution_engine.go` | runWorker, runTaskWatchdog, processNext, selectProviderForTask, buildChatContext, executeGeneration, writeTaskOutput, signalWorker, extractCode, estimateTokens                                                  |

## Implementation Steps

1. Read the full `orchestrator.go` to understand which methods go where.
2. Create 4 new files in `internal/core/services/` — all declare `package services`.
3. Move method bodies (cut from orchestrator.go, paste to new file). The struct definition and `NewOrchestrator` stay in `orchestrator.go`.
4. Run `go build ./internal/core/services/...` — fix any compilation errors (usually import statements that need moving to the new files).
5. Run `go vet` and `go test -race` to confirm nothing broke.

## Important Constraints

- Do NOT change any method signatures
- Do NOT change the package name
- Do NOT change the struct fields
- Do NOT add new abstractions — pure file split only
- Keep all imports in the file where they are used (Go allows per-file imports within a package)

## Acceptance Criteria

- [ ] `orchestrator.go` shrinks to ≤200 lines (struct + constructor + Start/Stop + shared helpers)
- [ ] 4 new files exist with appropriate method groupings
- [ ] `go vet ./internal/core/services/...` clean
- [ ] `go test ./internal/core/services/... -race -count=1` all pass (same results as before)
- [ ] `go build ./...` exits 0
