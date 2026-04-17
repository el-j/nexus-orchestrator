import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { NexusClient, AISession } from './nexusClient';

// ── Minimal vscode stubs ──────────────────────────────────────────────────────

class ThemeColor {
  constructor(public readonly id: string) {}
}

class ThemeIcon {
  constructor(
    public readonly id: string,
    public readonly color?: ThemeColor,
  ) {}
}

class MarkdownString {
  constructor(public readonly value: string) {}
}

class TreeItem {
  description?: string;
  tooltip?: unknown;
  iconPath?: unknown;
  contextValue?: string;
  constructor(
    public readonly label: string,
    public readonly collapsibleState?: number,
  ) {}
}

class EventEmitter<T = void> {
  readonly event = vi.fn();
  fire = vi.fn<(value?: T) => void>();
  dispose = vi.fn();
}

class Disposable {
  constructor(private readonly fn: () => void) {}
  dispose() {
    this.fn();
  }
}

vi.mock('vscode', () => ({
  TreeItem,
  TreeItemCollapsibleState: { None: 0 },
  ThemeIcon,
  ThemeColor,
  MarkdownString,
  EventEmitter,
  Disposable,
}));

// ── Helpers ───────────────────────────────────────────────────────────────────

function makeSession(overrides: Partial<AISession> = {}): AISession {
  return {
    id: 's-1',
    agentName: 'TestAgent',
    source: 'mcp',
    status: 'active',
    lastActivity: '2026-04-17T10:00:00Z',
    ...overrides,
  };
}

function buildClient(sessions: AISession[] = []): NexusClient {
  return {
    getAISessions: vi.fn().mockResolvedValue(sessions),
  } as unknown as NexusClient;
}

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('AISessionsTreeProvider', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('getChildren returns AISessionItem for each session', async () => {
    const { AISessionsTreeProvider } = await import('./aiSessionsTreeProvider');
    const client = buildClient([
      makeSession({ id: 's-1' }),
      makeSession({ id: 's-2', agentName: 'Claude' }),
    ]);
    const provider = new AISessionsTreeProvider(client);

    const items = await provider.getChildren();

    expect(items).toHaveLength(2);
    expect(items[0].session.id).toBe('s-1');
    expect(items[1].session.agentName).toBe('Claude');
  });

  it('getChildren returns empty array when API throws', async () => {
    const { AISessionsTreeProvider } = await import('./aiSessionsTreeProvider');
    const client = {
      getAISessions: vi.fn().mockRejectedValue(new Error('offline')),
    } as unknown as NexusClient;
    const provider = new AISessionsTreeProvider(client);

    const items = await provider.getChildren();

    expect(items).toHaveLength(0);
  });

  it('sorts sessions: active first, then idle, then others; newest first within group', async () => {
    const { AISessionsTreeProvider } = await import('./aiSessionsTreeProvider');
    const sessions: AISession[] = [
      makeSession({ id: 'idle-old', status: 'idle', lastActivity: '2026-04-17T08:00:00Z' }),
      makeSession({ id: 'active-2', status: 'active', lastActivity: '2026-04-17T09:00:00Z' }),
      makeSession({ id: 'active-1', status: 'active', lastActivity: '2026-04-17T10:00:00Z' }),
      makeSession({
        id: 'disconnected',
        status: 'disconnected',
        lastActivity: '2026-04-17T07:00:00Z',
      }),
    ];
    const client = buildClient(sessions);
    const provider = new AISessionsTreeProvider(client);

    const items = await provider.getChildren();

    expect(items[0].session.id).toBe('active-1');
    expect(items[1].session.id).toBe('active-2');
    expect(items[2].session.id).toBe('idle-old');
    expect(items[3].session.id).toBe('disconnected');
  });

  it('getTreeItem returns the AISessionItem unchanged', async () => {
    const { AISessionsTreeProvider, AISessionItem } = await import('./aiSessionsTreeProvider');
    const client = buildClient([]);
    const provider = new AISessionsTreeProvider(client);
    const item = new AISessionItem(makeSession());

    expect(provider.getTreeItem(item)).toBe(item);
  });

  it('startPolling sets up an interval and returns a Disposable', async () => {
    vi.useFakeTimers();
    const { AISessionsTreeProvider } = await import('./aiSessionsTreeProvider');
    const client = buildClient([makeSession()]);
    const provider = new AISessionsTreeProvider(client);

    const disposable = provider.startPolling(500);
    await vi.advanceTimersByTimeAsync(500);

    // The Disposable should be callable without throwing
    expect(() => disposable.dispose()).not.toThrow();

    provider.dispose();
    vi.useRealTimers();
  });

  it('dispose clears the polling timer and disposes the EventEmitter', async () => {
    vi.useFakeTimers();
    const clearSpy = vi.spyOn(globalThis, 'clearInterval');
    const { AISessionsTreeProvider } = await import('./aiSessionsTreeProvider');
    const client = buildClient([]);
    const provider = new AISessionsTreeProvider(client);

    provider.startPolling(1000);
    provider.dispose();

    expect(clearSpy).toHaveBeenCalled();

    vi.useRealTimers();
  });
});

describe('AISessionItem', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('label is the agentName', async () => {
    const { AISessionItem } = await import('./aiSessionsTreeProvider');
    const item = new AISessionItem(makeSession({ agentName: 'Claude Desktop' }));

    expect(item.label).toBe('Claude Desktop');
  });

  it('description includes source in brackets', async () => {
    const { AISessionItem } = await import('./aiSessionsTreeProvider');
    const item = new AISessionItem(makeSession({ source: 'mcp' }));

    expect(item.description).toContain('[mcp]');
  });

  it('description includes the last 2 path segments when projectPath is set', async () => {
    const { AISessionItem } = await import('./aiSessionsTreeProvider');
    const item = new AISessionItem(makeSession({ projectPath: '/home/user/projects/my-app' }));

    expect(item.description).toContain('projects');
    expect(item.description).toContain('my-app');
  });

  it('active+delegated session gets a green robot icon', async () => {
    const { AISessionItem } = await import('./aiSessionsTreeProvider');
    const item = new AISessionItem(makeSession({ status: 'active', delegatedToNexus: true }));

    const icon = item.iconPath as ThemeIcon;
    expect(icon.id).toBe('robot');
    expect(icon.color?.id).toContain('green');
  });

  it('active session gets a yellow robot icon', async () => {
    const { AISessionItem } = await import('./aiSessionsTreeProvider');
    const item = new AISessionItem(makeSession({ status: 'active', delegatedToNexus: false }));

    const icon = item.iconPath as ThemeIcon;
    expect(icon.id).toBe('robot');
    expect(icon.color?.id).toContain('yellow');
  });

  it('tooltip is a MarkdownString containing session ID and status', async () => {
    const { AISessionItem } = await import('./aiSessionsTreeProvider');
    const item = new AISessionItem(makeSession({ id: 'session-xyz', status: 'idle' }));

    const tooltip = item.tooltip as MarkdownString;
    expect(tooltip.value).toContain('session-xyz');
    expect(tooltip.value).toContain('idle');
  });
});
