import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { flushPromises } from '@vue/test-utils';
import { withSetup } from '../test/withSetup';

const { getDiscoveredProviders, triggerScan, on, off } = vi.hoisted(() => ({
  getDiscoveredProviders: vi.fn(),
  triggerScan: vi.fn(),
  on: vi.fn(),
  off: vi.fn(),
}));
vi.mock('../types/wails', () => ({ getDiscoveredProviders, triggerScan }));
vi.mock('./useGlobalSSE', () => ({ useGlobalSSE: () => ({ on, off }) }));
import { useDiscovery } from './useDiscovery';

describe('useDiscovery', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    [getDiscoveredProviders, triggerScan, on, off].forEach((m) => m.mockReset());
    getDiscoveredProviders.mockResolvedValue([{ id: 'd1' }]);
    triggerScan.mockResolvedValue(undefined);
  });
  afterEach(() => vi.useRealTimers());

  it('loads on mount, subscribes to SSE and polls every 15 seconds', async () => {
    const { result, unmount } = withSetup(() => useDiscovery());
    await flushPromises();
    expect(result.discovered.value).toEqual([{ id: 'd1' }]);
    expect(result.loading.value).toBe(false);
    expect(on).toHaveBeenCalledWith('provider_discovered', expect.any(Function));

    await vi.advanceTimersByTimeAsync(15_000);
    expect(getDiscoveredProviders).toHaveBeenCalledTimes(2);

    // The SSE handler refreshes immediately.
    const handler = on.mock.calls[0][1] as () => void;
    handler();
    await flushPromises();
    expect(getDiscoveredProviders).toHaveBeenCalledTimes(3);

    unmount();
    expect(off).toHaveBeenCalledWith('provider_discovered', handler);
    await vi.advanceTimersByTimeAsync(60_000);
    expect(getDiscoveredProviders).toHaveBeenCalledTimes(3);
  });

  it('treats a null answer as an empty list and records fetch errors', async () => {
    getDiscoveredProviders.mockResolvedValueOnce(null);
    const { result, unmount } = withSetup(() => useDiscovery());
    await flushPromises();
    expect(result.discovered.value).toEqual([]);
    getDiscoveredProviders.mockRejectedValueOnce(new Error('daemon down'));
    await result.refresh();
    expect(result.error.value).toContain('daemon down');
    getDiscoveredProviders.mockResolvedValueOnce([]);
    await result.refresh();
    expect(result.error.value).toBeNull();
    unmount();
  });

  it('scanNow triggers a scan, waits for the scanner, then refreshes', async () => {
    const { result, unmount } = withSetup(() => useDiscovery());
    await flushPromises();
    const done = result.scanNow();
    await flushPromises();
    expect(result.scanning.value).toBe(true);
    expect(triggerScan).toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(1500);
    await done;
    expect(result.scanning.value).toBe(false);
    expect(getDiscoveredProviders).toHaveBeenCalledTimes(2);
    unmount();
  });

  it('scanNow reports a failed scan and still clears the scanning flag', async () => {
    triggerScan.mockRejectedValue(new Error('scan failed'));
    const { result, unmount } = withSetup(() => useDiscovery());
    await flushPromises();
    await result.scanNow();
    expect(result.error.value).toContain('scan failed');
    expect(result.scanning.value).toBe(false);
    unmount();
  });
});
