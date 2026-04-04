import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { NexusClient } from './nexusClient';

const selectChatModelsMock = vi.fn();
const onDidChangeChatModelsMock = vi.fn();
const workspaceFolders = [{ uri: { fsPath: '/workspace' } }];
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
    workspaceFolders,
  },
  lm: {
    selectChatModels: (...args: unknown[]) => selectChatModelsMock(...args),
    onDidChangeChatModels: (cb: () => void) => {
      onDidChangeChatModelsMock(cb);
      return { dispose: vi.fn() };
    },
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
