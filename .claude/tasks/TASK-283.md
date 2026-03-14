# TASK-283 — VS Code: `AISessionsTreeProvider`

**Plan:** PLAN-044  
**Status:** TODO  
**Layer:** TypeScript · VS Code extension  
**Depends on:** TASK-285  
**New file:** `vscode-extension/src/aiSessionsTreeProvider.ts`  

## Objective

New tree view (`nexus.aiSessions`) showing all registered AI sessions with live status colours and inline "Delegate" button.

## Tree item structure

```typescript
export class AISessionItem extends vscode.TreeItem {
  constructor(public readonly session: AISession) {
    super(session.agentName, vscode.TreeItemCollapsibleState.None);
    this.contextValue = 'aiSession';
    this.description = this.buildDescription();
    this.iconPath = this.buildIcon();
    this.tooltip = this.buildTooltip();
  }
}
```

Status → VS Code theme colour mappings:
| Condition | ThemeIcon color |
|---|---|
| `active` + `delegatedToNexus` | `charts.green` |
| `active` | `charts.yellow` |
| `idle` | `charts.orange` |
| `disconnected` | `disabledForeground` |

Description: `[${source}] ${last 2 path segments of projectPath}` (if projectPath set)
Tooltip: full `projectPath`, `id`, `lastActivity`, capabilities list.

## `AISessionsTreeProvider` class

```typescript
export class AISessionsTreeProvider implements vscode.TreeDataProvider<AISessionItem>, vscode.Disposable {
  private readonly _onDidChangeTreeData = new vscode.EventEmitter<void>();
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event;
  private sessions: AISession[] = [];
  private pollingTimer: NodeJS.Timeout | undefined;

  constructor(private readonly client: NexusClient) {}

  refresh(): void           // fires _onDidChangeTreeData
  startPolling(ms: number): vscode.Disposable
  getTreeItem(element: AISessionItem): vscode.TreeItem
  getChildren(): Promise<AISessionItem[]>   // fetches /api/ai-sessions
}
```

`getChildren()` calls `client.listAISessions()` (new method, see TASK-285), updates `this.sessions`, returns sorted list (active first, then idle, then disconnected; within a group sort by `lastActivity` desc).

## `package.json` additions

Under `contributes.views.nexus`:
```json
{
  "id": "nexus.aiSessions",
  "name": "AI Agents",
  "icon": "$(robot)",
  "when": "true"
}
```

Under `contributes.menus["view/title"]`:
```json
{
  "command": "nexus.refreshAISessions",
  "when": "view == nexus.aiSessions",
  "group": "navigation"
}
```

Under `contributes.menus["view/item/context"]`:
```json
{
  "command": "nexus.delegateToNexus",
  "when": "view == nexus.aiSessions && viewItem == aiSession",
  "group": "inline"
}
```

## Acceptance Criteria

- Tree view renders without errors when daemon is unreachable (returns empty list)
- Delegated sessions show green icon colour
- `refresh()` causes tree to re-query the daemon within 100 ms
