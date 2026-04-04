# TASK-284 — VS Code: `nexus.delegateToNexus` command

**Plan:** PLAN-044  
**Status:** TODO  
**Layer:** TypeScript · VS Code extension  
**Depends on:** TASK-282, TASK-285  
**New file:** `vscode-extension/src/commands/delegateToNexus.ts`  

## Objective

Implement the delegation command with three delivery paths depending on agent type.

## Function signature

```typescript
export async function delegateToNexusCommand(
  client: NexusClient,
  sessionOrItem?: AISessionItem | AISession
): Promise<void>
```

When invoked from command palette (no arg), uses `vscode.window.showQuickPick` over `await client.listAISessions()` (filtered to `status === "active"`, not yet delegated).

## Delegation paths

### Determine path

```typescript
function delegationPath(session: AISession): 'cli' | 'mcp' | 'copilot' {
  if (session.agentName === 'GitHub Copilot' || session.agentName === 'GitHub Copilot Chat') return 'copilot';
  if (session.source === 'mcp' ||
      session.agentName.includes('Claude Desktop') ||
      session.agentName.includes('Antigravity')) return 'mcp';
  return 'cli';   // default: Cline, Continue, Claude CLI, terminal agents
}
```

### Path A — CLI / Terminal (`'cli'`)

```typescript
const projectPath = session.projectPath ?? vscode.workspace.workspaceFolders?.[0]?.uri.fsPath ?? '';
const filePath = vscode.Uri.file(path.join(projectPath, '.nexus-delegate.md'));
await vscode.workspace.fs.writeFile(filePath, Buffer.from(instruction, 'utf8'));
const terminal = vscode.window.createTerminal({ name: 'Nexus Delegate' });
terminal.show();
terminal.sendText(`echo "=== Nexus Delegation Instruction ===" && cat "${filePath.fsPath}"`);
vscode.window.showInformationMessage(
  'Delegation instruction written to .nexus-delegate.md',
  'Open File'
).then(action => { if (action === 'Open File') vscode.window.showTextDocument(filePath); });
```

### Path B — MCP-connected (`'mcp'`)

```typescript
// 1. Submit a nexus task to capture the pending work
const projectPath = session.projectPath ?? vscode.workspace.workspaceFolders?.[0]?.uri.fsPath ?? '';
await client.submitTask({
  instruction,
  projectPath,
  targetFile: '.nexus-delegate.md',
  command: 'auto',
});
// 2. If Copilot Chat available, open it with instruction pre-filled
try {
  await vscode.commands.executeCommand('workbench.action.chat.open', { query: instruction });
} catch {
  // Chat not available — show notification with copy action
  vscode.window.showInformationMessage('Task submitted to Nexus queue.', 'Copy Instruction')
    .then(a => { if (a === 'Copy Instruction') vscode.env.clipboard.writeText(instruction); });
}
```

### Path C — GitHub Copilot (`'copilot'`)

```typescript
try {
  await vscode.commands.executeCommand('workbench.action.chat.open', { query: instruction });
} catch {
  vscode.window.showInformationMessage(
    'Could not open Copilot Chat. Copy the delegation instruction?', 'Copy'
  ).then(a => { if (a === 'Copy') vscode.env.clipboard.writeText(instruction); });
}
```

## After dispatch

```typescript
// Refresh tree view so delegated session turns green
vscode.commands.executeCommand('nexus.refreshAISessions');
```

## Acceptance Criteria

- Path A writes `.nexus-delegate.md` and opens terminal
- Path C calls `workbench.action.chat.open`
- All paths handle `client.delegateSession` failure gracefully (show error notification)
- Unit tests in TASK-290 pass
