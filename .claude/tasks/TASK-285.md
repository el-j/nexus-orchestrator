# TASK-285 — VS Code: `NexusClient` additions

**Plan:** PLAN-044  
**Status:** TODO  
**Layer:** TypeScript · VS Code extension  
**Depends on:** TASK-281 (HTTP endpoints must exist)  
**File:** `vscode-extension/src/nexusClient.ts`  

## Objective

Extend `NexusClient` and its type exports with the PLAN-044 API surface. All additions are backward-compatible.

## New types to export

```typescript
export interface DiscoveredAgent {
  id: string;
  kind: string;
  name: string;
  detectionMethod: string;
  processName?: string;
  cliPath?: string;
  configPath?: string;
  mcpEndpoint?: string;
  isRunning: boolean;
  lastSeen: string;
}

export interface DelegateResponse {
  instruction: string;
  sessionId: string;
}
```

## Extend `AISession` interface

Add optional fields:
```typescript
delegatedToNexus?: boolean;
delegationTimestamp?: string;
agentCapabilities?: string[];
detectionMethod?: string;
```

## New methods on `NexusClient`

```typescript
/** Return all registered AI sessions. */
async listAISessions(): Promise<AISession[]> {
  return this.get<AISession[]>('/api/ai-sessions');
}

/** Return agents detected by the daemon's scanner (may trigger an on-demand scan). */
async getDiscoveredAgents(): Promise<DiscoveredAgent[]> {
  return this.get<DiscoveredAgent[]>('/api/ai-sessions/discovered');
}

/** Mark the session as delegated and return the delegation instruction. */
async delegateSession(sessionId: string): Promise<DelegateResponse> {
  return this.post<DelegateResponse>(
    `/api/ai-sessions/${encodeURIComponent(sessionId)}/delegate`,
    {}
  );
}
```

## Acceptance Criteria

- TypeScript compiles without errors: `npx tsc --noEmit` in `vscode-extension/`
- `listAISessions()` is used in `AISessionsTreeProvider` (TASK-283)
- `delegateSession()` is used in `delegateToNexus` command (TASK-284)
- `getDiscoveredAgents()` is available for future use in TASK-286 frontend (via daemon, not extension)
