import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { withSetup as mountSetup } from '../test/withSetup';
import type { LogEntry } from '../types/domain';

const h = vi.hoisted(() => ({
  on: vi.fn(),
  off: vi.fn(),
  connected: { value: false },
}));
vi.mock('./useServerUrl', () => ({ resolveServerUrl: () => Promise.resolve('http://d') }));
vi.mock('./useGlobalSSE', async () => {
  const { ref } = await import('vue');
  const connected = ref(false);
  h.connected = connected as unknown as { value: boolean };
  return { useGlobalSSE: () => ({ connected, on: h.on, off: h.off }) };
});

import { useLogs } from './useLogs';

// The mocked stream state is shared by all tests, so every mounted composable must
// be torn down or earlier tests' watchers would react to later connection flips.
const mounted: Array<() => void> = [];
function withSetup<T>(fn: () => T) {
  const m = mountSetup(fn);
  let done = false;
  const unmount = () => {
    if (!done) {
      done = true;
      m.unmount();
    }
  };
  mounted.push(unmount);
  return { ...m, unmount };
}

const fetchMock = vi.fn();
const entry = (m: string): LogEntry => ({ timestamp: 't', level: 'info', source: 's', message: m });
const ok = (rows: unknown) => Promise.resolve(new Response(JSON.stringify(rows), { status: 200 }));
const flush = () => vi.advanceTimersByTimeAsync(0);
const WARN = 'Realtime log stream disconnected. Polling fallback active while reconnecting.';

beforeEach(() => {
  vi.useFakeTimers();
  h.on.mockReset();
  h.off.mockReset();
  h.connected.value = false;
  fetchMock.mockReset().mockImplementation(() => ok([]));
  vi.stubGlobal('fetch', fetchMock);
});
afterEach(() => {
  mounted.splice(0).forEach((u) => u());
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

const logHandler = () =>
  h.on.mock.calls.find((c) => c[0] === 'log')![1] as (d: Record<string, unknown>) => void;

describe('useLogs', () => {
  it('loads history on mount and subscribes to log events', async () => {
    fetchMock.mockImplementation(() => ok([entry('a')]));
    const { result } = withSetup(() => useLogs());
    await flush();
    expect(fetchMock).toHaveBeenCalledWith('http://d/api/logs');
    expect(result.logs.value.map((l) => l.message)).toEqual(['a']);
    expect(h.on).toHaveBeenCalledWith('log', expect.any(Function));
  });

  it('appends live log events and ignores other event types', async () => {
    const { result } = withSetup(() => useLogs());
    await flush();
    logHandler()({ type: 'log', timestamp: 't', level: 'warn', source: 'x', message: 'live' });
    logHandler()({ type: 'task.queued', message: 'nope' });
    expect(result.logs.value).toEqual([
      { timestamp: 't', level: 'warn', source: 'x', message: 'live' },
    ]);
  });

  it('keeps only the newest 2000 lines, from history and from the stream', async () => {
    fetchMock.mockImplementation(() => ok(Array.from({ length: 2500 }, (_, i) => entry(`h${i}`))));
    const { result } = withSetup(() => useLogs());
    await flush();
    expect(result.logs.value).toHaveLength(2000);
    expect(result.logs.value[0].message).toBe('h500');

    logHandler()({ type: 'log', message: 'new' });
    expect(result.logs.value).toHaveLength(2000);
    expect(result.logs.value[0].message).toBe('h501');
    expect(result.logs.value[1999].message).toBe('new');
  });

  it('clear() empties the list', async () => {
    fetchMock.mockImplementation(() => ok([entry('a')]));
    const { result } = withSetup(() => useLogs());
    await flush();
    result.clear();
    expect(result.logs.value).toEqual([]);
  });

  it('polls every 3s while the stream is down and warns', async () => {
    const { result } = withSetup(() => useLogs());
    await flush();
    expect(result.error.value).toBe(WARN);
    fetchMock.mockClear();
    await vi.advanceTimersByTimeAsync(9000);
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it('stops polling and clears the warning once the stream connects', async () => {
    const { result } = withSetup(() => useLogs());
    await flush();
    h.connected.value = true;
    await flush();
    expect(result.error.value).toBeNull();
    fetchMock.mockClear();
    await vi.advanceTimersByTimeAsync(10_000);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('resumes polling and warns when the stream drops again', async () => {
    h.connected.value = true;
    const { result } = withSetup(() => useLogs());
    await flush();
    expect(result.error.value).toBeNull();
    fetchMock.mockClear();
    h.connected.value = false;
    await flush();
    expect(result.error.value).toBe(WARN);
    await vi.advanceTimersByTimeAsync(3000);
    expect(fetchMock.mock.calls.length).toBeGreaterThanOrEqual(2);
  });

  it('does not start a second poll timer', async () => {
    withSetup(() => useLogs());
    await flush();
    h.connected.value = true;
    await flush();
    h.connected.value = false;
    await flush();
    h.connected.value = true;
    h.connected.value = false;
    await flush();
    fetchMock.mockClear();
    await vi.advanceTimersByTimeAsync(3000);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('reports the real error when connected, the reconnect warning when not', async () => {
    h.connected.value = true;
    fetchMock.mockRejectedValue(new Error('boom'));
    const { result } = withSetup(() => useLogs());
    await flush();
    expect(result.error.value).toBe('boom');
    fetchMock.mockRejectedValue('weird');
    h.connected.value = false;
    await flush();
    expect(result.error.value).toBe(WARN);
  });

  it('uses a generic message for a non-Error failure while connected', async () => {
    h.connected.value = true;
    fetchMock.mockRejectedValue('weird');
    const { result } = withSetup(() => useLogs());
    await flush();
    expect(result.error.value).toBe('Failed to load logs');
  });

  it('keeps the current list when the server answers with an error status', async () => {
    fetchMock.mockImplementationOnce(() => ok([entry('keep')]));
    const { result } = withSetup(() => useLogs());
    await flush();
    fetchMock.mockImplementation(() => Promise.resolve(new Response('', { status: 500 })));
    await vi.advanceTimersByTimeAsync(3000);
    expect(result.logs.value.map((l) => l.message)).toEqual(['keep']);
  });

  it('unsubscribes and stops polling on unmount', async () => {
    const { unmount } = withSetup(() => useLogs());
    await flush();
    unmount();
    expect(h.off).toHaveBeenCalledWith('log', expect.any(Function));
    fetchMock.mockClear();
    await vi.advanceTimersByTimeAsync(10_000);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('unmounting while not polling is harmless', async () => {
    h.connected.value = true;
    const { unmount } = withSetup(() => useLogs());
    await flush();
    expect(() => unmount()).not.toThrow();
  });
});
