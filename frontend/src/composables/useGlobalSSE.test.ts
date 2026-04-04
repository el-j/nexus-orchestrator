import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest';
import { flushPromises } from '@vue/test-utils';

// ── Mock resolveServerUrl ─────────────────────────────────────────────────────

const { resolveServerUrl } = vi.hoisted(() => ({
  resolveServerUrl: vi.fn(),
}));

vi.mock('./useServerUrl', () => ({
  resolveServerUrl,
}));

// ── Class-based EventSource mock ──────────────────────────────────────────────
// vi.fn() does not invoke its implementation when called with `new`, so we use
// a real class and track calls via module-level variables.

let capturedInstance: FakeEventSource | null = null;
let constructorCalls = 0;

class FakeEventSource {
  readonly url: string;
  onopen: (() => void) | null = null;
  onmessage: ((e: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  readonly close = vi.fn();
  private readonly named = new Map<string, Array<(e: { data: string }) => void>>();

  constructor(url: string) {
    this.url = url;
    capturedInstance = this;
    constructorCalls++;
  }

  addEventListener(type: string, fn: (e: { data: string }) => void) {
    if (!this.named.has(type)) this.named.set(type, []);
    this.named.get(type)!.push(fn);
  }

  /** Test helper: fire an SSE event on this connection. */
  emit(type: string, data: string) {
    const event = { data };
    if (type === 'message') {
      this.onmessage?.(event);
    } else {
      this.named.get(type)?.forEach((fn) => fn(event));
    }
  }
}

// ── Setup / Teardown ──────────────────────────────────────────────────────────

beforeEach(async () => {
  capturedInstance = null;
  constructorCalls = 0;
  resolveServerUrl.mockResolvedValue('http://testdaemon');
  vi.stubGlobal('EventSource', FakeEventSource);
  const { useGlobalSSE } = await import('./useGlobalSSE');
  useGlobalSSE().disconnect();
});

afterEach(async () => {
  const { useGlobalSSE } = await import('./useGlobalSSE');
  useGlobalSSE().disconnect();
  vi.unstubAllGlobals();
});

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('useGlobalSSE', () => {
  it('creates EventSource with the correct events URL on connect()', async () => {
    const { useGlobalSSE } = await import('./useGlobalSSE');
    const { connect, disconnect } = useGlobalSSE();

    await connect();

    expect(constructorCalls).toBe(1);
    expect(capturedInstance).not.toBeNull();
    expect(capturedInstance!.url).toBe('http://testdaemon/api/events');

    disconnect();
  });

  it('routes onmessage events to subscribers matching the type field', async () => {
    const { useGlobalSSE } = await import('./useGlobalSSE');
    const { connect, on, off, disconnect } = useGlobalSSE();
    await connect();

    const taskHandler = vi.fn();
    const activityHandler = vi.fn();
    on('task', taskHandler);
    on('activity', activityHandler);

    capturedInstance!.emit('message', JSON.stringify({ type: 'task', id: 'T-1' }));
    expect(taskHandler).toHaveBeenCalledWith({ type: 'task', id: 'T-1' });
    expect(activityHandler).not.toHaveBeenCalled();

    capturedInstance!.emit('message', JSON.stringify({ type: 'activity', id: 'A-1' }));
    expect(activityHandler).toHaveBeenCalledWith({ type: 'activity', id: 'A-1' });
    expect(taskHandler).toHaveBeenCalledTimes(1);

    off('task', taskHandler);
    off('activity', activityHandler);
    disconnect();
  });

  it('dispatches named log events to log subscribers via addEventListener', async () => {
    const { useGlobalSSE } = await import('./useGlobalSSE');
    const { connect, on, off, disconnect } = useGlobalSSE();
    await connect();

    const logHandler = vi.fn();
    on('log', logHandler);

    capturedInstance!.emit('log', JSON.stringify({ type: 'log', message: 'hello' }));
    expect(logHandler).toHaveBeenCalledWith({ type: 'log', message: 'hello' });

    off('log', logHandler);
    disconnect();
  });

  it('off() prevents handler from being called after removal', async () => {
    const { useGlobalSSE } = await import('./useGlobalSSE');
    const { connect, on, off, disconnect } = useGlobalSSE();
    await connect();

    const handler = vi.fn();
    on('task', handler);

    capturedInstance!.emit('message', JSON.stringify({ type: 'task', id: '1' }));
    expect(handler).toHaveBeenCalledTimes(1);

    off('task', handler);
    capturedInstance!.emit('message', JSON.stringify({ type: 'task', id: '2' }));
    expect(handler).toHaveBeenCalledTimes(1);

    disconnect();
  });

  it('disconnect() closes EventSource and sets connected.value to false', async () => {
    const { useGlobalSSE } = await import('./useGlobalSSE');
    const { connect, disconnect, connected } = useGlobalSSE();
    await connect();

    const instance = capturedInstance!;
    expect(instance).not.toBeNull();
    instance.onopen?.();
    expect(connected.value).toBe(true);

    disconnect();

    expect(instance.close).toHaveBeenCalledTimes(1);
    expect(connected.value).toBe(false);
  });

  it('reconnects after EventSource onerror fires', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });

    const { useGlobalSSE } = await import('./useGlobalSSE');
    const { connect, disconnect } = useGlobalSSE();
    await connect();
    await flushPromises();

    const firstInstance = capturedInstance;
    expect(firstInstance).not.toBeNull();
    firstInstance!.onopen?.();

    firstInstance!.onerror?.();
    expect(constructorCalls).toBe(1);

    await vi.advanceTimersByTimeAsync(3200);
    await flushPromises();

    expect(constructorCalls).toBe(2);
    expect(capturedInstance).not.toBe(firstInstance);

    disconnect();
    vi.useRealTimers();
  });

  it('wildcard handlers receive all dispatched events', async () => {
    const { useGlobalSSE } = await import('./useGlobalSSE');
    const { connect, on, off, disconnect } = useGlobalSSE();
    await connect();

    const wildcardHandler = vi.fn();
    on('*', wildcardHandler);

    capturedInstance!.emit('message', JSON.stringify({ type: 'task' }));
    capturedInstance!.emit('message', JSON.stringify({ type: 'activity' }));
    expect(wildcardHandler).toHaveBeenCalledTimes(2);

    off('*', wildcardHandler);
    disconnect();
  });
});
