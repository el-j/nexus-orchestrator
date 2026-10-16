import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest';
import type { NexusClient, Task, TaskStatus } from './nexusClient';

vi.mock('vscode', async () => (await import('./test/fakeVscode')).createFakeVscode());
vi.mock('./activityLog', () => ({
  getKnownTaskSource: (id: string) => (id === 'mine' ? 'vscode queue' : undefined),
}));

import { fakeState, resetFakeState, ThemeIcon, MarkdownString, Uri } from './test/fakeVscode';
import { TaskItem, TaskQueueProvider } from './taskQueueProvider';

const task = (o: Partial<Task> = {}): Task => ({
  id: 'abcd-1234-ef56',
  projectPath: '/workspace',
  targetFile: 'a.ts',
  instruction: 'Do something',
  contextFiles: [],
  status: 'QUEUED',
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
  ...o,
});
const client = (tasks: Task[] | Error, alive = false) =>
  ({
    health: vi.fn(() => Promise.resolve(alive)),
    getAllTasks: vi.fn(() =>
      tasks instanceof Error ? Promise.reject(tasks) : Promise.resolve(tasks),
    ),
  }) as unknown as NexusClient;

beforeEach(() => resetFakeState());
afterEach(() => vi.useRealTimers());

describe('TaskItem run by', () => {
  it('says whether Nexus ran the task or an agent only set its status', () => {
    const tip = (o: Partial<Task>) => (new TaskItem(task(o)).tooltip as { value: string }).value;
    expect(tip({ status: 'COMPLETED', ranBy: 'Ollama' })).toContain('**Run by:** Nexus on Ollama');
    expect(tip({ status: 'COMPLETED' })).toContain('not run by Nexus — status set by an agent');
    expect(tip({ status: 'PROCESSING' })).toContain('not run by Nexus');
    expect(tip({ status: 'FAILED' })).toContain('not run by Nexus');
    expect(tip({ status: 'QUEUED' })).toContain('**Run by:** —');
  });
});

describe('TaskItem', () => {
  it('labels with emoji, short id and a truncated instruction', () => {
    const short = new TaskItem(task());
    expect(short.label).toBe('🕐 #abcd1234 — Do something');
    expect(new TaskItem(task({ instruction: 'x'.repeat(40) })).label).toBe(
      `🕐 #abcd1234 — ${'x'.repeat(25)}…`,
    );
    expect(short.contextValue).toBe('nexusTask');
  });

  it.each<[TaskStatus, string, string]>([
    ['QUEUED', '🕐', 'clock'],
    ['PROCESSING', '🕐', 'loading~spin'],
    ['COMPLETED', '✅', 'pass'],
    ['FAILED', '❌', 'error'],
    ['CANCELLED', '⛔', 'circle-slash'],
    ['TOO_LARGE', '📏', 'file-binary'],
    ['WAITING', '⏳', 'watch'],
    ['NO_PROVIDER', '🔌', 'plug'],
    ['DRAFT', '📝', 'edit'],
    ['BACKLOG', '📋', 'checklist'],
    ['SURPRISE' as TaskStatus, '❓', 'question'],
  ])('shows %s with %s and the %s icon', (status, emoji, icon) => {
    const item = new TaskItem(task({ status }));
    expect(String(item.label)).toContain(emoji);
    expect((item.iconPath as ThemeIcon).id).toBe(icon);
  });

  it('describes where the task came from and how it is routed', () => {
    expect(new TaskItem(task()).description).toBe('');
    expect(
      new TaskItem(task({ id: 'mine', providerHint: 'Ollama', modelId: 'llama3' })).description,
    ).toBe('vscode queue / Ollama / llama3');
  });

  it('builds a tooltip with every field, placeholders for blanks, and sanitised logs', () => {
    const tip = (t: Task) => (new TaskItem(t).tooltip as MarkdownString).value;
    expect(tip(task({ targetFile: '' }))).toContain('**Target:** —');
    expect(tip(task())).toContain('**Source:** unknown');
    expect(tip(task({ id: 'mine' }))).toContain('**Source:** vscode queue');
    expect(tip(task())).not.toContain('Logs');
    const logs = tip(task({ logs: '<script>&"x"' }));
    expect(logs).toContain('**Logs:**');
    expect(logs).not.toMatch(/[<>&]/);
    const long = tip(task({ logs: 'y'.repeat(600) }));
    expect(long).toContain('y'.repeat(500) + '...');
    expect(long).not.toContain('y'.repeat(501));
  });
});

describe('TaskQueueProvider', () => {
  it('lists active tasks oldest-first, then finished ones newest-first', async () => {
    const p = new TaskQueueProvider(
      client([
        task({ id: 'old-done', status: 'COMPLETED', updatedAt: '2026-01-01T00:00:00Z' }),
        task({ id: 'new-done', status: 'FAILED', updatedAt: '2026-01-03T00:00:00Z' }),
        task({ id: 'late-q', createdAt: '2026-01-02T00:00:00Z' }),
        task({ id: 'early-q', status: 'PROCESSING', createdAt: '2026-01-01T00:00:00Z' }),
      ]),
    );
    const items = await p.getChildren();
    expect(items.map((i) => (i as TaskItem).task.id)).toEqual([
      'early-q',
      'late-q',
      'new-done',
      'old-done',
    ]);
  });

  it('shows drafts and backlog items instead of failing (regression)', async () => {
    const p = new TaskQueueProvider(
      client([task({ status: 'DRAFT' }), task({ id: 'b', status: 'BACKLOG' })]),
    );
    expect(await p.getChildren()).toHaveLength(2);
  });

  it('scopes to the workspace folder by default, and shows all projects when toggled', async () => {
    const tasks = [task({ id: 'in' }), task({ id: 'out', projectPath: '/other' })];
    const p = new TaskQueueProvider(client(tasks));
    expect(p.getScopeToWorkspace()).toBe(true);
    expect((await p.getChildren()).map((i) => (i as TaskItem).task.id)).toEqual(['in']);
    p.setScopeToWorkspace(false);
    expect(p.getScopeToWorkspace()).toBe(false);
    expect(await p.getChildren()).toHaveLength(2);
  });

  it('honours the scopeToWorkspace setting, and shows everything without a folder', async () => {
    fakeState.config['scopeToWorkspace'] = false;
    const tasks = [task({ id: 'in' }), task({ id: 'out', projectPath: '/other' })];
    expect(await new TaskQueueProvider(client(tasks)).getChildren()).toHaveLength(2);
    resetFakeState();
    fakeState.folders = undefined;
    expect(await new TaskQueueProvider(client(tasks)).getChildren()).toHaveLength(2);
    fakeState.folders = [{ uri: Uri.file('/workspace'), name: 'w' }];
  });

  it('shows a single offline item when the daemon cannot be reached', async () => {
    const p = new TaskQueueProvider(client(new Error('down')));
    const [item] = await p.getChildren();
    expect(item.label).toContain('daemon offline');
    expect(item.contextValue).toBe('nexusDaemonOffline');
    expect((item.iconPath as ThemeIcon).id).toBe('debug-disconnect');
    expect(String(item.tooltip)).toContain('not running');
  });

  it('says the daemon has an error, not that it is offline, when it answers health but fails the list', async () => {
    const p = new TaskQueueProvider(client(new Error('HTTP 500 — internal server error'), true));
    const [item] = await p.getChildren();
    expect(item.label).toContain('daemon error');
    expect(item.contextValue).toBe('nexusDaemonError');
    expect(String(item.tooltip)).toContain('HTTP 500');
    const p2 = new TaskQueueProvider(client('plain text' as unknown as Error, true));
    (p2 as unknown as { client: { getAllTasks: () => Promise<never> } }).client.getAllTasks = () =>
      Promise.reject('plain text');
    expect(String((await p2.getChildren())[0].tooltip)).toContain('plain text');
  });

  it('returns the element itself as the tree item', async () => {
    const p = new TaskQueueProvider(client([task()]));
    const [item] = await p.getChildren();
    expect(p.getTreeItem(item)).toBe(item);
  });

  it('fires a change event on refresh and when the scope changes', () => {
    const p = new TaskQueueProvider(client([]));
    const changed = vi.fn();
    p.onDidChangeTreeData(changed);
    p.refresh();
    p.setScopeToWorkspace(false);
    expect(changed).toHaveBeenCalledTimes(2);
  });

  it('polls on an interval until the returned disposable is disposed', () => {
    vi.useFakeTimers();
    const p = new TaskQueueProvider(client([]));
    const changed = vi.fn();
    p.onDidChangeTreeData(changed);
    const sub = p.startPolling(1000);
    vi.advanceTimersByTime(3000);
    expect(changed).toHaveBeenCalledTimes(3);
    sub.dispose();
    vi.advanceTimersByTime(5000);
    expect(changed).toHaveBeenCalledTimes(3);
    p.startPolling().dispose(); // default interval
  });
});
