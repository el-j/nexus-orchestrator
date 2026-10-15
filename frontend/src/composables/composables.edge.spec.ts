import { defineComponent, h } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// Edge cases shared by the polling composables: null payloads from the daemon,
// non-Error rejections, and a component that goes away before its first fetch finishes.

const w = vi.hoisted(() => ({
  getProviders: vi.fn(),
  listAISessions: vi.fn(),
  deregisterAISession: vi.fn(),
  purgeDisconnectedSessions: vi.fn(),
  getDiscoveredProviders: vi.fn(),
  triggerScan: vi.fn(),
  getQueue: vi.fn(),
  getAllTasks: vi.fn(),
}));
vi.mock('../types/wails', () => w);
const sse = vi.hoisted(() => ({ on: vi.fn(), off: vi.fn() }));
vi.mock('./useGlobalSSE', () => ({ useGlobalSSE: () => sse }));
vi.mock('./useServerUrl', () => ({ resolveServerUrl: vi.fn().mockResolvedValue('http://daemon') }));

import { useProviders } from './useProviders';
import { useAISessions } from './useAISessions';
import { useDiscovery } from './useDiscovery';
import { useTasks } from './useTasks';
import { useActivities } from './useActivities';
import { useProjectFilter } from './useProjectFilter';
import { useDaemonHealth } from './useDaemonHealth';

function host<T>(use: () => T) {
  let state!: T;
  const wrapper = mount(
    defineComponent({
      setup() {
        state = use();
        return () => h('div');
      },
    }),
  );
  return { wrapper, state: () => state };
}

beforeEach(() => {
  vi.useFakeTimers();
  Object.values(w).forEach((f) => f.mockReset());
  w.getProviders.mockResolvedValue([]);
  w.listAISessions.mockResolvedValue([]);
  w.getDiscoveredProviders.mockResolvedValue([]);
  w.getQueue.mockResolvedValue([]);
  w.getAllTasks.mockResolvedValue([]);
});
afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

type Poller = {
  error: { value: string | null };
  refresh(): Promise<void>;
} & Record<string, { value: unknown }>;

const pollers: Array<[string, () => unknown, ReturnType<typeof vi.fn>, string, string]> = [
  ['useProviders', () => useProviders(), w.getProviders, 'providers', 'Failed to load providers'],
  [
    'useAISessions',
    () => useAISessions(),
    w.listAISessions,
    'sessions',
    'Failed to load AI sessions',
  ],
];

describe.each(pollers)('%s', (_name, use, source, field, fallback) => {
  it('treats a null payload as an empty list', async () => {
    source.mockResolvedValue(null);
    const h = host(use as () => Poller);
    await flushPromises();
    expect(h.state()[field].value).toEqual([]);
  });

  it('reports an Error message, and a generic one for anything else', async () => {
    source.mockRejectedValue(new Error('daemon down'));
    const h = host(use as () => Poller);
    await flushPromises();
    expect(h.state().error.value).toBe('daemon down');

    source.mockRejectedValue('plain string');
    await h.state().refresh();
    expect(h.state().error.value).toBe(fallback);
  });

  it('survives being unmounted before the first fetch resolves', async () => {
    const h = host(use as () => Poller);
    h.wrapper.unmount();
    await flushPromises();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(source).toHaveBeenCalledTimes(1);
  });

  it('keeps polling while mounted and stops once unmounted', async () => {
    const h = host(use as () => Poller);
    await flushPromises();
    await vi.advanceTimersByTimeAsync(30_000);
    const calls = source.mock.calls.length;
    expect(calls).toBeGreaterThan(1);
    h.wrapper.unmount();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(source).toHaveBeenCalledTimes(calls);
  });
});

describe('useAISessions actions', () => {
  it('refreshes after deregistering and purging', async () => {
    const h = host(() => useAISessions());
    await flushPromises();
    const before = w.listAISessions.mock.calls.length;
    await h.state().deregister('s1');
    await h.state().purgeDisconnected();
    expect(w.deregisterAISession).toHaveBeenCalledWith('s1');
    expect(w.purgeDisconnectedSessions).toHaveBeenCalled();
    expect(w.listAISessions.mock.calls.length).toBe(before + 2);
  });
});

describe('useDiscovery', () => {
  it('treats null as empty and stringifies failures', async () => {
    w.getDiscoveredProviders.mockResolvedValue(null);
    const h = host(() => useDiscovery());
    await flushPromises();
    expect(h.state().discovered.value).toEqual([]);
    w.getDiscoveredProviders.mockRejectedValue(new Error('x'));
    await h.state().refresh();
    expect(h.state().error.value).toContain('x');
  });

  it('scans, waits for the scanner, refreshes, and reports a failed scan', async () => {
    const h = host(() => useDiscovery());
    await flushPromises();
    const scan = h.state().scanNow();
    expect(h.state().scanning.value).toBe(true);
    await vi.advanceTimersByTimeAsync(1_500);
    await scan;
    expect(h.state().scanning.value).toBe(false);
    expect(w.triggerScan).toHaveBeenCalled();

    w.triggerScan.mockRejectedValue(new Error('scan failed'));
    await h.state().scanNow();
    expect(h.state().error.value).toContain('scan failed');
    expect(h.state().scanning.value).toBe(false);
  });

  it('survives an early unmount', async () => {
    const h = host(() => useDiscovery());
    h.wrapper.unmount();
    await flushPromises();
    expect(w.getDiscoveredProviders).toHaveBeenCalledTimes(1);
  });
});

describe('useTasks', () => {
  it('survives an early unmount', async () => {
    const h = host(() => useTasks());
    h.wrapper.unmount();
    await flushPromises();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(w.getQueue).toHaveBeenCalledTimes(1);
  });
});

describe('useActivities', () => {
  const stubFetch = (impl: () => Promise<unknown>) => vi.stubGlobal('fetch', vi.fn(impl));

  it('reports an HTTP failure, a thrown Error and a non-Error', async () => {
    stubFetch(() => Promise.resolve({ ok: false, status: 503 }));
    const h = host(() => useActivities());
    await flushPromises();
    expect(h.state().error.value).toBe('HTTP 503');

    stubFetch(() => Promise.reject(new Error('offline')));
    await h.state().refresh();
    expect(h.state().error.value).toBe('offline');

    stubFetch(() => Promise.reject('nope'));
    await h.state().refresh();
    expect(h.state().error.value).toBe('Failed to load activities');
  });

  it('treats a null body as no activity and filters by type', async () => {
    stubFetch(() => Promise.resolve({ ok: true, json: async () => null }));
    const h = host(() => useActivities({ typeFilter: 'message' }));
    await flushPromises();
    expect(h.state().activities.value).toEqual([]);

    stubFetch(() =>
      Promise.resolve({
        ok: true,
        json: async () => [
          { id: '1', activityType: 'message', agentName: 'a' },
          { id: '2', activityType: 'tool_use', agentName: 'a' },
        ],
      }),
    );
    await h.state().refresh();
    expect(h.state().filtered.value.map((a) => a.id)).toEqual(['1']);
  });

  it('prepends live activity from the event stream and ignores everything else', async () => {
    stubFetch(() => Promise.resolve({ ok: true, json: async () => [{ id: 'old' }] }));
    sse.on.mockClear();
    const h = host(() => useActivities());
    await flushPromises();
    const handler = sse.on.mock.calls.find((c) => c[0] === 'ai_activity_new')![1] as (
      d: unknown,
    ) => void;
    handler({ type: 'ai_activity_new', activity: { id: 'new' } });
    handler({ type: 'ai_activity_new' }); // no payload
    handler({ type: 'task.completed', activity: { id: 'wrong-type' } });
    expect(h.state().activities.value.map((a) => a.id)).toEqual(['new', 'old']);
  });

  it('survives an early unmount', async () => {
    stubFetch(() => Promise.resolve({ ok: true, json: async () => [] }));
    const h = host(() => useActivities());
    h.wrapper.unmount();
    await flushPromises();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(fetch).toHaveBeenCalledTimes(1);
  });
});

describe('useProjectFilter', () => {
  it('lists history projects, ignores a failed history fetch, and survives an early unmount', async () => {
    w.getAllTasks.mockResolvedValue([
      { projectPath: '/b' },
      { projectPath: '' },
      { projectPath: '/a' },
    ]);
    const h = host(() => useProjectFilter());
    await flushPromises();
    expect(h.state().projectList.value).toEqual(['/a', '/b']);

    w.getAllTasks.mockRejectedValue(new Error('x'));
    await vi.advanceTimersByTimeAsync(60_000);
    expect(h.state().projectList.value).toEqual(['/a', '/b']);

    host(() => useProjectFilter()).wrapper.unmount();
  });
});

describe('useDaemonHealth', () => {
  it('flips to disconnected only after three consecutive failures, on HTTP errors or network errors', async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
    fetchMock.mockResolvedValue({ ok: true });
    const h = host(() => useDaemonHealth());
    await flushPromises();
    expect(h.state().connected.value).toBe(true);

    fetchMock.mockResolvedValue({ ok: false });
    for (let i = 0; i < 2; i++) await vi.advanceTimersByTimeAsync(10_000);
    expect(h.state().connected.value).toBe(true);
    await vi.advanceTimersByTimeAsync(10_000);
    expect(h.state().connected.value).toBe(false);

    fetchMock.mockResolvedValue({ ok: true });
    await vi.advanceTimersByTimeAsync(10_000);
    expect(h.state().connected.value).toBe(true);

    fetchMock.mockRejectedValue(new Error('refused'));
    for (let i = 0; i < 3; i++) await vi.advanceTimersByTimeAsync(10_000);
    expect(h.state().connected.value).toBe(false);
    h.wrapper.unmount();
  });
});
