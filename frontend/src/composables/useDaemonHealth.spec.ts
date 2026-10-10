import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { flushPromises } from '@vue/test-utils';
import { withSetup } from '../test/withSetup';

vi.mock('./useServerUrl', () => ({ resolveServerUrl: vi.fn().mockResolvedValue('http://daemon') }));
import { useDaemonHealth } from './useDaemonHealth';

const fetchMock = vi.fn();

describe('useDaemonHealth', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    fetchMock.mockReset();
    vi.stubGlobal('fetch', fetchMock);
  });
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  const tick = async (ms: number) => {
    await vi.advanceTimersByTimeAsync(ms);
    await flushPromises();
  };

  it('checks immediately on mount and reports connected when /api/health is ok', async () => {
    fetchMock.mockResolvedValue({ ok: true });
    const { result, unmount } = withSetup(() => useDaemonHealth());
    await flushPromises();
    expect(fetchMock).toHaveBeenCalledWith(
      'http://daemon/api/health',
      expect.objectContaining({ signal: expect.anything() }),
    );
    expect(result.connected.value).toBe(true);
    expect(result.lastChecked.value).toBeInstanceOf(Date);
    unmount();
  });

  it('polls every 10 seconds and stops after unmount', async () => {
    fetchMock.mockResolvedValue({ ok: true });
    const { unmount } = withSetup(() => useDaemonHealth());
    await flushPromises();
    await tick(10_000);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    unmount();
    await tick(30_000);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it('tolerates two failures and only flips to disconnected on the third', async () => {
    fetchMock.mockResolvedValueOnce({ ok: true });
    const { result, unmount } = withSetup(() => useDaemonHealth());
    await flushPromises();
    expect(result.connected.value).toBe(true);

    fetchMock.mockRejectedValue(new Error('down'));
    await tick(10_000);
    await tick(10_000);
    expect(result.connected.value).toBe(true);
    await tick(10_000);
    expect(result.connected.value).toBe(false);
    unmount();
  });

  it('counts non-2xx answers as failures and recovers on the next success', async () => {
    fetchMock.mockResolvedValue({ ok: false });
    const { result, unmount } = withSetup(() => useDaemonHealth());
    await flushPromises();
    await tick(10_000);
    await tick(10_000);
    expect(result.connected.value).toBe(false);
    fetchMock.mockResolvedValue({ ok: true });
    await tick(10_000);
    expect(result.connected.value).toBe(true);
    unmount();
  });

  it('aborts a request that exceeds the 3 second timeout', async () => {
    let signal: AbortSignal | undefined;
    fetchMock.mockImplementation((_url: string, init: { signal: AbortSignal }) => {
      signal = init.signal;
      return new Promise((_resolve, reject) =>
        init.signal.addEventListener('abort', () => reject(new Error('aborted'))),
      );
    });
    const { unmount } = withSetup(() => useDaemonHealth());
    await tick(3_000);
    expect(signal?.aborted).toBe(true);
    unmount();
  });
});
