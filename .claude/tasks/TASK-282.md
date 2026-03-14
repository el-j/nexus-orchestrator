# TASK-282 — VS Code: `AgentDetector` class

**Plan:** PLAN-044  
**Status:** TODO  
**Layer:** TypeScript · VS Code extension  
**Depends on:** TASK-285 (NexusClient additions must be present)  
**New file:** `vscode-extension/src/agentDetector.ts`  

## Objective

Implement the 30-second polling loop that discovers all AI agents visible to the VS Code extension host and registers/heartbeats/deregisters them as `AISession` records with the daemon.

## Interface

```typescript
export interface DetectedAgent {
  agentName: string;
  source: 'vscode-discovered';
  externalId: string;    // "discover:<machineId>:<agentKind>:<workspacePath>"
  projectPath?: string;
  capabilities: string[];
  detectionMethod: string;
}
```

## `AgentDetector` class

```typescript
export class AgentDetector implements vscode.Disposable {
  readonly onDidChange: vscode.Event<void>;
  constructor(client: NexusClient, context: vscode.ExtensionContext)
  start(): void
  stop(): void
  dispose(): void
  async detectAll(): Promise<DetectedAgent[]>   // exposed for tests
}
```

### Detection Strategies (all run in `Promise.allSettled`)

**S1 — `detectVSCodeExtensions()`**
```typescript
const knownAIExtensions: Record<string, { name: string; capabilities: string[] }> = {
  'saoudrizwan.claude-dev':         { name: 'Cline',               capabilities: ['file-write','code-execute','terminal'] },
  'continue.continue':              { name: 'Continue',            capabilities: ['file-write','code-execute','chat'] },
  'codeium.codeium':                { name: 'Codeium',             capabilities: ['chat'] },
  'codegpt.codegpt-4':              { name: 'CodeGPT',             capabilities: ['chat'] },
  'anysphere.cursor-always-local':  { name: 'Cursor AI',           capabilities: ['file-write','code-execute'] },
  'github.copilot':                 { name: 'GitHub Copilot',      capabilities: ['chat'] },
  'github.copilot-chat':            { name: 'GitHub Copilot Chat', capabilities: ['chat'] },
};
```
Yield one `DetectedAgent` per installed extension. `externalId = "discover:<machineId>:ext:<extId>"`.

**S2 — `detectFromFilesystem()`**
Read with `import * as fs from 'fs/promises'` and `os.homedir()`. All reads wrapped in try/catch; silently skip on ENOENT.
- `~/.claude/settings.json` → if parses as `{ apiKey: string }` → `{ agentName: "Claude CLI", capabilities: ["code-execute","terminal"], detectionMethod: "fs-config" }`
- macOS: `~/Library/Application Support/Claude/` stat → `{ agentName: "Claude Desktop", capabilities: ["chat","mcp-client"], detectionMethod: "fs-config" }`
- Linux: `~/.config/claude/` stat → same
- Read `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `~/.config/claude/claude_desktop_config.json` (Linux); parse `mcpServers` object keys → for each key emit `{ agentName: "MCP: <serverName>", capabilities: ["mcp-client"], detectionMethod: "fs-mcp-config" }`.

**S3 — `detectFromTerminals()`**
```typescript
const pattern = /claude|cline|continue|cursor|copilot/i;
for (const term of vscode.window.terminals) {
  const shellPath = (term.creationOptions as vscode.TerminalOptions).shellPath ?? '';
  if (pattern.test(term.name) || pattern.test(shellPath)) {
    yield { agentName: `${term.name} (terminal)`, capabilities: ['terminal', 'code-execute'], ... };
  }
}
```

**S4 — `detectActiveLMParticipants()`**
```typescript
if (vscode.lm?.selectChatModels) {
  const models = await vscode.lm.selectChatModels({});
  for (const m of models) {
    if (m.vendor !== 'copilot') {  // copilot handled by S1
      yield { agentName: `${m.vendor}/${m.family}`, capabilities: ['chat'], detectionMethod: 'lm-participant' };
    }
  }
}
```

### Session lifecycle management

Internal `Map<externalId, string>` (externalId → sessionId). Per tick:
1. Call all strategies → collect `DetectedAgent[]` (deduplicated by `externalId`).
2. For each new agent (not in map): `client.registerSession(...)` → store returned session ID.
3. For each existing agent (in map): `client.heartbeatSession(sessionId)`.
4. For each disappeared agent (was in map, not in current list): `client.deregisterSession(sessionId)` + remove from map.
5. Fire `onDidChange` emitter.

All network calls wrapped in try/catch; failures logged to activity channel, not surfaced as VS Code notifications. Guard all calls with `if (this.isDisposed) return`.

## Acceptance Criteria

- Unit tests in TASK-290 pass
- `AgentDetector` disposes cleanly when the extension deactivates (clearInterval called)
- Does NOT throw any unhandled promise rejections
