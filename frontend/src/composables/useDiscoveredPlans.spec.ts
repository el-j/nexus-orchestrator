import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { ref } from 'vue';
import { withSetup } from '../test/withSetup';
import { useDiscoveredPlans } from './useDiscoveredPlans';

vi.mock('./useServerUrl', () => ({ resolveServerUrl: () => Promise.resolve('http://d') }));

const fetchMock = vi.fn();
const json = (body: unknown, status = 200) =>
  Promise.resolve(new Response(JSON.stringify(body), { status }));
const flush = () => vi.advanceTimersByTimeAsync(0);

beforeEach(() => {
  vi.useFakeTimers();
  fetchMock.mockReset().mockImplementation(() => json([]));
  vi.stubGlobal('fetch', fetchMock);
});
afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe('useDiscoveredPlans', () => {
  it('loads plans on mount, for all projects when none is selected', async () => {
    fetchMock.mockImplementation(() => json([{ id: 'p1' }]));
    const { result } = withSetup(() => useDiscoveredPlans(ref(null)));
    await flush();
    expect(fetchMock).toHaveBeenCalledWith('http://d/api/plans/discovered');
    expect(result.plans.value).toEqual([{ id: 'p1' }]);
    expect(result.loading.value).toBe(false);
  });

  it('scopes the request to the selected project and reloads when it changes', async () => {
    const project = ref<string | null>('/a b');
    withSetup(() => useDiscoveredPlans(project));
    await flush();
    expect(fetchMock).toHaveBeenLastCalledWith(
      'http://d/api/plans/discovered?projectPath=%2Fa%20b',
    );
    project.value = '/c';
    await flush();
    expect(fetchMock).toHaveBeenLastCalledWith('http://d/api/plans/discovered?projectPath=%2Fc');
  });

  it('treats a null body as no plans (Go nil slice)', async () => {
    fetchMock.mockImplementation(() => json(null));
    const { result } = withSetup(() => useDiscoveredPlans(ref(null)));
    await flush();
    expect(result.plans.value).toEqual([]);
    expect(result.error.value).toBeNull();
  });

  it('reports a failed response and keeps the old plans', async () => {
    fetchMock.mockImplementationOnce(() => json([{ id: 'keep' }]));
    const { result } = withSetup(() => useDiscoveredPlans(ref(null)));
    await flush();
    fetchMock.mockImplementation(() => json({}, 503));
    await result.fetchPlans();
    expect(result.error.value).toBe('HTTP 503');
    expect(result.plans.value).toEqual([{ id: 'keep' }]);
  });

  it('clears the error after a successful reload', async () => {
    fetchMock.mockImplementationOnce(() => json({}, 500));
    const { result } = withSetup(() => useDiscoveredPlans(ref(null)));
    await flush();
    expect(result.error.value).toBe('HTTP 500');
    await result.fetchPlans();
    expect(result.error.value).toBeNull();
  });

  it('stringifies non-Error failures', async () => {
    fetchMock.mockRejectedValue('weird');
    const { result } = withSetup(() => useDiscoveredPlans(ref(null)));
    await flush();
    expect(result.error.value).toBe('weird');
    expect(result.loading.value).toBe(false);
  });

  it('scan() posts the scan then reloads', async () => {
    const project = ref<string | null>('/p');
    const { result } = withSetup(() => useDiscoveredPlans(project));
    await flush();
    fetchMock.mockClear();
    await result.scan();
    expect(fetchMock.mock.calls[0]).toEqual([
      'http://d/api/plans/discovered/scan?projectPath=%2Fp',
      { method: 'POST' },
    ]);
    expect(fetchMock.mock.calls[1][0]).toBe('http://d/api/plans/discovered?projectPath=%2Fp');
  });

  it('scan() without a project posts without a query', async () => {
    const { result } = withSetup(() => useDiscoveredPlans(ref(null)));
    await flush();
    fetchMock.mockClear();
    await result.scan();
    expect(fetchMock.mock.calls[0][0]).toBe('http://d/api/plans/discovered/scan');
  });

  it('scan() surfaces a failed scan without reloading', async () => {
    const { result } = withSetup(() => useDiscoveredPlans(ref(null)));
    await flush();
    fetchMock.mockClear();
    fetchMock.mockImplementationOnce(() => json({}, 500));
    await result.scan();
    expect(result.error.value).toBe('HTTP 500');
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(result.loading.value).toBe(false);
  });

  it('scan() reports a network error', async () => {
    const { result } = withSetup(() => useDiscoveredPlans(ref(null)));
    await flush();
    fetchMock.mockRejectedValueOnce(new Error('offline'));
    await result.scan();
    expect(result.error.value).toBe('offline');
  });

  it('polls every 30s and stops on unmount', async () => {
    const { unmount } = withSetup(() => useDiscoveredPlans(ref(null)));
    await flush();
    fetchMock.mockClear();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    unmount();
    fetchMock.mockClear();
    await vi.advanceTimersByTimeAsync(120_000);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('reports a non-Error failure of a scan as text', async () => {
    fetchMock.mockImplementation(() => Promise.reject('offline'));
    const { result } = withSetup(() => useDiscoveredPlans(ref(null)));
    await flush();
    await result.scan();
    expect(result.error.value).toBe('offline');
  });
});
