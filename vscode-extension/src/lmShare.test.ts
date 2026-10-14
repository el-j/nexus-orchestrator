import { beforeEach, describe, expect, it, vi } from 'vitest';

const h = vi.hoisted(() => ({
  config: {} as Record<string, unknown>,
  bridge: { baseUrl: 'http://127.0.0.1:1/v1', token: 't', close: vi.fn(async () => {}) },
  start: vi.fn(),
  inSync: vi.fn(),
  sync: vi.fn(),
  drop: vi.fn(),
  log: vi.fn(),
  modelListeners: [] as Array<() => void>,
  configListeners: [] as Array<(e: { affectsConfiguration(s: string): boolean }) => void>,
  hasLm: true,
}));

vi.mock('vscode', () => ({
  workspace: {
    getConfiguration: () => ({
      get: (k: string, d: unknown) => (k in h.config ? h.config[k] : d),
    }),
    onDidChangeConfiguration: (cb: never) => {
      h.configListeners.push(cb);
      return { dispose() {} };
    },
  },
  get lm() {
    return h.hasLm
      ? { onDidChangeChatModels: (cb: () => void) => (h.modelListeners.push(cb), { dispose() {} }) }
      : undefined;
  },
}));
vi.mock('./lmBridge', () => ({ startLmBridge: h.start }));
vi.mock('./lmProviders', () => ({
  isInSync: h.inSync,
  syncLmProviders: h.sync,
  dropLmProviders: h.drop,
}));
vi.mock('./activityLog', () => ({ logNexusActivity: h.log }));

import { LmShare } from './lmShare';
import type { NexusClient } from './nexusClient';

describe('LmShare', () => {
  let health: ReturnType<typeof vi.fn>;
  let share: LmShare;
  beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
    h.config = {};
    h.modelListeners.length = 0;
    h.configListeners.length = 0;
    h.hasLm = true;
    h.bridge.close.mockResolvedValue(undefined);
    h.start.mockResolvedValue(h.bridge);
    h.inSync.mockResolvedValue(false);
    h.sync.mockResolvedValue(['a', 'b']);
    h.drop.mockResolvedValue(undefined);
    health = vi.fn(async () => true);
    share = new LmShare({ health } as unknown as NexusClient);
  });

  it('does nothing while the setting is off', async () => {
    await share.refresh();
    expect(h.start).not.toHaveBeenCalled();
    expect(h.sync).not.toHaveBeenCalled();
  });

  it('starts the bridge and registers the models when enabled', async () => {
    h.config = { enabled: true, vendors: ['copilot'] };
    await share.refresh();
    expect(h.sync).toHaveBeenCalledWith(expect.anything(), h.bridge, ['copilot']);
    expect(h.log).toHaveBeenCalledWith('lm-bridge', '2 editor model(s) registered with the daemon');
    share.dispose();
  });

  it('starts only one bridge across refreshes', async () => {
    h.config = { enabled: true };
    await share.refresh();
    await share.refresh();
    expect(h.start).toHaveBeenCalledTimes(1);
    share.dispose();
  });

  it('skips registration when the daemon is already in sync', async () => {
    h.config = { enabled: true };
    h.inSync.mockResolvedValue(true);
    await share.refresh();
    expect(h.sync).not.toHaveBeenCalled();
    share.dispose();
  });

  it('waits when the daemon is not reachable', async () => {
    h.config = { enabled: true };
    health.mockResolvedValue(false);
    await share.refresh();
    expect(h.start).toHaveBeenCalled();
    expect(h.sync).not.toHaveBeenCalled();
    share.dispose();
  });

  it('logs a failure and carries on', async () => {
    h.config = { enabled: true };
    h.sync.mockRejectedValueOnce(new Error('daemon said no'));
    await share.refresh();
    expect(h.log).toHaveBeenCalledWith('lm-bridge', 'daemon said no');
    h.sync.mockRejectedValueOnce('odd');
    await share.refresh();
    expect(h.log).toHaveBeenCalledWith('lm-bridge', 'odd');
    share.dispose();
  });

  it('ignores a refresh while another one runs', async () => {
    h.config = { enabled: true };
    let release: () => void = () => {};
    health.mockImplementation(() => new Promise((r) => (release = () => r(true))));
    const first = share.refresh();
    await share.refresh();
    release();
    await first;
    expect(h.start).toHaveBeenCalledTimes(1);
    share.dispose();
  });

  it('re-checks on a timer', async () => {
    h.config = { enabled: true };
    await share.refresh();
    h.inSync.mockClear();
    await vi.advanceTimersByTimeAsync(30_000);
    expect(h.inSync).toHaveBeenCalledTimes(1);
    share.dispose();
  });

  it('stops the bridge and drops the providers when switched off', async () => {
    h.config = { enabled: true };
    await share.refresh();
    h.config = { enabled: false };
    await share.refresh();
    expect(h.bridge.close).toHaveBeenCalled();
    expect(h.drop).toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(h.inSync).toHaveBeenCalledTimes(1); // the timer is gone
  });

  it('stop is a no-op when nothing runs, and survives a failing cleanup', async () => {
    await share.stop();
    expect(h.drop).not.toHaveBeenCalled();
    h.config = { enabled: true };
    await share.refresh();
    h.drop.mockRejectedValue(new Error('daemon gone'));
    await expect(share.stop()).resolves.toBeUndefined();
  });

  describe('register', () => {
    const context = () => ({ subscriptions: [] as unknown[] });
    it('follows model-list and setting changes and is disposed with the context', async () => {
      h.config = { enabled: true };
      const ctx = context();
      share.register(ctx as never);
      await vi.advanceTimersByTimeAsync(0);
      expect(h.start).toHaveBeenCalledTimes(1);
      h.sync.mockClear();
      h.modelListeners[0]!();
      await vi.advanceTimersByTimeAsync(0);
      expect(h.sync).toHaveBeenCalledTimes(1);
      h.sync.mockClear();
      h.configListeners[0]!({ affectsConfiguration: (s) => s === 'nexus.lmBridge' });
      await vi.advanceTimersByTimeAsync(0);
      expect(h.sync).toHaveBeenCalledTimes(1);
      h.sync.mockClear();
      h.configListeners[0]!({ affectsConfiguration: () => false });
      await vi.advanceTimersByTimeAsync(0);
      expect(h.sync).not.toHaveBeenCalled();
      expect(ctx.subscriptions).toContain(share);
      share.dispose();
    });
    it('works in an editor without the language-model API', () => {
      h.hasLm = false;
      const ctx = context();
      share.register(ctx as never);
      expect(h.modelListeners).toHaveLength(0);
    });
  });
});
