import { afterEach, describe, expect, it, vi } from 'vitest';
import type { AISession, NexusClient } from './nexusClient';

vi.mock('vscode', async () => (await import('./test/fakeVscode')).createFakeVscode());

import { MarkdownString, ThemeIcon } from './test/fakeVscode';
import { AISessionItem, AISessionsTreeProvider } from './aiSessionsTreeProvider';

const s = (o: Partial<AISession> = {}): AISession => ({
  id: 's1',
  agentName: 'Copilot',
  source: 'vscode',
  status: 'active',
  lastActivity: '2026-01-01T00:00:00Z',
  ...o,
});
const client = (sessions: AISession[] | Error) =>
  ({
    getAISessions: vi.fn(() =>
      sessions instanceof Error ? Promise.reject(sessions) : Promise.resolve(sessions),
    ),
  }) as unknown as NexusClient;
const color = (item: AISessionItem) => (item.iconPath as ThemeIcon).color?.id;

afterEach(() => vi.useRealTimers());

describe('AISessionItem', () => {
  it('shows name, source and the last two path segments (either separator)', () => {
    const posix = new AISessionItem(s({ projectPath: '/home/me/work/app' }));
    expect(posix.label).toBe('Copilot');
    expect(posix.description).toBe('[vscode] work/app');
    expect(new AISessionItem(s({ projectPath: 'C:\\code\\proj' })).description).toBe(
      '[vscode] code/proj',
    );
    expect(new AISessionItem(s()).description).toBe('[vscode]');
    expect(posix.contextValue).toBe('aiSession');
  });

  it.each<[Partial<AISession>, string]>([
    [{ status: 'active', delegatedToNexus: true }, 'charts.green'],
    [{ status: 'active' }, 'charts.yellow'],
    [{ status: 'idle' }, 'charts.orange'],
    [{ status: 'disconnected' }, 'disabledForeground'],
  ])('colours %o as %s', (o, expected) => {
    const item = new AISessionItem(s(o));
    expect(color(item)).toBe(expected);
    expect((item.iconPath as ThemeIcon).id).toBe('robot');
  });

  it('builds a tooltip with the optional details only when present', () => {
    const tip = (x: AISession) => (new AISessionItem(x).tooltip as MarkdownString).value;
    expect(tip(s())).toContain('**Copilot**');
    expect(tip(s())).not.toContain('Project');
    expect(tip(s())).not.toContain('Delegated');
    const full = tip(
      s({ projectPath: '/p', agentCapabilities: ['chat', 'edit'], delegatedToNexus: true }),
    );
    expect(full).toContain('Project: /p');
    expect(full).toContain('Capabilities: chat, edit');
    expect(full).toContain('✓ Delegated to Nexus');
    expect(tip(s({ lastActivity: '' }))).not.toContain('Last activity');
  });
});

describe('AISessionsTreeProvider', () => {
  it('sorts active, then idle, then the rest, newest first within each', async () => {
    const p = new AISessionsTreeProvider(
      client([
        s({ id: 'd', status: 'disconnected', lastActivity: '2026-01-05' }),
        s({ id: 'i', status: 'idle', lastActivity: '2026-01-04' }),
        s({ id: 'a-old', status: 'active', lastActivity: '2026-01-01' }),
        s({ id: 'a-new', status: 'active', lastActivity: '2026-01-03' }),
      ]),
    );
    expect((await p.getChildren()).map((i) => i.session.id)).toEqual(['a-new', 'a-old', 'i', 'd']);
  });

  it('copes with a missing lastActivity when sorting', async () => {
    const p = new AISessionsTreeProvider(
      client([s({ id: 'x', lastActivity: undefined as unknown as string }), s({ id: 'y' })]),
    );
    expect((await p.getChildren()).map((i) => i.session.id)).toEqual(['y', 'x']);
  });

  it('shows nothing, rather than failing, when the daemon is unreachable', async () => {
    expect(await new AISessionsTreeProvider(client(new Error('down'))).getChildren()).toEqual([]);
  });

  it('returns the item itself as its tree item', async () => {
    const p = new AISessionsTreeProvider(client([s()]));
    const [item] = await p.getChildren();
    expect(p.getTreeItem(item)).toBe(item);
  });

  it('refreshes on demand and on a polling interval until disposed', () => {
    vi.useFakeTimers();
    const p = new AISessionsTreeProvider(client([]));
    const changed = vi.fn();
    p.onDidChangeTreeData(changed);
    p.refresh();
    const sub = p.startPolling(1000);
    vi.advanceTimersByTime(2000);
    expect(changed).toHaveBeenCalledTimes(3);
    sub.dispose();
    sub.dispose(); // idempotent
    vi.advanceTimersByTime(5000);
    expect(changed).toHaveBeenCalledTimes(3);
  });

  it('dispose() stops polling and releases the emitter', () => {
    vi.useFakeTimers();
    const p = new AISessionsTreeProvider(client([]));
    const changed = vi.fn();
    p.onDidChangeTreeData(changed);
    p.startPolling(1000);
    p.dispose();
    vi.advanceTimersByTime(5000);
    expect(changed).not.toHaveBeenCalled();
    new AISessionsTreeProvider(client([])).dispose(); // without a timer
  });
});
