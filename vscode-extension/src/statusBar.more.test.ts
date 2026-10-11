import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { AIActivity, AISession, NexusClient, Provider, Task } from './nexusClient';

const item = vi.hoisted(() => ({
  text: '',
  tooltip: '' as string,
  command: undefined as string | undefined,
  show: vi.fn(),
  dispose: vi.fn(),
}));
const snapshot = vi.hoisted(() => ({ lastQueuedTaskId: undefined as string | undefined }));
vi.mock('vscode', () => ({
  StatusBarAlignment: { Left: 1 },
  window: { createStatusBarItem: vi.fn(() => item) },
}));
vi.mock('./activityLog', () => ({ getActivitySnapshot: () => snapshot }));

import { NexusStatusBar } from './statusBar';

const NOW = new Date('2026-06-01T12:00:00Z').getTime();
const ago = (ms: number) => new Date(NOW - ms).toISOString();
const task = (o: Partial<Task> = {}): Task => ({
  id: 't',
  projectPath: '/p',
  targetFile: '',
  instruction: '',
  contextFiles: [],
  status: 'QUEUED',
  createdAt: '',
  updatedAt: ago(1000),
  ...o,
});
const sess = (o: Partial<AISession> = {}): AISession => ({
  id: 's',
  agentName: 'a',
  source: 'mcp',
  status: 'active',
  lastActivity: '',
  ...o,
});
const act = (o: Partial<AIActivity> = {}): AIActivity => ({
  id: 'a',
  agentName: 'claude',
  activityType: 'message',
  summary: '',
  timestamp: ago(1000),
  ...o,
});

interface Data {
  alive?: boolean;
  providers?: Provider[] | Error;
  tasks?: Task[] | Error;
  sessions?: AISession[] | Error;
  activities?: AIActivity[] | Error;
}
function clientFor(d: Data): NexusClient {
  const val = <T>(v: T | Error | undefined, dflt: T) =>
    v instanceof Error ? Promise.reject(v) : Promise.resolve(v ?? dflt);
  return {
    health: vi.fn().mockResolvedValue(d.alive ?? true),
    getProviders: vi.fn(() => val(d.providers, [])),
    getTasks: vi.fn(() => val(d.tasks, [])),
    getAISessions: vi.fn(() => val(d.sessions, [])),
    getActivities: vi.fn(() => val(d.activities, [])),
  } as unknown as NexusClient;
}
async function render(d: Data): Promise<{ text: string; tip: string }> {
  const bar = new NexusStatusBar(clientFor(d));
  await bar.update();
  return { text: item.text, tip: String(item.tooltip) };
}

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(NOW);
  item.text = '';
  item.tooltip = '';
  item.show.mockClear();
  item.dispose.mockClear();
  snapshot.lastQueuedTaskId = undefined;
  vi.spyOn(console, 'warn').mockImplementation(() => {});
});
afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe('NexusStatusBar.update', () => {
  it('shows the item and wires its click command', () => {
    new NexusStatusBar(clientFor({}));
    expect(item.show).toHaveBeenCalled();
    expect(item.command).toBe('nexus.statusBarAction');
  });

  it('shows offline when the daemon does not answer', async () => {
    const r = await render({ alive: false });
    expect(r.text).toBe('$(warning) Nexus: offline');
    expect(r.tip).toBe('Click to open actions');
  });

  it('shows offline when the refresh itself throws', async () => {
    const client = clientFor({});
    (client.health as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('boom'));
    const bar = new NexusStatusBar(client);
    await bar.update();
    expect(item.text).toBe('$(warning) Nexus: offline');
  });

  it('shows the idle state with session counts per source', async () => {
    const r = await render({
      sessions: [
        sess({ source: 'mcp' }),
        sess({ source: 'vscode' }),
        sess({ source: 'http' }),
        sess({ source: 'mcp', status: 'idle' }),
      ],
    });
    expect(r.text).toBe('$(zap) Nexus Q0 M1 V1');
    expect(r.tip).toContain('MCP sessions: 1');
    expect(r.tip).toContain('Copilot direct sessions: 1');
    expect(r.tip).toContain('HTTP sessions: 1');
    expect(r.tip).toContain('AI sessions total: 3');
  });

  it('spins while tasks are queued or processing', async () => {
    const r = await render({
      tasks: [task(), task({ status: 'PROCESSING' }), task({ status: 'COMPLETED' })],
    });
    expect(r.text).toBe('$(sync~spin) Nexus Q2 M0 V0');
    expect(r.tip).toContain('Nexus queue: 2 active task(s)');
  });

  it('shows active agents and how many are generating (agents win over the queue)', async () => {
    const r = await render({
      tasks: [task()],
      activities: [
        act(),
        act({ agentName: 'copilot', activityType: 'generation' }),
        act({ agentName: 'old', timestamp: ago(10 * 60_000) }),
      ],
    });
    expect(r.text).toBe('🤖 2 agent(s) · 1 generating');
    expect(r.tip).toContain('2 AI agents active in last 5m');
  });

  it('shows failures from the last hour first, ignoring older ones', async () => {
    const r = await render({
      tasks: [task({ status: 'FAILED', updatedAt: ago(60_000) })],
      activities: [act()],
    });
    expect(r.text).toBe('$(error) Nexus Q0 M0 V0');
    const old = await render({ tasks: [task({ status: 'FAILED', updatedAt: ago(2 * 3600_000) })] });
    expect(old.text).toBe('$(zap) Nexus Q0 M0 V0');
  });

  it('flags a partial refresh: degraded, or a ~ marker while busy', async () => {
    const degraded = await render({ providers: new Error('x') });
    expect(degraded.text).toBe('$(warning) Nexus: degraded');
    expect(degraded.tip).toContain('Some daemon reads failed');
    const busy = await render({ providers: new Error('x'), tasks: [task()] });
    expect(busy.text).toBe('$(sync~spin) Nexus Q1 M0 V0 ~');
    const tasksDown = await render({ tasks: new Error('y') });
    expect(tasksDown.text).toBe('$(warning) Nexus: degraded');
    expect(console.warn).toHaveBeenCalledWith('statusBar: degraded — failed to fetch:', 'tasks');
  });

  it('copes with failing session and activity reads', async () => {
    const r = await render({ sessions: new Error('s'), activities: new Error('a') });
    expect(r.text).toBe('$(zap) Nexus Q0 M0 V0');
  });

  it('mentions the last task queued by the extension, shortened', async () => {
    snapshot.lastQueuedTaskId = 'abcd-1234-ef56-7890';
    const r = await render({});
    expect(r.tip).toContain('Last queued by extension: #abcd1234');
    expect(
      (
        await render({
          providers: [
            { name: 'x', active: true },
            { name: 'y', active: false },
          ],
        })
      ).tip,
    ).toContain('Providers active: 1');
  });
});

describe('NexusStatusBar polling', () => {
  it('updates immediately, then on the interval, until disposed', async () => {
    const client = clientFor({});
    const bar = new NexusStatusBar(client);
    const sub = bar.startPolling(10_000);
    await vi.advanceTimersByTimeAsync(0);
    expect(client.health).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(20_000);
    expect(client.health).toHaveBeenCalledTimes(3);
    sub.dispose();
    expect(item.dispose).toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(client.health).toHaveBeenCalledTimes(3);
  });

  it('defaults to a 10s interval and never stacks pollers', async () => {
    const client = clientFor({});
    const bar = new NexusStatusBar(client);
    bar.startPolling();
    bar.startPolling();
    await vi.advanceTimersByTimeAsync(0);
    (client.health as ReturnType<typeof vi.fn>).mockClear();
    await vi.advanceTimersByTimeAsync(10_000);
    expect(client.health).toHaveBeenCalledTimes(1);
    bar.dispose();
    bar.dispose(); // idempotent
  });
});
