# TASK-266 — HTTP endpoint DELETE /api/ai-sessions?status=disconnected + Orchestrator port method

**Plan**: PLAN-041  
**Status**: done

## What
1. Add `PurgeDisconnectedSessions(ctx context.Context) (int, error)` to `ports.Orchestrator` interface
2. Implement it in `OrchestratorService` 
3. Add HTTP handler `DELETE /api/ai-sessions?status=disconnected` in httpapi/server.go
4. Wire route in server setup
