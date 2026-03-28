# TASK-384: VS Code extension — workspace isolation tests

**Plan:** PLAN-055 | **Wave:** 5 | **Status:** done | **Role:** testing

## Goal

The existing `taskQueueProvider.test.ts` has 1 test but doesn't test:

- Project-scoped filtering (the TASK-375 feature)
- WorkspaceOrchView project isolation (TASK-377 feature)
- Scope toggle persistence

This task depends on TASK-375 and TASK-376 being implemented. The tests here provide verification that the extension features introduced in Wave 2 work correctly.

## File to Edit

- `vscode-extension/src/taskQueueProvider.test.ts` — TASK-376 creates the isolation tests
- This task creates: `vscode-extension/src/workspaceOrchView.test.ts`

## workspaceOrchView.test.ts

```typescript
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { NexusClient, Task } from './nexusClient';

// VS Code mock includes workspaceFolders
vi.mock('vscode', () => ({
  workspace: {
    workspaceFolders: [
      {
        uri: { fsPath: '/project-a' },
        name: 'project-a',
        index: 0,
      },
    ],
    getConfiguration: vi.fn().mockReturnValue({
      get: vi.fn().mockReturnValue(true),
    }),
  },
  EventEmitter: class {
    event = vi.fn();
    fire = vi.fn();
  },
  TreeItem: class {
    constructor(label: string) {
      (this as any).label = label;
    }
  },
  TreeItemCollapsibleState: { Collapsed: 1, Expanded: 2, None: 0 },
  ThemeIcon: class {
    constructor(public id: string) {}
  },
  MarkdownString: class {
    constructor(public value: string) {}
  },
}));

describe('WorkspaceOrchView — task isolation', () => {
  const allTasks: Task[] = [
    {
      id: 't1',
      projectPath: '/project-a',
      instruction: 'task a',
      status: 'QUEUED',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      contextFiles: [],
      providerHint: '',
      modelId: '',
      targetFile: '',
      tags: [],
    },
    {
      id: 't2',
      projectPath: '/project-b',
      instruction: 'task b',
      status: 'QUEUED',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      contextFiles: [],
      providerHint: '',
      modelId: '',
      targetFile: '',
      tags: [],
    },
    {
      id: 't3',
      projectPath: '/project-a',
      instruction: 'task a2',
      status: 'COMPLETED',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      contextFiles: [],
      providerHint: '',
      modelId: '',
      targetFile: '',
      tags: [],
    },
  ];

  it('LiveTasksGroupNode only contains tasks for its folder', async () => {
    const { WorkspaceOrchView } = await import('./workspaceOrchView');
    const client = { getAllTasks: vi.fn().mockResolvedValue(allTasks) } as unknown as NexusClient;
    const view = new WorkspaceOrchView(client);
    const topItems = await view.getChildren();
    // Find the folder node for project-a
    const projANode = topItems.find((item) => (item as any).folderPath === '/project-a');
    expect(projANode).toBeDefined();
    const folderChildren = await view.getChildren(projANode);
    // Find LiveTasksGroupNode
    const liveNode = folderChildren.find((c) => (c as any).kind === 'liveTasks');
    expect(liveNode).toBeDefined();
    // LiveTasksGroupNode.tasks should only contain /project-a tasks
    expect((liveNode as any).tasks).toHaveLength(2);
    (liveNode as any).tasks.forEach((t: Task) => {
      expect(t.projectPath).toBe('/project-a');
    });
  });
});
```

## Verification

- `npm test` in vscode-extension passes all existing + new tests
- `WorkspaceOrchView` test confirms per-folder task isolation

## Status

done
