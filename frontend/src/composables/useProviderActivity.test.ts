import { defineComponent, h } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest';
import { useProviderActivity } from './useProviderActivity';

const { resolveServerUrl } = vi.hoisted(() => ({
  resolveServerUrl: vi.fn(),
}));

vi.mock('./useServerUrl', () => ({ resolveServerUrl }));

function mountPA() {
  let state: ReturnType<typeof useProviderActivity>;
  const Harness = defineComponent({
    setup() {
      state = useProviderActivity();
      return () => h('div');
    },
  });
  const wrapper = mount(Harness);
  return { wrapper, get: () => state! };
}

function makeActivity(agentName: string, model: string, offsetSec = 0) {
  return {
    agentName,
    model,
    timestamp: new Date(Date.now() - offsetSec * 1000).toISOString(),
    activityType: 'generation',
  };
}

describe('useProviderActivity', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resolveServerUrl.mockResolvedValue('http://daemon');
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('groups activities by agentName and collects model IDs', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => [
          makeActivity('ollama', 'llama3'),
          makeActivity('ollama', 'codellama'),
          makeActivity('lmstudio', 'mistral'),
        ],
      }),
    );

    const { wrapper, get } = mountPA();
    await flushPromises();

    const ollama = get().providerStates.value.find((p) => p.provider === 'ollama');
    const lmstudio = get().providerStates.value.find((p) => p.provider === 'lmstudio');

    expect(ollama?.models).toContain('llama3');
    expect(ollama?.models).toContain('codellama');
    expect(lmstudio?.models).toContain('mistral');

    wrapper.unmount();
  });

  it('sets active:true for activities within the last 30s', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => [makeActivity('recent', 'model-a', 5)],
      }),
    );

    const { wrapper, get } = mountPA();
    await flushPromises();

    expect(get().providerStates.value.find((p) => p.provider === 'recent')?.active).toBe(true);

    wrapper.unmount();
  });

  it('sets active:false for activities older than 30s', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => [makeActivity('stale', 'model-a', 60)],
      }),
    );

    const { wrapper, get } = mountPA();
    await flushPromises();

    expect(get().providerStates.value.find((p) => p.provider === 'stale')?.active).toBe(false);

    wrapper.unmount();
  });

  it('sets error.value when fetch throws', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('network down')));

    const { wrapper, get } = mountPA();
    await flushPromises();

    expect(get().error.value).toContain('network down');

    wrapper.unmount();
  });

  it('cleans up the polling interval on unmount', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => [] }));
    const clearSpy = vi.spyOn(globalThis, 'clearInterval');

    const { wrapper } = mountPA();
    await flushPromises();
    wrapper.unmount();

    expect(clearSpy).toHaveBeenCalled();
  });

  it('polls again on the 15s interval', async () => {
    vi.useFakeTimers();
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: async () => [] });
    vi.stubGlobal('fetch', fetchMock);

    const { wrapper } = mountPA();
    await flushPromises();
    const callsBefore = fetchMock.mock.calls.length;

    await vi.advanceTimersByTimeAsync(15_000);
    await flushPromises();

    expect(fetchMock.mock.calls.length).toBeGreaterThan(callsBefore);

    wrapper.unmount();
    vi.useRealTimers();
  });
});
