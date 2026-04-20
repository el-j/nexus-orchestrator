import { defineComponent, h } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useProviders } from './useProviders';

const { mockGetProviders } = vi.hoisted(() => ({
  mockGetProviders: vi.fn(),
}));

vi.mock('../types/wails', () => ({
  getProviders: mockGetProviders,
}));

function makeProvider(name: string, active = true) {
  return { name, active, activeModel: 'llama3', models: ['llama3'], baseURL: 'http://localhost' };
}

function mountUseProviders() {
  let state: ReturnType<typeof useProviders>;
  const Harness = defineComponent({
    setup() {
      state = useProviders();
      return () => h('div');
    },
  });
  const wrapper = mount(Harness);
  return { wrapper, get: () => state! };
}

describe('useProviders', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetProviders.mockResolvedValue([]);
  });

  it('populates providers.value on mount; loading resets to false', async () => {
    mockGetProviders.mockResolvedValue([makeProvider('ollama'), makeProvider('lmstudio', false)]);

    const { wrapper, get } = mountUseProviders();
    expect(get().loading.value).toBe(true);

    await flushPromises();

    expect(get().providers.value).toHaveLength(2);
    expect(get().providers.value[0].name).toBe('ollama');
    expect(get().loading.value).toBe(false);
    expect(get().error.value).toBeNull();

    wrapper.unmount();
  });

  it('sets error.value when getProviders throws', async () => {
    mockGetProviders.mockRejectedValue(new Error('LLM backend unreachable'));

    const { wrapper, get } = mountUseProviders();
    await flushPromises();

    expect(get().error.value).toBe('LLM backend unreachable');
    expect(get().providers.value).toHaveLength(0);
    expect(get().loading.value).toBe(false);

    wrapper.unmount();
  });

  it('refresh() re-fetches providers', async () => {
    mockGetProviders
      .mockResolvedValueOnce([makeProvider('ollama')])
      .mockResolvedValueOnce([makeProvider('ollama'), makeProvider('lmstudio')]);

    const { wrapper, get } = mountUseProviders();
    await flushPromises();
    expect(get().providers.value).toHaveLength(1);

    await get().refresh();
    await flushPromises();

    expect(get().providers.value).toHaveLength(2);
    expect(mockGetProviders).toHaveBeenCalledTimes(2);

    wrapper.unmount();
  });

  it('polls again on the 30s interval', async () => {
    vi.useFakeTimers();
    mockGetProviders.mockResolvedValue([makeProvider('ollama')]);

    const { wrapper } = mountUseProviders();
    await flushPromises();
    const callsBefore = mockGetProviders.mock.calls.length;

    await vi.advanceTimersByTimeAsync(30_000);
    await flushPromises();

    expect(mockGetProviders.mock.calls.length).toBeGreaterThan(callsBefore);

    wrapper.unmount();
    vi.useRealTimers();
  });

  it('cleans up the polling interval on unmount', async () => {
    mockGetProviders.mockResolvedValue([]);
    const clearSpy = vi.spyOn(globalThis, 'clearInterval');

    const { wrapper } = mountUseProviders();
    await flushPromises();
    wrapper.unmount();

    expect(clearSpy).toHaveBeenCalled();
  });

  it('handles null response from getProviders gracefully', async () => {
    mockGetProviders.mockResolvedValue(null);

    const { wrapper, get } = mountUseProviders();
    await flushPromises();

    expect(get().providers.value).toEqual([]);

    wrapper.unmount();
  });
});
