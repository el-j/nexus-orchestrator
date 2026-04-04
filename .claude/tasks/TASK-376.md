# TASK-376: Add workspace isolation tests to taskQueueProvider.test.ts

**Plan:** PLAN-055 | **Wave:** 2 | **Status:** done | **Role:** extension

## Goal

Extend `vscode-extension/src/taskQueueProvider.test.ts` with tests verifying that `TaskQueueProvider` correctly scopes tasks to the current workspace folder and that the scope toggle works.

## Files to Edit

- `vscode-extension/src/taskQueueProvider.test.ts` — add describe blocks

## Tests to Add

```typescript
describe('TaskQueueProvider — workspace isolation', () => {
  const projectATasks: Task[] = [
    { id: 'a1', projectPath: '/workspace/project-a', instruction: 'task a1', status: 'QUEUED', ... },
    { id: 'a2', projectPath: '/workspace/project-a', instruction: 'task a2', status: 'COMPLETED', ... },
  ];
  const projectBTasks: Task[] = [
    { id: 'b1', projectPath: '/workspace/project-b', instruction: 'task b1', status: 'QUEUED', ... },
  ];
  const allTasks = [...projectATasks, ...projectBTasks];

  beforeEach(() => {
    // Mock vscode.workspace.workspaceFolders to return project-a
    vi.mocked(vscode.workspace).workspaceFolders = [
      { uri: { fsPath: '/workspace/project-a' }, name: 'project-a', index: 0 }
    ] as any;
    // Mock config: scopeToWorkspace = true
    vi.mocked(vscode.workspace.getConfiguration).mockReturnValue({
      get: vi.fn((key: string, def: unknown) => key === 'scopeToWorkspace' ? true : def),
    } as any);
  });

  it('filters to workspace folder when scopeToWorkspace=true', async () => {
    const { TaskQueueProvider } = await import('./taskQueueProvider');
    const client = { getAllTasks: vi.fn().mockResolvedValue(allTasks) } as unknown as NexusClient;
    const provider = new TaskQueueProvider(client);
    const children = await provider.getChildren();
    // Should only show project-a tasks
    expect(children.every(c => 'task' in c && (c as any).task.projectPath === '/workspace/project-a')).toBe(true);
    expect(children).toHaveLength(2);
  });

  it('shows all tasks when scopeToWorkspace=false', async () => {
    vi.mocked(vscode.workspace.getConfiguration).mockReturnValue({
      get: vi.fn((key: string, def: unknown) => key === 'scopeToWorkspace' ? false : def),
    } as any);
    const { TaskQueueProvider } = await import('./taskQueueProvider');
    const client = { getAllTasks: vi.fn().mockResolvedValue(allTasks) } as unknown as NexusClient;
    const provider = new TaskQueueProvider(client);
    const children = await provider.getChildren();
    expect(children).toHaveLength(3);
  });

  it('toggle switches scope and fires refresh', async () => {
    const { TaskQueueProvider } = await import('./taskQueueProvider');
    const client = { getAllTasks: vi.fn().mockResolvedValue(allTasks) } as unknown as NexusClient;
    const provider = new TaskQueueProvider(client);
    // Initially scoped to workspace → 2 tasks
    let children = await provider.getChildren();
    expect(children).toHaveLength(2);
    // Toggle off
    provider.setScopeToWorkspace(false);
    children = await provider.getChildren();
    expect(children).toHaveLength(3);
    // Toggle back on
    provider.setScopeToWorkspace(true);
    children = await provider.getChildren();
    expect(children).toHaveLength(2);
  });
});
```

## Verification

- `npm test` in vscode-extension directory passes all tests including new describe blocks
- 3 new test cases all green

## Status

done
