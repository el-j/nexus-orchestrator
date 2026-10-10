import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { flushPromises } from '@vue/test-utils';
import { withSetup } from '../test/withSetup';

const { liveTasks, getAllTasks } = vi.hoisted(() => ({
  liveTasks: { value: [] as Array<{ projectPath: string }> },
  getAllTasks: vi.fn(),
}));
vi.mock('./useTasks', () => ({ useTasks: () => ({ tasks: liveTasks }) }));
vi.mock('../types/wails', () => ({ getAllTasks }));
import { useProjectFilter } from './useProjectFilter';

describe('useProjectFilter', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    liveTasks.value = [];
    getAllTasks.mockReset();
  });
  afterEach(() => vi.useRealTimers());

  it('merges live and historical project paths, de-duplicated and sorted', async () => {
    liveTasks.value = [{ projectPath: '/b' }, { projectPath: '/a' }, { projectPath: '' }];
    getAllTasks.mockResolvedValue([
      { projectPath: '/c' },
      { projectPath: '/a' },
      { projectPath: '' },
    ]);
    const { result, unmount } = withSetup(() => useProjectFilter());
    await flushPromises();
    expect(result.projectList.value).toEqual(['/a', '/b', '/c']);
    unmount();
  });

  it('survives a failing history fetch (live tasks still populate the list)', async () => {
    liveTasks.value = [{ projectPath: '/live' }];
    getAllTasks.mockRejectedValue(new Error('offline'));
    const { result, unmount } = withSetup(() => useProjectFilter());
    await flushPromises();
    expect(result.projectList.value).toEqual(['/live']);
    unmount();
  });

  it('re-reads history every minute until unmounted, and exposes the shared project state', async () => {
    getAllTasks.mockResolvedValue([]);
    const { result, unmount } = withSetup(() => useProjectFilter());
    await flushPromises();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(getAllTasks).toHaveBeenCalledTimes(2);
    result.setProject('/picked');
    expect(result.currentProject.value).toBe('/picked');
    result.setProject(null);
    expect(result.currentProject.value).toBeNull();
    unmount();
    await vi.advanceTimersByTimeAsync(120_000);
    expect(getAllTasks).toHaveBeenCalledTimes(2);
  });
});
