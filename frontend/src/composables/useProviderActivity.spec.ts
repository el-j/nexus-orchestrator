import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { flushPromises } from '@vue/test-utils';
import { withSetup } from '../test/withSetup';

vi.mock('./useServerUrl', () => ({ resolveServerUrl: vi.fn().mockResolvedValue('http://daemon') }));
import { useProviderActivity } from './useProviderActivity';

const fetchMock = vi.fn();
const NOW = new Date('2026-03-01T12:00:00Z');

describe('useProviderActivity', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(NOW);
    fetchMock.mockReset();
    vi.stubGlobal('fetch', fetchMock);
  });
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it('groups generation activities by provider and flags recently seen ones as active', async () => {
    const recent = new Date(NOW.getTime() - 5_000).toISOString();
    const old = new Date(NOW.getTime() - 5 * 60_000).toISOString();
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => [
        { agentName: 'ollama', model: 'llama3', timestamp: old },
        { agentName: 'ollama', model: 'qwen', timestamp: recent },
        { agentName: 'ollama', model: 'llama3', timestamp: old }, // duplicate model
        { agentName: 'lmstudio', timestamp: old }, // no model reported
      ],
    });
    const { result, unmount } = withSetup(() => useProviderActivity());
    await flushPromises();

    expect(fetchMock).toHaveBeenCalledWith('http://daemon/api/activities?type=generation&limit=50');
    const byName = Object.fromEntries(result.providerStates.value.map((p) => [p.provider, p]));
    expect(byName.ollama.models.sort()).toEqual(['llama3', 'qwen']);
    expect(byName.ollama.active).toBe(true);
    expect(byName.ollama.lastSeen).toBe(recent);
    expect(byName.lmstudio.models).toEqual([]);
    expect(byName.lmstudio.active).toBe(false);
    expect(result.error.value).toBeNull();
    unmount();
  });

  it('keeps the previous state on a non-ok answer and reports thrown errors', async () => {
    fetchMock.mockResolvedValueOnce({ ok: false });
    const { result, unmount } = withSetup(() => useProviderActivity());
    await flushPromises();
    expect(result.providerStates.value).toEqual([]);
    expect(result.error.value).toBeNull();

    fetchMock.mockRejectedValueOnce(new Error('offline'));
    await result.refresh();
    expect(result.error.value).toContain('offline');
    unmount();
  });

  it('refreshes every 15 seconds until unmounted', async () => {
    fetchMock.mockResolvedValue({ ok: true, json: async () => [] });
    const { unmount } = withSetup(() => useProviderActivity());
    await flushPromises();
    await vi.advanceTimersByTimeAsync(15_000);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    unmount();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
