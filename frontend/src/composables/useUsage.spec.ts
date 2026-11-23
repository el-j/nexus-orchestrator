import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { defineComponent, ref, type Ref } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { useUsage } from './useUsage';
import * as wails from '../types/wails';

vi.mock('../types/wails', () => ({ getUsage: vi.fn() }));

const report = {
  since: '2026-10-17T00:00:00Z',
  totals: { provider: '', calls: 1, inputTokens: 10, outputTokens: 5 },
  usage: [],
};

function host(since?: Ref<string> | string, project?: Ref<string | null> | string | null) {
  let api!: ReturnType<typeof useUsage>;
  const w = mount(
    defineComponent({
      setup() {
        api = useUsage(since, project);
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
    expect(wails.getUsage).toHaveBeenCalledWith('24h', null);
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
    expect(wails.getUsage).toHaveBeenCalledWith('7d', null);
    expect(api().error.value).toBe('HTTP 503');
    await api().refresh();
    expect(api().error.value).toBeNull();
  });

  it('refreshes when a reactive timeframe ref changes', async () => {
    const timeframe = ref('24h');
    host(timeframe);
    await flushPromises();
    expect(wails.getUsage).toHaveBeenCalledWith('24h', null);

    timeframe.value = '7d';
    await flushPromises();
    expect(wails.getUsage).toHaveBeenCalledWith('7d', null);

    timeframe.value = 'all';
    await flushPromises();
    expect(wails.getUsage).toHaveBeenCalledWith('all', null);
  });

  it('refreshes when a reactive project ref changes or static project string is passed', async () => {
    const project = ref<string | null>(null);
    host('24h', project);
    await flushPromises();
    expect(wails.getUsage).toHaveBeenCalledWith('24h', null);

    project.value = '/projects/a';
    await flushPromises();
    expect(wails.getUsage).toHaveBeenCalledWith('24h', '/projects/a');

    // Static project string
    host('7d', '/projects/b');
    await flushPromises();
    expect(wails.getUsage).toHaveBeenCalledWith('7d', '/projects/b');
  });

  it('treats an answer that is not a usage report as an error', async () => {
    vi.mocked(wails.getUsage).mockResolvedValueOnce([] as never);
    const { api } = host();
    await flushPromises();
    expect(api().error.value).toBe('Unexpected usage answer');
    expect(api().report.value).toBeNull();
    vi.mocked(wails.getUsage).mockResolvedValueOnce(null as never);
    await api().refresh();
    expect(api().error.value).toBe('Unexpected usage answer');
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
