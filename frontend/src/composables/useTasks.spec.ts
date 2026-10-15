import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { withSetup } from '../test/withSetup';
import type { Task } from '../types/domain';

const h = vi.hoisted(() => ({
  getQueue: vi.fn(),
  createDraft: vi.fn(),
  promoteTask: vi.fn(),
  updateTask: vi.fn(),
  cancelTask: vi.fn(),
  on: vi.fn(),
  off: vi.fn(),
  project: { value: null as string | null },
}));
vi.mock('../types/wails', () => ({
  getQueue: h.getQueue,
  createDraft: h.createDraft,
  promoteTask: h.promoteTask,
  updateTask: h.updateTask,
  cancelTask: h.cancelTask,
}));
vi.mock('./useGlobalSSE', () => ({ useGlobalSSE: () => ({ on: h.on, off: h.off }) }));
vi.mock('./useProjectState', async () => {
  const { ref } = await import('vue');
  return { currentProject: ref(null) };
});

import { currentProject } from './useProjectState';
import { useTasks } from './useTasks';

const task = (o: Partial<Task> = {}): Task =>
  ({ id: 't', projectPath: '/p', status: 'QUEUED', ...o }) as Task;
const flush = () => vi.advanceTimersByTimeAsync(0);

beforeEach(() => {
  vi.useFakeTimers();
  Object.values(h).forEach((f) => typeof f === 'function' && f.mockReset());
  h.getQueue.mockResolvedValue([]);
  (currentProject as unknown as { value: string | null }).value = null;
});
afterEach(() => vi.useRealTimers());

describe('useTasks', () => {
  it('loads the queue on mount and clears loading', async () => {
    h.getQueue.mockResolvedValue([task({ id: 'a' })]);
    const { result } = withSetup(() => useTasks());
    expect(result.loading.value).toBe(true);
    await flush();
    expect(result.tasks.value.map((t) => t.id)).toEqual(['a']);
    expect(result.loading.value).toBe(false);
    expect(result.error.value).toBeNull();
  });

  it('treats a null queue as empty', async () => {
    h.getQueue.mockResolvedValue(null);
    const { result } = withSetup(() => useTasks());
    await flush();
    expect(result.tasks.value).toEqual([]);
  });

  it('records the error message, a generic one for non-Errors, and recovers', async () => {
    h.getQueue.mockRejectedValueOnce(new Error('down'));
    const { result } = withSetup(() => useTasks());
    await flush();
    expect(result.error.value).toBe('down');
    h.getQueue.mockRejectedValueOnce('weird');
    await result.refresh();
    expect(result.error.value).toBe('Failed to load tasks');
    await result.refresh();
    expect(result.error.value).toBeNull();
  });

  it('polls every 15s and stops on unmount', async () => {
    const { unmount } = withSetup(() => useTasks());
    await flush();
    h.getQueue.mockClear();
    await vi.advanceTimersByTimeAsync(15_000);
    expect(h.getQueue).toHaveBeenCalledTimes(1);
    unmount();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(h.getQueue).toHaveBeenCalledTimes(1);
    expect(h.off).toHaveBeenCalledWith('*', expect.any(Function));
  });

  it('refreshes on task events only, not on logs, activity or the hello frame', async () => {
    withSetup(() => useTasks());
    await flush();
    const handler = h.on.mock.calls.find((c) => c[0] === '*')![1] as (d: { type: string }) => void;
    h.getQueue.mockClear();
    for (const type of ['log', 'ai_activity_new', 'connected', 'message']) handler({ type });
    await flush();
    expect(h.getQueue).not.toHaveBeenCalled();
    handler({ type: 'task.completed' });
    await flush();
    expect(h.getQueue).toHaveBeenCalledTimes(1);
  });

  it('queuedTasks keeps only queued and processing tasks of the selected project', async () => {
    h.getQueue.mockResolvedValue([
      task({ id: '1', status: 'QUEUED', projectPath: '/a' }),
      task({ id: '2', status: 'PROCESSING', projectPath: '/b' }),
      task({ id: '3', status: 'COMPLETED', projectPath: '/a' }),
    ]);
    const { result } = withSetup(() => useTasks());
    await flush();
    expect(result.queuedTasks.value.map((t) => t.id)).toEqual(['1', '2']);
    (currentProject as unknown as { value: string | null }).value = '/a';
    expect(result.queuedTasks.value.map((t) => t.id)).toEqual(['1']);
  });

  it('mutations call the API and then refresh', async () => {
    h.createDraft.mockResolvedValue('d1');
    h.updateTask.mockResolvedValue(task({ id: 'u' }));
    const { result } = withSetup(() => useTasks());
    await flush();

    h.getQueue.mockClear();
    expect(await result.createDraft({ instruction: 'x' })).toBe('d1');
    await result.promoteTask('p');
    await result.cancelTask('c');
    expect((await result.updateTask('u', { priority: 1 })).id).toBe('u');
    expect(h.createDraft).toHaveBeenCalledWith({ instruction: 'x' });
    expect(h.promoteTask).toHaveBeenCalledWith('p');
    expect(h.cancelTask).toHaveBeenCalledWith('c');
    expect(h.updateTask).toHaveBeenCalledWith('u', { priority: 1 });
    expect(h.getQueue).toHaveBeenCalledTimes(4);
  });

  it('does not refresh when a mutation fails, and rethrows', async () => {
    h.cancelTask.mockRejectedValue(new Error('nope'));
    const { result } = withSetup(() => useTasks());
    await flush();
    h.getQueue.mockClear();
    await expect(result.cancelTask('c')).rejects.toThrow('nope');
    expect(h.getQueue).not.toHaveBeenCalled();
  });
});
