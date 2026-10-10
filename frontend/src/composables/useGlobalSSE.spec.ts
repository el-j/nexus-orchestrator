import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

type Listener = (e: MessageEvent) => void;
class FakeES {
  static all: FakeES[] = [];
  onopen: (() => void) | null = null;
  onmessage: Listener | null = null;
  onerror: (() => void) | null = null;
  closed = false;
  listeners = new Map<string, Listener>();
  constructor(public url: string) {
    FakeES.all.push(this);
  }
  addEventListener(type: string, l: Listener) {
    this.listeners.set(type, l);
  }
  close() {
    this.closed = true;
  }
  message(data: string) {
    this.onmessage?.({ data } as MessageEvent);
  }
  named(type: string, data: string) {
    this.listeners.get(type)?.({ data } as MessageEvent);
  }
}

const resolveServerUrl = vi.hoisted(() => vi.fn());
vi.mock('./useServerUrl', () => ({ resolveServerUrl }));

// The composable keeps module-level connection state, so load a fresh copy per test.
async function fresh() {
  vi.resetModules();
  const mod = await import('./useGlobalSSE');
  return mod.useGlobalSSE();
}
const tick = () => vi.advanceTimersByTimeAsync(0);

beforeEach(() => {
  vi.useFakeTimers();
  FakeES.all = [];
  resolveServerUrl.mockReset().mockResolvedValue('http://d:1');
  vi.stubGlobal('EventSource', FakeES);
});
afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe('useGlobalSSE', () => {
  it('connects once to /api/events and reports connected on open', async () => {
    const sse = await fresh();
    void sse.connect();
    void sse.connect(); // second call while connecting is a no-op
    await tick();
    void sse.connect(); // and so is one after the source exists
    await tick();
    expect(FakeES.all).toHaveLength(1);
    expect(FakeES.all[0].url).toBe('http://d:1/api/events');
    expect(sse.connected.value).toBe(false);
    FakeES.all[0].onopen!();
    expect(sse.connected.value).toBe(true);
  });

  it('dispatches messages to type and wildcard handlers', async () => {
    const sse = await fresh();
    const star = vi.fn();
    const typed = vi.fn();
    const other = vi.fn();
    sse.on('*', star);
    sse.on('task.queued', typed);
    sse.on('task.failed', other);
    void sse.connect();
    await tick();
    FakeES.all[0].message(JSON.stringify({ type: 'task.queued', taskId: 'a' }));
    expect(star).toHaveBeenCalledWith({ type: 'task.queued', taskId: 'a' });
    expect(typed).toHaveBeenCalledTimes(1);
    expect(other).not.toHaveBeenCalled();
  });

  it('labels a frame without a string type as "message"', async () => {
    const sse = await fresh();
    const h = vi.fn();
    sse.on('message', h);
    void sse.connect();
    await tick();
    FakeES.all[0].message(JSON.stringify({ type: 5 }));
    FakeES.all[0].message(JSON.stringify({}));
    expect(h).toHaveBeenCalledTimes(2);
  });

  it('routes named "log" events as type log', async () => {
    const sse = await fresh();
    const h = vi.fn();
    sse.on('log', h);
    void sse.connect();
    await tick();
    FakeES.all[0].named('log', JSON.stringify({ message: 'hi', type: 'ignored' }));
    expect(h).toHaveBeenCalledWith({ message: 'hi', type: 'log' });
  });

  it('ignores malformed frames', async () => {
    const sse = await fresh();
    const h = vi.fn();
    sse.on('*', h);
    void sse.connect();
    await tick();
    FakeES.all[0].message('{not json');
    FakeES.all[0].named('log', 'nope');
    expect(h).not.toHaveBeenCalled();
  });

  it('off() stops delivery and is safe for unknown types', async () => {
    const sse = await fresh();
    const h = vi.fn();
    sse.on('x', h);
    sse.off('x', h);
    sse.off('never-registered', h);
    void sse.connect();
    await tick();
    FakeES.all[0].message(JSON.stringify({ type: 'x' }));
    expect(h).not.toHaveBeenCalled();
  });

  it('reconnects after an error with exponential backoff capped at 30s', async () => {
    const sse = await fresh();
    void sse.connect();
    await tick();
    FakeES.all[0].onopen!();

    const fail = async (n: number) => {
      FakeES.all[n].onerror!();
      expect(FakeES.all[n].closed).toBe(true);
      expect(sse.connected.value).toBe(false);
    };

    await fail(0);
    await vi.advanceTimersByTimeAsync(2999);
    expect(FakeES.all).toHaveLength(1);
    await vi.advanceTimersByTimeAsync(1);
    expect(FakeES.all).toHaveLength(2); // after 3s

    await fail(1);
    await vi.advanceTimersByTimeAsync(5999);
    expect(FakeES.all).toHaveLength(2);
    await vi.advanceTimersByTimeAsync(1);
    expect(FakeES.all).toHaveLength(3); // after a further 6s

    for (let i = 2; i < 8; i++) {
      await fail(i);
      await vi.advanceTimersByTimeAsync(30_000);
    }
    const before = FakeES.all.length;
    await fail(before - 1);
    await vi.advanceTimersByTimeAsync(29_999);
    expect(FakeES.all).toHaveLength(before); // still capped at 30s, never longer
    await vi.advanceTimersByTimeAsync(1);
    expect(FakeES.all).toHaveLength(before + 1);
  });

  it('does not stack reconnect timers on repeated errors', async () => {
    const sse = await fresh();
    void sse.connect();
    await tick();
    FakeES.all[0].onerror!();
    FakeES.all[0].onerror!();
    await vi.advanceTimersByTimeAsync(3000);
    expect(FakeES.all).toHaveLength(2);
  });

  it('resets the backoff after a successful open', async () => {
    const sse = await fresh();
    void sse.connect();
    await tick();
    FakeES.all[0].onerror!();
    await vi.advanceTimersByTimeAsync(3000); // delay is now 6s
    FakeES.all[1].onopen!(); // success resets to 3s
    FakeES.all[1].onerror!();
    await vi.advanceTimersByTimeAsync(3000);
    expect(FakeES.all).toHaveLength(3);
  });

  it('retries when the server URL cannot be resolved', async () => {
    resolveServerUrl.mockRejectedValueOnce(new Error('no daemon'));
    const sse = await fresh();
    void sse.connect();
    await tick();
    expect(FakeES.all).toHaveLength(0);
    expect(sse.connected.value).toBe(false);
    await vi.advanceTimersByTimeAsync(3000);
    expect(FakeES.all).toHaveLength(1);
  });

  it('disconnect closes the stream, cancels a pending retry and allows connecting again', async () => {
    const sse = await fresh();
    void sse.connect();
    await tick();
    FakeES.all[0].onopen!();
    FakeES.all[0].onerror!(); // schedules a retry
    sse.disconnect();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(FakeES.all).toHaveLength(1);
    expect(sse.connected.value).toBe(false);

    void sse.connect();
    await tick();
    expect(FakeES.all).toHaveLength(2);
    sse.disconnect();
    expect(FakeES.all[1].closed).toBe(true);
  });

  it('a successful open clears a pending retry timer', async () => {
    const sse = await fresh();
    void sse.connect();
    await tick();
    FakeES.all[0].onerror!(); // retry scheduled
    sse.disconnect(); // drop it, then connect manually before the timer would fire
    void sse.connect();
    await tick();
    FakeES.all[1].onopen!();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(FakeES.all).toHaveLength(2);
  });
});
