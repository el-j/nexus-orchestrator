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
  usage?: unknown;
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
    ...(d.usage === undefined ? {} : { getUsage: vi.fn(() => Promise.resolve(d.usage)) }),
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

describe('NexusStatusBar limits, waiting tasks and usage', () => {
  it('counts WAITING tasks apart from the active ones', async () => {
    const r = await render({
      tasks: [task({ status: 'QUEUED' }), task({ id: 'w', status: 'WAITING' })],
    });
    expect(r.text).toContain('Q1');
    expect(r.text).toContain('W1');
    expect(r.tip).toContain('1 task(s) waiting for a provider limit to reset');
    const idle = await render({ tasks: [task({ id: 'w', status: 'WAITING' })] });
    expect(idle.text).toBe('$(watch) Nexus Q0 W1 M0 V0');
  });

  it('lists providers that cool down, with the reason, and says so when none is left', async () => {
    const until = new Date(NOW + 3600_000).toISOString();
    const cooling = (name: string): Provider => ({
      name,
      active: true,
      rateLimited: true,
      rateLimitedUntil: until,
      cooldownReason: `${name}: rate limited (429)`,
    });
    const some = await render({ providers: [cooling('Claude'), { name: 'Ollama', active: true }] });
    expect(some.tip).toMatch(
      /Claude: cooling down until \d{2}:\d{2} \(Claude: rate limited \(429\)\)/,
    );
    expect(some.text).not.toContain('all providers');
    const all = await render({ providers: [cooling('Claude')] });
    expect(all.text).toBe('$(watch) Nexus: every provider is cooling down');
    expect(all.tip).toContain('Claude: cooling down');
  });

  it('flags a plan that is nearly used up and ignores a plan that reports no share', async () => {
    const providers: Provider[] = [
      {
        name: 'Claude',
        active: true,
        quota: {
          window: 'five_hour',
          status: 'allowed',
          utilization: 0.93,
          resetsAt: new Date(NOW + 3600_000).toISOString(),
        },
      },
      {
        name: 'Copilot',
        active: true,
        quota: { window: 'month', status: 'allowed', utilization: 0.4 },
      },
      {
        name: 'Fresh',
        active: true,
        quota: { window: 'five_hour', status: 'allowed', utilization: -1 },
      },
      {
        name: 'NoReset',
        active: true,
        quota: { window: 'day', status: 'allowed', utilization: 0.95 },
      },
    ];
    const r = await render({ providers });
    expect(r.tip).toMatch(/Claude: plan 93% used \(resets \d{2}:\d{2}\)/);
    expect(r.tip).toContain('NoReset: plan 95% used');
    expect(r.tip).not.toContain('Copilot 40%');
    expect(r.tip).not.toContain('Fresh');
  });

  it('shows the tokens of the last 24 hours, and survives a daemon without the usage ledger', async () => {
    const usage = { totals: { calls: 3, inputTokens: 12345, outputTokens: 678 }, usage: [] };
    expect((await render({ usage })).tip).toContain('Tokens (24h): 12.3k in · 678 out');
    expect(
      (
        await render({
          usage: { totals: { calls: 0, inputTokens: 0, outputTokens: 0 }, usage: [] },
        })
      ).tip,
    ).not.toContain('Tokens (24h)');
    expect((await render({})).tip).not.toContain('Tokens (24h)');
  });
});
