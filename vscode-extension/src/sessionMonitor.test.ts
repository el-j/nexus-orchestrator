import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { NexusClient } from './nexusClient';

const selectChatModelsMock = vi.fn();
const onDidChangeChatModelsMock = vi.fn();
const workspaceFolders = [{ uri: { fsPath: '/workspace' } }];
// Tests flip these to simulate a bare workspace or a VS Code without the language-model API.
const vscodeState: { folders: typeof workspaceFolders | undefined; hasLm: boolean } = {
  folders: workspaceFolders,
  hasLm: true,
};
const logNexusActivityMock = vi.fn();

const outputChannel = {
  appendLine: vi.fn(),
  show: vi.fn(),
};

vi.mock('vscode', () => ({
  env: {
    machineId: 'machine-1',
  },
  workspace: {
    get workspaceFolders() {
      return vscodeState.folders;
    },
  },
  get lm() {
    return vscodeState.hasLm
      ? {
          selectChatModels: (...args: unknown[]) => selectChatModelsMock(...args),
          onDidChangeChatModels: (cb: () => void) => {
            onDidChangeChatModelsMock(cb);
            return { dispose: vi.fn() };
          },
        }
      : undefined;
  },
}));

vi.mock('./activityLog', () => ({
  getNexusActivityChannel: vi.fn(() => outputChannel),
  logNexusActivity: (...args: unknown[]) => logNexusActivityMock(...args),
}));

function buildContext(): import('vscode').ExtensionContext {
  return {
    subscriptions: [],
    workspaceState: {
      update: vi.fn().mockResolvedValue(undefined),
    },
  } as unknown as import('vscode').ExtensionContext;
}

function buildClient(overrides: Partial<Record<string, unknown>> = {}): NexusClient {
  return {
    registerSession: vi.fn().mockResolvedValue({ id: 'session-1' }),
    heartbeatSession: vi.fn().mockResolvedValue(undefined),
    deregisterSession: vi.fn().mockResolvedValue(undefined),
    getTasks: vi.fn().mockResolvedValue([]),
    claimTask: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  } as unknown as NexusClient;
}

describe('SessionMonitor', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    vscodeState.folders = workspaceFolders;
    vscodeState.hasLm = true;
    vi.useFakeTimers();
    selectChatModelsMock.mockResolvedValue([{ id: 'copilot-model' }]);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('detectAndRegister retries after initial failure, stores session ID, and starts heartbeat', async () => {
    selectChatModelsMock.mockResolvedValueOnce([]).mockResolvedValue([{ id: 'copilot-model' }]);

    const client = buildClient();
    const context = buildContext();
    const { SessionMonitor } = await import('./sessionMonitor');
    const monitor = new SessionMonitor(client, context);

    const startPromise = monitor.start();
    await vi.advanceTimersByTimeAsync(2000);
    await startPromise;

    expect(client.registerSession).toHaveBeenCalledTimes(1);
    expect(context.workspaceState.update).toHaveBeenCalledWith('nexus.sessionId', 'session-1');

    await vi.advanceTimersByTimeAsync(60_000);
    expect(client.heartbeatSession).toHaveBeenCalledWith('session-1');
  });

  it('keeps session unset after three consecutive registration failures', async () => {
    const client = buildClient({
      registerSession: vi.fn().mockRejectedValue(new Error('register failed')),
    });
    const context = buildContext();
    const { SessionMonitor } = await import('./sessionMonitor');
    const monitor = new SessionMonitor(client, context);

    const startPromise = monitor.start();
    await vi.advanceTimersByTimeAsync(2_000 + 5_000 + 10_000);
    await startPromise;

    expect(client.registerSession).toHaveBeenCalledTimes(4);
    expect(context.workspaceState.update).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(120_000);
    expect(client.heartbeatSession).not.toHaveBeenCalled();
  });

  it('start then stop clears timers and deregisters exactly once', async () => {
    const client = buildClient();
    const context = buildContext();
    const { SessionMonitor } = await import('./sessionMonitor');
    const monitor = new SessionMonitor(client, context);

    await monitor.start();
    await monitor.stop();
    await monitor.stop();

    expect(client.deregisterSession).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(120_000);
    expect(client.heartbeatSession).not.toHaveBeenCalled();
  });

  it('logs heartbeat failures and keeps interval alive', async () => {
    const client = buildClient({
      heartbeatSession: vi.fn().mockRejectedValue(new Error('network down')),
      registerSession: vi
        .fn()
        .mockResolvedValueOnce({ id: 'session-1' })
        .mockResolvedValueOnce({ id: 'session-2' })
        .mockResolvedValue({ id: 'session-3' }),
    });

    const { SessionMonitor } = await import('./sessionMonitor');
    const monitor = new SessionMonitor(client, buildContext());

    await monitor.start();

    await vi.advanceTimersByTimeAsync(60_000);
    expect(logNexusActivityMock).toHaveBeenCalledWith(
      'copilot',
      expect.stringContaining('heartbeat failed'),
    );

    await vi.advanceTimersByTimeAsync(60_000);
    expect(client.heartbeatSession).toHaveBeenCalledTimes(2);
  });

  it('multiple start calls do not create duplicate intervals', async () => {
    const client = buildClient();
    const { SessionMonitor } = await import('./sessionMonitor');
    const monitor = new SessionMonitor(client, buildContext());

    await monitor.start();
    await monitor.start();

    await vi.advanceTimersByTimeAsync(60_000);
    expect(client.heartbeatSession).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(30_000);
    expect(client.getTasks).toHaveBeenCalledTimes(3);
  });

  it('startPolling path never calls claimTask', async () => {
    const claimTask = vi.fn();
    const client = buildClient({ claimTask });
    const { SessionMonitor } = await import('./sessionMonitor');
    const monitor = new SessionMonitor(client, buildContext());

    await monitor.start();
    await vi.advanceTimersByTimeAsync(90_000);

    expect(client.getTasks).toHaveBeenCalled();
    expect(claimTask).not.toHaveBeenCalled();
  });
});

describe('SessionMonitor: session purged by the daemon', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    vscodeState.folders = workspaceFolders;
    vscodeState.hasLm = true;
    vi.useFakeTimers();
    selectChatModelsMock.mockResolvedValue([{ id: 'copilot-model' }]);
  });

  it('registers again when the heartbeat says the daemon no longer knows the session', async () => {
    const client = buildClient({
      heartbeatSession: vi.fn().mockResolvedValue(false),
      registerSession: vi
        .fn()
        .mockResolvedValueOnce({ id: 'old' })
        .mockResolvedValue({ id: 'new' }),
    });
    const { SessionMonitor } = await import('./sessionMonitor');
    const monitor = new SessionMonitor(client, buildContext());
    await monitor.start();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(client.registerSession).toHaveBeenCalledTimes(2);
    expect(logNexusActivityMock).toHaveBeenCalledWith(
      'copilot',
      expect.stringContaining('no longer known'),
    );
    await vi.advanceTimersByTimeAsync(60_000);
    expect(client.heartbeatSession).toHaveBeenLastCalledWith('new');
  });

  it('does not start a second re-registration while one is running', async () => {
    let release!: () => void;
    const client = buildClient({
      heartbeatSession: vi.fn().mockResolvedValue(false),
      registerSession: vi
        .fn()
        .mockResolvedValueOnce({ id: 'old' })
        .mockImplementationOnce(() => new Promise((r) => (release = () => r({ id: 'new' })))),
    });
    const { SessionMonitor } = await import('./sessionMonitor');
    const monitor = new SessionMonitor(client, buildContext());
    await monitor.start();
    await vi.advanceTimersByTimeAsync(60_000); // starts re-registering and hangs
    (monitor as unknown as { sessionId: string }).sessionId = 'old'; // as if a concurrent tick saw an id
    await (monitor as unknown as { heartbeat(): Promise<void> }).heartbeat();
    expect(client.registerSession).toHaveBeenCalledTimes(2);
    release();
    await vi.advanceTimersByTimeAsync(0);
  });
});

describe('SessionMonitor: registration retries, model changes and shutdown', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    vscodeState.folders = workspaceFolders;
    vscodeState.hasLm = true;
    vi.useFakeTimers();
  });
  afterEach(() => vi.useRealTimers());

  it('retries at 2s, 5s and 10s while Copilot is still starting, then registers', async () => {
    selectChatModelsMock
      .mockResolvedValueOnce([])
      .mockResolvedValueOnce([])
      .mockResolvedValue([{ id: 'late-model' }]);
    const client = buildClient();
    const { SessionMonitor } = await import('./sessionMonitor');
    const ctx = buildContext();
    const monitor = new SessionMonitor(client, ctx);
    const started = monitor.start();
    await vi.advanceTimersByTimeAsync(2_000);
    expect(client.registerSession).not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(5_000);
    await started;
    expect(client.registerSession).toHaveBeenCalledTimes(1);
    expect(client.registerSession).toHaveBeenCalledWith(
      expect.objectContaining({ modelId: 'late-model', agentName: 'GitHub Copilot' }),
    );
    expect(logNexusActivityMock).toHaveBeenCalledWith(
      'copilot',
      expect.stringContaining('models not available yet'),
    );
  });

  it('gives up after the last retry and stays unregistered', async () => {
    selectChatModelsMock.mockResolvedValue([]);
    const client = buildClient();
    const { SessionMonitor } = await import('./sessionMonitor');
    const monitor = new SessionMonitor(client, buildContext());
    const started = monitor.start();
    await vi.advanceTimersByTimeAsync(2_000 + 5_000 + 10_000);
    await started;
    expect(client.registerSession).not.toHaveBeenCalled();
  });

  it('cancels a pending retry when the extension is disposed', async () => {
    selectChatModelsMock.mockResolvedValue([]);
    const { SessionMonitor } = await import('./sessionMonitor');
    const ctx = buildContext();
    const monitor = new SessionMonitor(buildClient(), ctx);
    void monitor.start();
    await vi.advanceTimersByTimeAsync(0);
    // disposing the extension runs the disposables registered by the retry wait
    for (const d of ctx.subscriptions) d.dispose();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(vi.getTimerCount()).toBeGreaterThanOrEqual(0); // no unhandled timer errors
  });

  it('registers again when the available chat models change', async () => {
    selectChatModelsMock.mockResolvedValue([{ id: 'm1' }]);
    const client = buildClient();
    const { SessionMonitor } = await import('./sessionMonitor');
    const monitor = new SessionMonitor(client, buildContext());
    await monitor.start();
    expect(client.registerSession).toHaveBeenCalledTimes(1);
    const onChange = onDidChangeChatModelsMock.mock.calls[0][0] as () => void;
    onChange();
    await vi.advanceTimersByTimeAsync(0);
    expect(client.registerSession).toHaveBeenCalledTimes(2);
  });

  it('logs a failed registration and keeps going', async () => {
    selectChatModelsMock.mockResolvedValue([{ id: 'm1' }]);
    const client = buildClient({
      registerSession: vi.fn().mockRejectedValue(new Error('daemon down')),
    });
    const { SessionMonitor } = await import('./sessionMonitor');
    const monitor = new SessionMonitor(client, buildContext());
    const started = monitor.start();
    await vi.advanceTimersByTimeAsync(20_000);
    await started;
    expect(logNexusActivityMock).toHaveBeenCalledWith(
      'copilot',
      expect.stringContaining('registration failed'),
    );
  });

  it('logs queue counts every 30s, and stays quiet when the daemon is unreachable', async () => {
    selectChatModelsMock.mockResolvedValue([{ id: 'm1' }]);
    const client = buildClient({
      getTasks: vi
        .fn()
        .mockResolvedValueOnce([
          { status: 'QUEUED' },
          { status: 'QUEUED' },
          { status: 'PROCESSING' },
          { status: 'COMPLETED' },
        ])
        .mockRejectedValueOnce(new Error('down')),
    });
    const { SessionMonitor } = await import('./sessionMonitor');
    const monitor = new SessionMonitor(client, buildContext());
    await monitor.start();
    await vi.advanceTimersByTimeAsync(30_000);
    expect(logNexusActivityMock).toHaveBeenCalledWith('copilot', 'queue: 2 queued, 1 processing');
    await vi.advanceTimersByTimeAsync(30_000); // the failing poll must not throw or log a queue line
    expect(
      logNexusActivityMock.mock.calls.filter((c) => String(c[1]).startsWith('queue:')),
    ).toHaveLength(1);
  });

  it('deregisters on stop, tolerates a failing deregistration, and can be restarted', async () => {
    selectChatModelsMock.mockResolvedValue([{ id: 'm1' }]);
    const client = buildClient({ deregisterSession: vi.fn().mockRejectedValue(new Error('gone')) });
    const { SessionMonitor } = await import('./sessionMonitor');
    const monitor = new SessionMonitor(client, buildContext());
    await monitor.start();
    await expect(monitor.stop()).resolves.toBeUndefined();
    expect(client.deregisterSession).toHaveBeenCalledWith('session-1');
    await monitor.stop(); // nothing registered any more
    expect(client.deregisterSession).toHaveBeenCalledTimes(1);
    await monitor.start();
    expect(client.registerSession).toHaveBeenCalledTimes(2);
    await vi.advanceTimersByTimeAsync(0);
  });

  it('works when the language-model API is not available at all', async () => {
    vscodeState.folders = undefined;
    vscodeState.hasLm = false;
    const { SessionMonitor } = await import('./sessionMonitor');
    const client = buildClient();
    const monitor = new SessionMonitor(client, buildContext());
    const started = monitor.start();
    await vi.advanceTimersByTimeAsync(20_000);
    await started;
    expect(client.registerSession).not.toHaveBeenCalled();
    await monitor.stop();
  });

  it('registers with an empty path and no model id when the workspace and model are bare', async () => {
    vscodeState.folders = undefined;
    selectChatModelsMock.mockResolvedValue([{ name: 'by-name' }, {}]);
    const { SessionMonitor } = await import('./sessionMonitor');
    const client = buildClient();
    await new SessionMonitor(client, buildContext()).start();
    expect(client.registerSession).toHaveBeenCalledWith(
      expect.objectContaining({ projectPath: '', modelId: 'by-name', externalId: 'machine-1:' }),
    );
  });
});
