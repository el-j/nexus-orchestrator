# TASK-288 — Extension wiring: integrate AgentDetector into extension.ts

**Plan:** PLAN-044  
**Status:** TODO  
**Layer:** TypeScript · VS Code extension  
**Depends on:** TASK-282, TASK-283, TASK-284  

## Objective

Connect all PLAN-044 components into the extension activation lifecycle and declare new contribution points in `package.json`.

## Changes

### `vscode-extension/src/extension.ts`

Add imports:
```typescript
import { AgentDetector } from './agentDetector';
import { AISessionsTreeProvider } from './aiSessionsTreeProvider';
import { delegateToNexusCommand } from './commands/delegateToNexus';
```

In `activate()` (after `SessionMonitor.start()`):

```typescript
// ── Universal Agent Detector ────────────────────────────────────────────────
const agentDetector = new AgentDetector(getClient(), context);
agentDetector.start();
context.subscriptions.push(agentDetector);

// ── AI Sessions tree view ───────────────────────────────────────────────────
const aiSessionsProvider = new AISessionsTreeProvider(getClient());
context.subscriptions.push(
  vscode.window.registerTreeDataProvider('nexus.aiSessions', aiSessionsProvider)
);
// Refresh tree whenever AgentDetector detects changes
agentDetector.onDidChange(() => aiSessionsProvider.refresh());
context.subscriptions.push(aiSessionsProvider.startPolling(15_000));

// ── nexus.refreshAISessions ─────────────────────────────────────────────────
context.subscriptions.push(
  vscode.commands.registerCommand('nexus.refreshAISessions', () => {
    aiSessionsProvider.refresh();
    void agentDetector.detectAll();
  })
);

// ── nexus.delegateToNexus ───────────────────────────────────────────────────
context.subscriptions.push(
  vscode.commands.registerCommand('nexus.delegateToNexus', (item?: AISessionItem) =>
    delegateToNexusCommand(getClient(), item)
  )
);

// ── nexus.delegateAllSessions ───────────────────────────────────────────────
context.subscriptions.push(
  vscode.commands.registerCommand('nexus.delegateAllSessions', async () => {
    const sessions = await getClient().listAISessions();
    const active = sessions.filter(s => s.status === 'active' && !s.delegatedToNexus);
    for (const s of active) {
      await delegateToNexusCommand(getClient(), s);
    }
  })
);
```

### `vscode-extension/package.json` additions

Under `contributes.commands`:
```json
{ "command": "nexus.delegateToNexus",      "title": "Send to Nexus Orchestrator", "icon": "$(arrow-up)" },
{ "command": "nexus.refreshAISessions",    "title": "Refresh AI Agents",          "icon": "$(refresh)" },
{ "command": "nexus.delegateAllSessions",  "title": "Delegate All Active Agents" }
```

Under `contributes.views.nexus` (add after existing views):
```json
{ "id": "nexus.aiSessions", "name": "AI Agents", "icon": "$(robot)" }
```

Under `contributes.menus["view/title"]`:
```json
{ "command": "nexus.refreshAISessions", "when": "view == nexus.aiSessions", "group": "navigation" }
```

Under `contributes.menus["view/item/context"]`:
```json
{ "command": "nexus.delegateToNexus", "when": "view == nexus.aiSessions && viewItem == aiSession", "group": "inline" }
```

### `vscode-extension/package.json` — new configuration property

```json
"nexus.enableMCPPortSweep": {
  "type": "boolean",
  "default": true,
  "description": "When true, nexusOrchestrator probes a curated set of localhost ports for MCP-compatible servers during agent discovery. Disable if you experience unexpected connections to local services."
}
```

## Acceptance Criteria

- Extension activates without errors after these changes
- `nexus.aiSessions` tree view appears in the nexus side panel
- `nexus.delegateToNexus` appears in the inline context menu of each AI session tree item
- `AgentDetector` is disposed cleanly on extension deactivation
- TypeScript compilation clean (`npx tsc --noEmit`)
