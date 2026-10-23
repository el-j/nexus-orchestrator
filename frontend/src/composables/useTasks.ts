import { ref, computed, onMounted, onUnmounted } from 'vue';
import type { Task } from '../types/domain';
import {
  getQueue,
  createDraft as wailsCreateDraft,
  promoteTask as wailsPromoteTask,
  updateTask as wailsUpdateTask,
  cancelTask as wailsCancelTask,
  type PromoteResult,
} from '../types/wails';
import { currentProject } from './useProjectState';
import { useGlobalSSE } from './useGlobalSSE';

/** Task queue and history, refreshed on task.* SSE events. */
export function useTasks() {
  const tasks = ref<Task[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);
  let interval: ReturnType<typeof setInterval> | null = null;
  let disposed = false; // set on unmount so a fetch that finishes late starts nothing

  const { on, off } = useGlobalSSE();

  function sseHandler(data: { type: string; [key: string]: unknown }) {
    // Only task lifecycle events ("task.queued", "task.completed", …) change the
    // queue. The stream also carries every log line and activity event, and
    // refetching the queue for each of those would hammer the daemon.
    if (data.type.startsWith('task.')) void refresh();
  }

  const queuedTasks = computed(() =>
    tasks.value.filter(
      (t) =>
        (t.status === 'QUEUED' || t.status === 'WAITING' || t.status === 'PROCESSING') &&
        (currentProject.value === null || t.projectPath === currentProject.value),
    ),
  );

  async function refresh() {
    try {
      tasks.value = (await getQueue()) ?? [];
      error.value = null;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load tasks';
    }
  }

  async function createDraft(task: Partial<Task>): Promise<string> {
    const id = await wailsCreateDraft(task);
    await refresh();
    return id;
  }

  async function promoteTask(id: string): Promise<PromoteResult> {
    const res = await wailsPromoteTask(id);
    await refresh();
    return res;
  }

  async function cancelTask(id: string): Promise<void> {
    await wailsCancelTask(id);
    await refresh();
  }

  async function updateTask(id: string, updates: Partial<Task>): Promise<Task> {
    const updated = await wailsUpdateTask(id, updates);
    await refresh();
    return updated;
  }

  onMounted(async () => {
    loading.value = true;
    await refresh();
    loading.value = false;
    if (disposed) return;
    on('*', sseHandler);
    interval = setInterval(refresh, 15_000);
  });

  onUnmounted(() => {
    disposed = true;
    off('*', sseHandler);
    if (interval) clearInterval(interval);
  });

  return {
    tasks,
    loading,
    error,
    refresh,
    queuedTasks,
    createDraft,
    promoteTask,
    cancelTask,
    updateTask,
  };
}
