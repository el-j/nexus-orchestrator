import { defineComponent, h, ref } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';

// vi.hoisted runs before imports — cannot reference imported values (e.g. ref) here
const { resolveServerUrl, mockOn, mockOff } = vi.hoisted(() => ({
  resolveServerUrl: vi.fn(),
  mockOn: vi.fn(),
  mockOff: vi.fn(),
}));

// ref() is safe at module scope (after imports resolve)
const connectedRef = ref(true);

vi.mock('./useServerUrl', () => ({ resolveServerUrl }));
vi.mock('./useGlobalSSE', () => ({
  useGlobalSSE: () => ({ connected: connectedRef, on: mockOn, off: mockOff }),
}));

function makeEntry(message: string) {
  return { timestamp: new Date().toISOString(), level: 'info' as const, source: 'test', message };
}

async function mountUseLogs() {
  vi.resetModules();
  const { useLogs } = await import('./useLogs');
  let state: ReturnType<typeof useLogs>;
  const Harness = defineComponent({
    setup() {
      state = useLogs();
      return () => h('div');
    },
  });
  const wrapper = mount(Harness);
  return { wrapper, get: () => state! };
}

describe('useLogs', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    connectedRef.value = true;
    resolveServerUrl.mockResolvedValue('http://daemon');
  });

  it('fetches initial logs from /api/logs on mount', async () => {
    const entries = [makeEntry('boot'), makeEntry('ready')];
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => entries,
      }),
    );

    const { wrapper, get } = await mountUseLogs();
    await flushPromises();

    expect(get().logs.value).toHaveLength(2);
    expect(get().logs.value[0].message).toBe('boot');

    wrapper.unmount();
    vi.unstubAllGlobals();
  });

  it('appends log entries from SSE events', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => [] }));

    const { wrapper, get } = await mountUseLogs();
    await flushPromises();
    expect(get().logs.value).toHaveLength(0);

    const logCall = mockOn.mock.calls.find(([type]) => type === 'log');
    expect(logCall).toBeDefined();
    const handler = logCall![1] as (data: Record<string, unknown>) => void;

    handler({
      type: 'log',
      timestamp: new Date().toISOString(),
      level: 'info',
      source: 'svc',
      message: 'SSE msg',
    });

    expect(get().logs.value).toHaveLength(1);
    expect(get().logs.value[0].message).toBe('SSE msg');

    wrapper.unmount();
    vi.unstubAllGlobals();
  });

  it('clear() empties logs.value', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => [makeEntry('a'), makeEntry('b')],
      }),
    );

    const { wrapper, get } = await mountUseLogs();
    await flushPromises();
    expect(get().logs.value).toHaveLength(2);

    get().clear();
    expect(get().logs.value).toHaveLength(0);

    wrapper.unmount();
    vi.unstubAllGlobals();
  });

  it('starts polling when SSE is disconnected', async () => {
    vi.useFakeTimers();
    connectedRef.value = false;
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: async () => [] });
    vi.stubGlobal('fetch', fetchMock);

    const { wrapper } = await mountUseLogs();
    await flushPromises();
    const callsAfterMount = fetchMock.mock.calls.length;

    await vi.advanceTimersByTimeAsync(3_000);
    await flushPromises();

    expect(fetchMock.mock.calls.length).toBeGreaterThan(callsAfterMount);

    wrapper.unmount();
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it('stops polling when SSE reconnects', async () => {
    vi.useFakeTimers();
    connectedRef.value = false;
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: async () => [] });
    vi.stubGlobal('fetch', fetchMock);

    const { wrapper } = await mountUseLogs();
    await flushPromises();

    // SSE reconnects
    connectedRef.value = true;
    await flushPromises();

    const callsAtReconnect = fetchMock.mock.calls.length; // includes watcher-triggered fetch
    await vi.advanceTimersByTimeAsync(6_000);
    await flushPromises();

    // Polling stopped — fetch count must not grow after reconnect
    expect(fetchMock.mock.calls.length).toBe(callsAtReconnect);

    wrapper.unmount();
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it('cleans up SSE handler on unmount', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => [] }));

    const { wrapper } = await mountUseLogs();
    await flushPromises();
    wrapper.unmount();

    expect(mockOff).toHaveBeenCalledWith('log', expect.any(Function));

    vi.unstubAllGlobals();
  });
});
