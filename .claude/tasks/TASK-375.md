# TASK-375: VS Code extension — TaskQueueProvider workspace isolation + toggle

**Plan:** PLAN-055 | **Wave:** 2 | **Status:** done | **Role:** extension

## Problem

`TaskQueueProvider.getChildren()` calls `this.client.getAllTasks()` with no project filter. When the user has multiple VS Code windows open (one for each project), every window shows every task from every project. This is confusing and incorrect.

## Strategy

1. Detect the current workspace folder via `vscode.workspace.workspaceFolders`
2. Filter `getAllTasks()` results to tasks where `task.projectPath === currentFolder`
3. Configuration setting `nexus.scopeToWorkspace` (default: `true`) — when `false`, show all projects (existing behavior)
4. Show a scope indicator in the tree view title: "Tasks (this project)" vs "Tasks (all projects)"

## Files to Edit

- `vscode-extension/src/taskQueueProvider.ts` — add `workspacePath` param + config check
- `vscode-extension/src/extension.ts` — pass workspace path when constructing `TaskQueueProvider`
- `package.json` (vscode-extension) — add `nexus.scopeToWorkspace` config entry

## Implementation

### taskQueueProvider.ts changes:

```typescript
export class TaskQueueProvider implements vscode.TreeDataProvider<TaskItem | DaemonOfflineItem> {
  private workspacePath: string | undefined;
  private scopeToWorkspace = true;

  constructor(private readonly client: NexusClient) {
    // Read VS Code config
    const cfg = vscode.workspace.getConfiguration('nexus');
    this.scopeToWorkspace = cfg.get<boolean>('scopeToWorkspace', true);
    // Determine current workspace root
    this.workspacePath = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
  }

  setScopeToWorkspace(scoped: boolean): void {
    this.scopeToWorkspace = scoped;
    this.refresh();
  }

  async getChildren(): Promise<(TaskItem | DaemonOfflineItem)[]> {
    try {
      const tasks = await this.client.getAllTasks();
      const filtered =
        this.scopeToWorkspace && this.workspacePath
          ? tasks.filter((t) => t.projectPath === this.workspacePath)
          : tasks;
      const active = filtered
        .filter((t) => t.status === 'QUEUED' || t.status === 'PROCESSING')
        .sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime());
      const recent = filtered
        .filter((t) => t.status !== 'QUEUED' && t.status !== 'PROCESSING')
        .sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime());
      return [...active, ...recent].map((t) => new TaskItem(t));
    } catch {
      return [new DaemonOfflineItem()];
    }
  }
}
```

### package.json configuration:

```json
{
  "contributes": {
    "configuration": {
      "properties": {
        "nexus.scopeToWorkspace": {
          "type": "boolean",
          "default": true,
          "description": "Show only tasks from the current workspace folder. Disable to see tasks from all projects."
        }
      }
    },
    "commands": [
      {
        "command": "nexus.toggleWorkspaceScope",
        "title": "Nexus: Toggle Workspace Scope (current / all projects)",
        "icon": "$(filter)"
      }
    ],
    "menus": {
      "view/title": [
        {
          "command": "nexus.toggleWorkspaceScope",
          "when": "view == nexusTaskQueue",
          "group": "navigation"
        }
      ]
    }
  }
}
```

### extension.ts — wire toggle command:

```typescript
context.subscriptions.push(
  vscode.commands.registerCommand('nexus.toggleWorkspaceScope', () => {
    const current = taskQueueProvider.getScopeToWorkspace();
    taskQueueProvider.setScopeToWorkspace(!current);
    vscode.window.showInformationMessage(
      `Nexus task scope: ${!current ? 'current workspace' : 'all projects'}`,
    );
  }),
);
```

## Verification

- `npm test` in vscode-extension passes
- With `scopeToWorkspace = true`: only tasks where `projectPath === workspaceFolders[0].fsPath` shown
- With `scopeToWorkspace = false`: all tasks shown
- Toggle command switches between modes

## Status

done
