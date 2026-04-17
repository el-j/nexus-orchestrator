import { defineComponent, h } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest';

const { resolveServerUrl } = vi.hoisted(() => ({
  resolveServerUrl: vi.fn(),
}));

vi.mock('./useServerUrl', () => ({ resolveServerUrl }));

async function mountHealth() {
  vi.resetModules();
  const { useDaemonHealth } = await import('./useDaemonHealth');
  let state: ReturnType<typeof useDaemonHealth>;
  const Harness = defineComponent({
    setup() {
      state = useDaemonHealth();
      return () => h('div');
    },
  });
  const wrapper = mount(Harness);
  return { wrapper, get: () => state! };
}

describe('useDaemonHealth', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
    resolveServerUrl.mockResolvedValue('http://daemon');
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it('connected becomes true when /api/health returns ok', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true }));
    const { wrapper, get } = await mountHealth();
    await flushPromises();

    expect(get().connected.value).toBe(true);

    wrapper.unmount();
  });

  it('connected stays false when fetch throws', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('offline')));
    const { wrapper, get } = await mountHealth();
    await flushPromises();

    expect(get().connected.value).toBe(false);

    wrapper.unmount();
  });

  it('connected becomes false after FAILURE_THRESHOLD consecutive failures', async () => {
    // First ping succeeds, then three fail.
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValueOnce({ ok: true }).mockResolvedValue({ ok: false }),
    );
    const { wrapper, get } = await mountHealth();
    await flushPromises();
    expect(get().connected.value).toBe(true);

    // Advance 3 poll cycles (10s each).
    for (let i = 0; i < 3; i++) {
      await vi.advanceTimersByTimeAsync(10_000);
      await flushPromises();
    }

    expect(get().connected.value).toBe(false);

    wrapper.unmount();
  });

  it('cleans up the poll timer on unmount', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true }));
    const clearSpy = vi.spyOn(globalThis, 'clearInterval');

    const { wrapper } = await mountHealth();
    await flushPromises();
    wrapper.unmount();

    expect(clearSpy).toHaveBeenCalled();
  });

  it('lastChecked is updated on each ping', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true }));
    const { wrapper, get } = await mountHealth();
    await flushPromises();

    const first = get().lastChecked.value;
    expect(first).toBeInstanceOf(Date);

    await vi.advanceTimersByTimeAsync(10_000);
    await flushPromises();

    expect(get().lastChecked.value!.getTime()).toBeGreaterThanOrEqual(first!.getTime());

    wrapper.unmount();
  });
});
