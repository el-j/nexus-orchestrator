import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { AIActivity, BrainStatus, NexusClient, Task } from './nexusClient';
import type { OrchestratorPlan, WorkspaceOrchestration } from './workspaceScanner';

vi.mock('vscode', async () => (await import('./test/fakeVscode')).createFakeVscode());

import { fakeState, resetFakeState, ThemeIcon, Uri } from './test/fakeVscode';
import { WorkspaceOrchViewProvider } from './workspaceOrchView';

const NOW = new Date('2026-06-01T12:00:00Z').getTime();
const plan = (o: Partial<OrchestratorPlan> = {}): OrchestratorPlan => ({
  id: 'PLAN-1',
  title: 'First plan',
  status: 'active',
  tasks: [],
  ...o,
});
const orch = (o: Partial<WorkspaceOrchestration> = {}): WorkspaceOrchestration => ({
  folderPath: '/workspace',
  folderName: 'workspace',
  activePlan: null,
  allPlans: [],
  lastModified: new Date(NOW),
  ...o,
});
const task = (o: Partial<Task> = {}): Task => ({
  id: 'abcdef123456',
  projectPath: '/workspace',
  targetFile: '',
  instruction: 'Do it',
  contextFiles: [],
  status: 'QUEUED',
  createdAt: '',
  updatedAt: '2026-01-01T00:00:00Z',
  ...o,
});
const activity = (o: Partial<AIActivity> = {}): AIActivity => ({
  id: 'a',
  agentName: 'claude',
  activityType: 'message',
  summary: 'did a thing',
  timestamp: new Date(NOW - 5_000).toISOString(),
  ...o,
});
const brain = (o: Partial<BrainStatus> = {}): BrainStatus => ({
  projectPath: '/workspace',
  initialized: true,
  entryCount: 3,
  kindCounts: {},
  totalTokens: 100,
  lastUpdated: '2026-01-01T00:00:00Z',
  ...o,
});

function scannerWith(orchs: WorkspaceOrchestration[]) {
  let listener: (() => void) | undefined;
  return {
    scan: vi.fn(),
    getOrchestrations: vi.fn(() => orchs),
    onDidChange: vi.fn((l: () => void) => {
      listener = l;
    }),
    fire: () => listener?.(),
  };
}
type Scanner = ReturnType<typeof scannerWith>;
const asScanner = (s: Scanner) =>
  s as unknown as ConstructorParameters<typeof WorkspaceOrchViewProvider>[0];

interface ClientData {
  activities?: AIActivity[] | Error;
  tasks?: Task[] | Error;
  brain?: BrainStatus | Error;
}
const client = (d: ClientData = {}) =>
  ({
    getActivities: vi.fn(() =>
      d.activities instanceof Error
        ? Promise.reject(d.activities)
        : Promise.resolve(d.activities ?? []),
    ),
    getAllTasks: vi.fn(() =>
      d.tasks instanceof Error ? Promise.reject(d.tasks) : Promise.resolve(d.tasks ?? []),
    ),
    getBrainStatus: vi.fn(() =>
      d.brain instanceof Error ? Promise.reject(d.brain) : Promise.resolve(d.brain ?? brain()),
    ),
  }) as unknown as NexusClient;

const labels = (nodes: Array<{ label?: unknown }>) => nodes.map((n) => String(n.label));
const iconId = (n: { iconPath?: unknown }) => (n.iconPath as ThemeIcon | undefined)?.id;

beforeEach(() => {
  resetFakeState();
  vi.useFakeTimers();
  vi.setSystemTime(NOW);
});
afterEach(() => vi.useRealTimers());

describe('root', () => {
  it('is empty without workspace folders, and explains that no plans or tasks were found', async () => {
    fakeState.folders = undefined;
    expect(await new WorkspaceOrchViewProvider(asScanner(scannerWith([]))).getChildren()).toEqual(
      [],
    );
    fakeState.folders = [{ uri: Uri.file('/workspace'), name: 'w' }];
    const [empty] = await new WorkspaceOrchViewProvider(asScanner(scannerWith([]))).getChildren();
    expect(empty.label).toBe('No plans or tasks found in workspace (.claude/plans, .claude/tasks)');
    expect(iconId(empty)).toBe('info');
  });

  it('lists a folder node per orchestration with its plan count', async () => {
    const [f] = await new WorkspaceOrchViewProvider(
      asScanner(scannerWith([orch({ allPlans: [plan(), plan({ id: 'P2' })] })])),
    ).getChildren();
    expect(f.label).toBe('workspace');
    expect(f.description).toBe('2 plans');
    expect(f.tooltip).toBe('/workspace');
    expect(iconId(f)).toBe('folder');
  });

  it('returns the element itself as its tree item', async () => {
    const p = new WorkspaceOrchViewProvider(asScanner(scannerWith([orch()])));
    const [f] = await p.getChildren();
    expect(p.getTreeItem(f)).toBe(f);
  });
});

describe('refresh', () => {
  it('rescans and redraws without calling the daemon', () => {
    const s = scannerWith([orch()]);
    const c = client();
    const p = new WorkspaceOrchViewProvider(asScanner(s), c);
    const changed = vi.fn();
    p.onDidChangeTreeData(changed);
    p.refresh();
    expect(s.scan).toHaveBeenCalled();
    expect(changed).toHaveBeenCalled();
    expect(c.getActivities).not.toHaveBeenCalled();
    expect(c.getBrainStatus).not.toHaveBeenCalled();
  });

  it('redraws when the scanner reports a change', () => {
    const s = scannerWith([orch()]);
    const p = new WorkspaceOrchViewProvider(asScanner(s));
    const changed = vi.fn();
    p.onDidChangeTreeData(changed);
    s.fire();
    expect(changed).toHaveBeenCalled();
  });
});

describe('folder children without a daemon client', () => {
  it('shows the active plan, history of the others, or a placeholder', async () => {
    const active = plan({ id: 'A', title: 'Active one', status: 'completed' });
    const old = plan({ id: 'O', title: 'Old', status: 'failed' });
    const p = new WorkspaceOrchViewProvider(
      asScanner(scannerWith([orch({ activePlan: active, allPlans: [active, old] })])),
    );
    const [folder] = await p.getChildren();
    const kids = await p.getChildren(folder);
    expect(labels(kids)).toEqual(['Active: Active one', 'History']);
    expect(kids[0].description).toBe('A');
    expect(iconId(kids[0])).toBe('pass');
    expect(kids[1].description).toBe('1 plans');

    const bare = new WorkspaceOrchViewProvider(asScanner(scannerWith([orch()])));
    const [f2] = await bare.getChildren();
    expect(labels(await bare.getChildren(f2))).toEqual(['No plans found']);
  });

  it('expands plans into tasks with status icons', async () => {
    const tasks = ['done', 'in-progress', 'todo', 'blocked', 'weird'].map((status, i) => ({
      id: `T${i}`,
      title: `Task ${i}`,
      status,
      role: 'dev',
    }));
    const active = plan({ tasks, title: 'x'.repeat(60) });
    const p = new WorkspaceOrchViewProvider(
      asScanner(
        scannerWith([
          orch({
            activePlan: active,
            allPlans: [active, plan({ id: 'H', tasks: [tasks[0]], status: 'weird' })],
          }),
        ]),
      ),
    );
    const [folder] = await p.getChildren();
    const [activeNode, historyNode] = await p.getChildren(folder);
    const taskNodes = await p.getChildren(activeNode);
    expect(taskNodes.map((t) => iconId(t))).toEqual([
      'pass',
      'loading~spin',
      'circle-outline',
      'error',
      'question',
    ]);
    expect(taskNodes[0].label).toBe('T0 — Task 0');
    expect(taskNodes[0].description).toBe('dev');

    const [planNode] = await p.getChildren(historyNode);
    expect(planNode.label).toBe('H — First plan');
    expect(planNode.description).toBe('weird');
    expect(iconId(planNode)).toBe('circle-outline');
    expect(await p.getChildren(planNode)).toHaveLength(1);
  });

  it('uses the right icon for active, completed and failed plans', async () => {
    const make = (status: string) => plan({ id: status, status });
    const plans = ['active', 'completed', 'failed'].map(make);
    const p = new WorkspaceOrchViewProvider(asScanner(scannerWith([orch({ allPlans: plans })])));
    const [folder] = await p.getChildren();
    const [history] = await p.getChildren(folder);
    const nodes = await p.getChildren(history);
    expect(nodes.map((n) => iconId(n))).toEqual(['loading~spin', 'pass', 'error']);
  });
});

describe('folder children with the daemon', () => {
  const expand = async (c: NexusClient, o = orch({ activePlan: plan() })) => {
    const p = new WorkspaceOrchViewProvider(asScanner(scannerWith([o])), c);
    const [folder] = await p.getChildren();
    return { p, kids: await p.getChildren(folder) };
  };

  it('shows live activity, live tasks, the active plan and the brain, in that order', async () => {
    const { kids } = await expand(client({ activities: [activity()], tasks: [task()] }));
    expect(labels(kids)).toEqual([
      'Live Activity (1)',
      'Live Tasks (1)',
      'Active: First plan',
      'Project Brain',
    ]);
  });

  it('filters tasks to the folder (ignoring trailing slashes), active first then five recent', async () => {
    const tasks = [
      task({ id: 'other', projectPath: '/elsewhere' }),
      task({ id: 'q1', projectPath: '/workspace/' }),
      ...Array.from({ length: 7 }, (_, i) =>
        task({ id: `done${i}`, status: 'COMPLETED', updatedAt: `2026-01-0${i + 1}T00:00:00Z` }),
      ),
      task({ id: 'win', projectPath: 'C:\\win\\', status: 'FAILED' }),
    ];
    const { p, kids } = await expand(client({ tasks }));
    const group = kids.find((k) => String(k.label).startsWith('Live Tasks'))!;
    expect(group.label).toBe('Live Tasks (6)'); // 1 active + 5 most recent
    const items = await p.getChildren(group);
    expect(items.map((i) => (i as unknown as { task: Task }).task.id)).toEqual([
      'q1',
      'done6',
      'done5',
      'done4',
      'done3',
      'done2',
    ]);
  });

  it('matches Windows-style folder paths too', async () => {
    const win = orch({ folderPath: 'C:\\win\\', folderName: 'win' });
    const { kids } = await expand(client({ tasks: [task({ projectPath: 'C:\\win' })] }), win);
    expect(labels(kids)).toContain('Live Tasks (1)');
  });

  it('renders a live task with truncated text, status and a tooltip', async () => {
    const { p, kids } = await expand(
      client({ tasks: [task({ id: 'id0xxxxxxxx', instruction: 'y'.repeat(60) })] }),
      orch(),
    );
    const items = await p.getChildren(kids.find((k) => String(k.label).startsWith('Live Tasks'))!);
    expect(items[0].label).toBe('y'.repeat(50) + '\u2026');
    expect(items[0].description).toBe('QUEUED');
    expect(String(items[0].tooltip)).toBe('QUEUED · id0xxxxx · /workspace');
    expect(await p.getChildren(items[0])).toEqual([]);
  });

  it.each([
    ['QUEUED', 'clock'],
    ['PROCESSING', 'loading~spin'],
    ['COMPLETED', 'pass'],
    ['FAILED', 'error'],
    ['CANCELLED', 'circle-slash'],
    ['TOO_LARGE', 'file-binary'],
    ['NO_PROVIDER', 'plug'],
    ['DRAFT', 'edit'],
    ['BACKLOG', 'checklist'],
    ['MYSTERY', 'question'],
  ])('shows a %s live task with the %s icon', async (status, icon) => {
    const { p, kids } = await expand(
      client({ tasks: [task({ status: status as Task['status'] })] }),
      orch(),
    );
    const [item] = await p.getChildren(kids.find((k) => String(k.label).startsWith('Live Tasks'))!);
    expect(iconId(item)).toBe(icon);
  });

  it('renders activities with emoji, agent and age', async () => {
    const acts = [
      activity({ activityType: 'tool_use', summary: 'ran tool' }),
      activity({ activityType: 'thinking', timestamp: new Date(NOW - 30_000).toISOString() }),
      activity({ activityType: 'file_edit', timestamp: new Date(NOW - 5 * 60_000).toISOString() }),
      activity({
        activityType: 'generation',
        timestamp: new Date(NOW - 3 * 3600_000).toISOString(),
        projectPath: '/p',
      }),
      activity({ activityType: 'weird' as AIActivity['activityType'] }),
    ];
    const { p, kids } = await expand(client({ activities: acts }));
    const group = kids[0];
    expect(group.label).toBe('Live Activity (5)');
    const items = await p.getChildren(group);
    expect(String(items[0].label)).toBe('🔧 ran tool');
    expect(items.map((i) => i.description)).toEqual([
      'claude · just now',
      'claude · 30s ago',
      'claude · 5m ago',
      'claude · 3h ago',
      'claude · just now',
    ]);
    expect(String(items[4].label).startsWith('• ')).toBe(true);
    expect(String(items[3].tooltip)).toContain('/p');
    expect(await p.getChildren(items[0])).toEqual([]);
  });

  it('describes the brain, initialised or not, and wires the ingest command', async () => {
    const ready = await expand(client({ brain: brain() }), orch());
    const node = ready.kids.find((k) => k.label === 'Project Brain')!;
    expect(node.description).toBe('3 entries · 100 tokens');
    expect(String(node.tooltip)).toContain('Last Synchronized:');
    expect(node.command).toEqual({
      title: 'Sync Nexus Brain',
      command: 'nexus.brain.ingest',
      arguments: ['/workspace'],
    });
    expect(await ready.p.getChildren(node)).toEqual([]);

    const cold = await expand(client({ brain: brain({ initialized: false }) }), orch());
    const coldNode = cold.kids.find((k) => k.label === 'Project Brain')!;
    expect(coldNode.description).toBe('Uninitialized');

    const noDate = await expand(client({ brain: brain({ lastUpdated: undefined }) }), orch());
    expect(String(noDate.kids.find((k) => k.label === 'Project Brain')!.tooltip)).toContain('N/A');
  });

  it('skips each live section independently when the daemon is offline', async () => {
    const offline = client({
      activities: new Error('x'),
      tasks: new Error('y'),
      brain: new Error('z'),
    });
    const { kids } = await expand(offline, orch({ activePlan: plan() }));
    expect(labels(kids)).toEqual(['Active: First plan']);
    const partial = await expand(client({ activities: [], tasks: [] }), orch());
    expect(labels(partial.kids)).toEqual(['Project Brain']); // empty live sections are not shown
  });

  it('falls back to the placeholder when there is truly nothing to show', async () => {
    const { kids } = await expand(client({ brain: new Error('x') }), orch());
    expect(labels(kids)).toEqual(['No plans found']);
  });
});
