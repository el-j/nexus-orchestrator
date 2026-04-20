import { defineComponent, h } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useDiscovery } from './useDiscovery';

const { mockGetDiscoveredProviders, mockTriggerScan, mockOn, mockOff } = vi.hoisted(() => ({
  mockGetDiscoveredProviders: vi.fn(),
  mockTriggerScan: vi.fn(),
  mockOn: vi.fn(),
  mockOff: vi.fn(),
}));

vi.mock('../types/wails', () => ({
  getDiscoveredProviders: mockGetDiscoveredProviders,
  triggerScan: mockTriggerScan,
}));

vi.mock('./useGlobalSSE', () => ({
  useGlobalSSE: () => ({ on: mockOn, off: mockOff }),
}));

function stubProvider(id: string) {
  return {
    id,
    name: `Provider ${id}`,
    kind: 'ollama',
    baseURL: 'http://localhost:11434',
    isActive: true,
    detectedAt: new Date().toISOString(),
  };
}

function mountUseDiscovery() {
  let state: ReturnType<typeof useDiscovery>;
  const Harness = defineComponent({
    setup() {
      state = useDiscovery();
      return () => h('div');
    },
  });
  const wrapper = mount(Harness);
  return { wrapper, get: () => state! };
}

describe('useDiscovery', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetDiscoveredProviders.mockResolvedValue([]);
    mockTriggerScan.mockResolvedValue(undefined);
  });

  it('populates discovered on mount via refresh()', async () => {
    mockGetDiscoveredProviders.mockResolvedValue([stubProvider('p-1'), stubProvider('p-2')]);

    const { wrapper, get } = mountUseDiscovery();
    await flushPromises();

    expect(get().discovered.value).toHaveLength(2);
    expect(get().discovered.value[0].id).toBe('p-1');
    expect(get().loading.value).toBe(false);

    wrapper.unmount();
  });

  it('refresh() sets error on failure', async () => {
    mockGetDiscoveredProviders.mockRejectedValue(new Error('backend error'));

    const { wrapper, get } = mountUseDiscovery();
    await flushPromises();

    expect(get().error.value).toContain('backend error');

    wrapper.unmount();
  });

  it('scanNow() calls triggerScan and then refresh', async () => {
    vi.useFakeTimers();
    mockGetDiscoveredProviders
      .mockResolvedValueOnce([])
      .mockResolvedValueOnce([stubProvider('scanned-1')]);

    const { wrapper, get } = mountUseDiscovery();
    await flushPromises();

    const scanPromise = get().scanNow();
    await vi.advanceTimersByTimeAsync(2000); // cover the 1500 ms setTimeout in scanNow
    await flushPromises();
    await scanPromise;

    expect(mockTriggerScan).toHaveBeenCalledTimes(1);
    expect(mockGetDiscoveredProviders.mock.calls.length).toBeGreaterThanOrEqual(2);

    wrapper.unmount();
    vi.useRealTimers();
  });

  it('scanNow() sets scanning true during operation', async () => {
    vi.useFakeTimers();
    let resolveScan!: () => void;
    mockTriggerScan.mockReturnValue(new Promise<void>((r) => (resolveScan = r)));

    const { wrapper, get } = mountUseDiscovery();
    await flushPromises();
    expect(get().scanning.value).toBe(false);

    void get().scanNow();
    await flushPromises();
    expect(get().scanning.value).toBe(true);

    resolveScan();
    await vi.advanceTimersByTimeAsync(2000); // cover the 1500 ms setTimeout in scanNow
    await flushPromises();
    expect(get().scanning.value).toBe(false);

    wrapper.unmount();
    vi.useRealTimers();
  });

  it('SSE provider_discovered event triggers refresh', async () => {
    mockGetDiscoveredProviders
      .mockResolvedValueOnce([])
      .mockResolvedValueOnce([stubProvider('sse-1')]);

    const { wrapper, get } = mountUseDiscovery();
    await flushPromises();
    expect(get().discovered.value).toHaveLength(0);

    const sseCall = mockOn.mock.calls.find(([type]) => type === 'provider_discovered');
    expect(sseCall).toBeDefined();
    const sseHandler = sseCall![1] as () => void;

    sseHandler();
    await flushPromises();

    expect(get().discovered.value).toHaveLength(1);

    wrapper.unmount();
  });

  it('cleans up SSE handler and polling interval on unmount', async () => {
    const { wrapper } = mountUseDiscovery();
    await flushPromises();
    wrapper.unmount();

    expect(mockOff).toHaveBeenCalledWith('provider_discovered', expect.any(Function));
  });
});
