import { beforeEach, describe, expect, it, vi } from 'vitest';

const workspaceFolders: Array<{ uri: { fsPath: string } }> = [];
const existsSyncMock = vi.fn();
const readFileSyncMock = vi.fn();
const statSyncMock = vi.fn();

class EventEmitter<T> {
  event = vi.fn();
  fire = vi.fn<(value?: T) => void>();
  dispose = vi.fn();
}

class RelativePattern {
  constructor(
    public readonly _folder: unknown,
    public readonly _pattern: string,
  ) {}
}

vi.mock('fs', () => ({
  existsSync: existsSyncMock,
  readFileSync: readFileSyncMock,
  statSync: statSyncMock,
}));

vi.mock('vscode', () => ({
  EventEmitter,
  RelativePattern,
  workspace: {
    get workspaceFolders() {
      return workspaceFolders;
    },
    createFileSystemWatcher: vi.fn(() => ({
      onDidChange: vi.fn(),
      onDidCreate: vi.fn(),
      onDidDelete: vi.fn(),
      dispose: vi.fn(),
    })),
    onDidChangeWorkspaceFolders: vi.fn(() => ({ dispose: vi.fn() })),
  },
}));

function buildContext(): import('vscode').ExtensionContext {
  return {
    subscriptions: [],
  } as unknown as import('vscode').ExtensionContext;
}

describe('WorkspaceScanner', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    workspaceFolders.length = 0;
    workspaceFolders.push({ uri: { fsPath: '/repo-one' } });
    statSyncMock.mockReturnValue({ mtime: new Date('2026-04-04T00:00:00Z') });
  });

  it('returns safe default when orchestrator file is missing', async () => {
    existsSyncMock.mockReturnValue(false);

    const { WorkspaceScanner } = await import('./workspaceScanner');
    const scanner = new WorkspaceScanner(buildContext());

    scanner.scan();

    expect(scanner.getOrchestrations()).toEqual([]);
  });

  it('returns safe default when orchestrator file is empty', async () => {
    existsSyncMock.mockReturnValue(true);
    readFileSyncMock.mockReturnValue('');

    const { WorkspaceScanner } = await import('./workspaceScanner');
    const scanner = new WorkspaceScanner(buildContext());

    scanner.scan();

    expect(scanner.getOrchestrations()).toEqual([]);
  });

  it('parses valid orchestrator file fields correctly', async () => {
    existsSyncMock.mockReturnValue(true);
    readFileSyncMock.mockReturnValue(
      JSON.stringify({
        activePlanId: 'PLAN-1',
        plans: {
          'PLAN-1': {
            id: 'PLAN-1',
            title: 'Plan One',
            status: 'active',
            tasks: ['TASK-1'],
          },
        },
        tasks: {
          'TASK-1': {
            id: 'TASK-1',
            title: 'Implement tests',
            status: 'todo',
            role: 'coder',
          },
        },
      }),
    );

    const { WorkspaceScanner } = await import('./workspaceScanner');
    const scanner = new WorkspaceScanner(buildContext());

    scanner.scan();
    const orchs = scanner.getOrchestrations();

    expect(orchs).toHaveLength(1);
    expect(orchs[0].activePlan?.id).toBe('PLAN-1');
    expect(orchs[0].allPlans).toHaveLength(1);
    expect(orchs[0].allPlans[0].tasks).toHaveLength(1);
    expect(orchs[0].allPlans[0].tasks[0]).toEqual({
      id: 'TASK-1',
      title: 'Implement tests',
      status: 'todo',
      role: 'coder',
    });
  });

  it('ignores unknown fields while parsing known values', async () => {
    existsSyncMock.mockReturnValue(true);
    readFileSyncMock.mockReturnValue(
      JSON.stringify({
        activePlanId: 'PLAN-2',
        plans: {
          'PLAN-2': {
            id: 'PLAN-2',
            title: 'Plan Two',
            status: 'active',
            tasks: ['TASK-2'],
            randomPlanField: 'ignored',
          },
        },
        tasks: {
          'TASK-2': {
            id: 'TASK-2',
            title: 'Known Task',
            status: 'done',
            role: 'reviewer',
            extraTaskField: 123,
          },
        },
        unknownRootField: { anything: true },
      }),
    );

    const { WorkspaceScanner } = await import('./workspaceScanner');
    const scanner = new WorkspaceScanner(buildContext());

    scanner.scan();
    const orchs = scanner.getOrchestrations();

    expect(orchs).toHaveLength(1);
    expect(orchs[0].activePlan?.id).toBe('PLAN-2');
    expect(orchs[0].allPlans[0].tasks[0].id).toBe('TASK-2');
    expect(orchs[0].allPlans[0].tasks[0].status).toBe('done');
  });

  it('handles counters.nextTaskId as string gracefully', async () => {
    existsSyncMock.mockReturnValue(true);
    readFileSyncMock.mockReturnValue(
      JSON.stringify({
        activePlanId: 'PLAN-3',
        counters: {
          nextTaskId: '490',
        },
        plans: {
          'PLAN-3': {
            id: 'PLAN-3',
            title: 'Plan Three',
            status: 'active',
            tasks: ['TASK-3'],
          },
        },
        tasks: {
          'TASK-3': {
            id: 'TASK-3',
            title: 'Counter coercion coverage',
            status: 'todo',
            role: 'coder',
          },
        },
      }),
    );

    const { WorkspaceScanner } = await import('./workspaceScanner');
    const scanner = new WorkspaceScanner(buildContext());

    expect(() => scanner.scan()).not.toThrow();
    const orchs = scanner.getOrchestrations();

    expect(orchs).toHaveLength(1);
    expect(orchs[0].activePlan?.id).toBe('PLAN-3');
    expect(orchs[0].allPlans[0].tasks[0].id).toBe('TASK-3');
  });
});
