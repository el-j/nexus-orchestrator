# TASK-280 — OrchestratorService: `DelegateToNexus` + `GetDiscoveredAgents`

**Plan:** PLAN-044  
**Status:** TODO  
**Layer:** Go · services (`internal/core/services/`)  
**Depends on:** TASK-277, TASK-279  

## Objective

Add `DelegateToNexus` and `GetDiscoveredAgents` to `OrchestratorService`, satisfying the new `Orchestrator` port methods.

## Changes

### `internal/core/services/orchestrator.go`

**New private interface (local, avoids importing repo_sqlite):**
```go
type discoveredAgentStore interface {
    UpsertDiscoveredAgent(ctx context.Context, a domain.DiscoveredAgent) error
    ListDiscoveredAgents(ctx context.Context) ([]domain.DiscoveredAgent, error)
}
```

**New fields on `OrchestratorService`:**
```go
agentScanner       ports.AgentScanner
agentRepo          discoveredAgentStore
lastAgentScan      time.Time
lastAgentScanMu    sync.Mutex
```

**New setters (consistent with `SetAISessionRepo` pattern):**
```go
func (o *OrchestratorService) SetAgentScanner(s ports.AgentScanner)
func (o *OrchestratorService) SetDiscoveredAgentRepo(r discoveredAgentStore)
```

**`GetDiscoveredAgents(ctx context.Context) ([]domain.DiscoveredAgent, error)`:**
```
1. Lock lastAgentScanMu.
2. If agentScanner != nil AND (lastAgentScan.IsZero() OR time.Since(lastAgentScan) > 30s):
   a. Unlock mutex (scan can be slow).
   b. agents, err = agentScanner.ScanAgents(ctx) — log error but don't fail.
   c. For each agent: agentRepo.UpsertDiscoveredAgent(ctx, agent) — ignore upsert errors (best-effort).
   d. Lock mutex; lastAgentScan = time.Now(); unlock.
3. If agentRepo != nil: return agentRepo.ListDiscoveredAgents(ctx).
4. Else: return agents (from step 2b), nil.
```

**`DelegateToNexus(ctx context.Context, sessionID string) (string, error)`:**
```
1. aiSessionRepo.GetAISessionByID(ctx, sessionID) → ErrNotFound passthrough.
2. Compose instruction string using delegationInstructionTemplate (unexported const/func).
3. session.DelegatedToNexus = true
4. session.DelegationTimestamp = &now
5. aiSessionRepo.SaveAISession(ctx, session)
6. return instruction, nil
```

**Delegation instruction template** (unexported `delegationInstruction(session domain.AISession) string`):
```
You are now operating under nexusOrchestrator coordination.

YOUR NEW WORKFLOW:
1. Check the nexus queue:  GET http://127.0.0.1:63987/api/tasks
2. Claim the next QUEUED task:
   POST http://127.0.0.1:63987/api/tasks/{id}/claim
   Body: {"sessionId": "<SESSION_ID>"}
3. Execute the task's instruction for <PROJECT_PATH>.
4. Report completion:
   PUT http://127.0.0.1:63987/api/tasks/{id}/status
   Body: {"sessionId": "<SESSION_ID>", "status": "COMPLETED", "logs": "<summary>"}
5. Repeat from step 1.  If queue is empty, wait 30 s and check again.

Do NOT take autonomous actions outside the nexus task queue while this delegation is active.

Nexus session ID: <SESSION_ID>
Project:          <PROJECT_PATH>
Timestamp:        <ISO8601>
```

## Acceptance Criteria

- `var _ ports.Orchestrator = (*OrchestratorService)(nil)` still compiles
- `CGO_ENABLED=1 go test -race ./internal/core/services/...` passes
- Tests in TASK-289 cover both methods
