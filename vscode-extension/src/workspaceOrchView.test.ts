import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { WorkspaceOrchestration } from './workspaceScanner';

const workspaceFolders: Array<{ uri: { fsPath: string } }> = [];

class ThemeIcon {
  constructor(public readonly id: string) {}
}

class EventEmitter<T> {
  event = vi.fn();
  fire = vi.fn<(value?: T) => void>();
  dispose = vi.fn();
}

class TreeItem {
  label: string;
  description?: string;
  tooltip?: unknown;
  iconPath?: unknown;
  contextValue?: string;

  constructor(label: string) {
    this.label = label;
  }
}

vi.mock('vscode', () => ({
  ThemeIcon,
  EventEmitter,
  TreeItem,
  TreeItemCollapsibleState: {
    None: 0,
    Collapsed: 1,
    Expanded: 2,
  },
  workspace: {
    get workspaceFolders() {
      return workspaceFolders;
    },
  },
}));

function buildScanner(orchestrations: WorkspaceOrchestration[]) {
  return {
    scan: vi.fn(),
    getOrchestrations: vi.fn(() => orchestrations),
    onDidChange: vi.fn((_cb: () => void) => ({ dispose: vi.fn() })),
  };
}

describe('WorkspaceOrchViewProvider', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    workspaceFolders.length = 0;
  });

  it('returns empty array when workspace has no folders', async () => {
    const { WorkspaceOrchViewProvider } = await import('./workspaceOrchView');
    const provider = new WorkspaceOrchViewProvider(buildScanner([]) as never);

    const children = await provider.getChildren();

    expect(children).toEqual([]);
  });

  it('renders valid orchestration folder tree with task nodes', async () => {
    workspaceFolders.push({ uri: { fsPath: '/repo-one' } });
    const { WorkspaceOrchViewProvider } = await import('./workspaceOrchView');

    const orchestration: WorkspaceOrchestration = {
      folderPath: '/repo-one',
      folderName: 'repo-one',
      lastModified: new Date('2026-04-04T00:00:00Z'),
      activePlan: {
        id: 'PLAN-1',
        title: 'Active Plan',
        status: 'active',
        tasks: [
          { id: 'TASK-1', title: 'Do one', status: 'todo', role: 'coder' },
          { id: 'TASK-2', title: 'Do two', status: 'done', role: 'tester' },
        ],
      },
      allPlans: [
        {
          id: 'PLAN-1',
          title: 'Active Plan',
          status: 'active',
          tasks: [
            { id: 'TASK-1', title: 'Do one', status: 'todo', role: 'coder' },
            { id: 'TASK-2', title: 'Do two', status: 'done', role: 'tester' },
          ],
        },
        {
          id: 'PLAN-0',
          title: 'Old Plan',
          status: 'completed',
          tasks: [{ id: 'TASK-0', title: 'Done', status: 'done', role: 'coder' }],
        },
      ],
    };

    const provider = new WorkspaceOrchViewProvider(buildScanner([orchestration]) as never);

    const roots = await provider.getChildren();
    expect(roots).toHaveLength(1);
    expect((roots[0] as { label: string }).label).toBe('repo-one');

    const folderChildren = await provider.getChildren(roots[0] as never);
    expect(folderChildren).toHaveLength(2);
    expect((folderChildren[0] as { label: string }).label).toContain('Active: Active Plan');
    expect((folderChildren[1] as { label: string }).label).toBe('History');

    const activeTasks = await provider.getChildren(folderChildren[0] as never);
    expect(activeTasks).toHaveLength(2);

    const historyPlans = await provider.getChildren(folderChildren[1] as never);
    expect(historyPlans).toHaveLength(1);
    const historyPlanTasks = await provider.getChildren(historyPlans[0] as never);
    expect(historyPlanTasks).toHaveLength(1);
  });

  it('handles malformed or missing orchestration data safely', async () => {
    workspaceFolders.push({ uri: { fsPath: '/repo-one' } });
    const { WorkspaceOrchViewProvider } = await import('./workspaceOrchView');
    const provider = new WorkspaceOrchViewProvider(buildScanner([]) as never);

    await expect(provider.getChildren()).resolves.not.toThrow();
    const roots = await provider.getChildren();

    expect(roots).toHaveLength(1);
    expect((roots[0] as { label: string }).label).toContain(
      'No orchestrator.json found in workspace',
    );
  });

  it('shows both workspace folders as root nodes in multi-folder mode', async () => {
    workspaceFolders.push({ uri: { fsPath: '/repo-one' } }, { uri: { fsPath: '/repo-two' } });
    const { WorkspaceOrchViewProvider } = await import('./workspaceOrchView');

    const orchs: WorkspaceOrchestration[] = [
      {
        folderPath: '/repo-one',
        folderName: 'repo-one',
        lastModified: new Date('2026-04-04T00:00:00Z'),
        activePlan: null,
        allPlans: [],
      },
      {
        folderPath: '/repo-two',
        folderName: 'repo-two',
        lastModified: new Date('2026-04-04T00:00:00Z'),
        activePlan: null,
        allPlans: [],
      },
    ];

    const provider = new WorkspaceOrchViewProvider(buildScanner(orchs) as never);

    const roots = await provider.getChildren();

    expect(roots).toHaveLength(2);
    expect((roots[0] as { label: string }).label).toBe('repo-one');
    expect((roots[1] as { label: string }).label).toBe('repo-two');
  });
});
