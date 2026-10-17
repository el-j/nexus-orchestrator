import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { defineComponent } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { useUsage } from './useUsage';
import * as wails from '../types/wails';

vi.mock('../types/wails', () => ({ getUsage: vi.fn() }));

const report = {
  since: '2026-10-17T00:00:00Z',
  totals: { provider: '', calls: 1, inputTokens: 10, outputTokens: 5 },
  usage: [],
};

function host(since?: string) {
  let api!: ReturnType<typeof useUsage>;
  const w = mount(
    defineComponent({
      setup() {
        api = useUsage(since);
        return () => null;
      },
    }),
  );
  return { w, api: () => api };
}

describe('useUsage', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.mocked(wails.getUsage).mockReset().mockResolvedValue(report);
  });
  afterEach(() => vi.useRealTimers());

  it('loads the report on mount, asks for 24h by default and refreshes every minute', async () => {
    const { w, api } = host();
    await flushPromises();
    expect(wails.getUsage).toHaveBeenCalledWith('24h');
    expect(api().report.value).toEqual(report);
    expect(api().loading.value).toBe(false);
    await vi.advanceTimersByTimeAsync(60_000);
    expect(wails.getUsage).toHaveBeenCalledTimes(2);
    w.unmount();
    await vi.advanceTimersByTimeAsync(120_000);
    expect(wails.getUsage).toHaveBeenCalledTimes(2);
  });

  it('passes a custom window and keeps the message of a failure', async () => {
    vi.mocked(wails.getUsage).mockRejectedValueOnce(new Error('HTTP 503'));
    const { api } = host('7d');
    await flushPromises();
    expect(wails.getUsage).toHaveBeenCalledWith('7d');
    expect(api().error.value).toBe('HTTP 503');
    await api().refresh();
    expect(api().error.value).toBeNull();
  });

  it('describes an error that is not an Error', async () => {
    vi.mocked(wails.getUsage).mockRejectedValueOnce('boom');
    const { api } = host();
    await flushPromises();
    expect(api().error.value).toBe('Failed to load usage');
  });

  it('starts no timer when it was unmounted while the first fetch was running', async () => {
    let finish!: (r: typeof report) => void;
    vi.mocked(wails.getUsage).mockReturnValueOnce(new Promise((r) => (finish = r)));
    const { w } = host();
    w.unmount();
    finish(report);
    await flushPromises();
    await vi.advanceTimersByTimeAsync(120_000);
    expect(wails.getUsage).toHaveBeenCalledTimes(1);
  });
});
