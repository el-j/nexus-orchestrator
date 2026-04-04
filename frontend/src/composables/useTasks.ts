import { ref, computed, onMounted, onUnmounted } from 'vue';
import type { Task } from '../types/domain';
import {
  getQueue,
  createDraft as wailsCreateDraft,
  promoteTask as wailsPromoteTask,
  updateTask as wailsUpdateTask,
  cancelTask as wailsCancelTask,
} from '../types/wails';
import { currentProject } from './useProjectState';
import { useGlobalSSE } from './useGlobalSSE';

export function useTasks() {
  const tasks = ref<Task[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);
  let interval: ReturnType<typeof setInterval> | null = null;

  const { on, off } = useGlobalSSE();

  function sseHandler(data: { type: string; [key: string]: unknown }) {
    if (data.type !== 'connected') refresh();
  }

  const queuedTasks = computed(() =>
    tasks.value.filter(
      (t) =>
        (t.status === 'QUEUED' || t.status === 'PROCESSING') &&
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

  async function promoteTask(id: string): Promise<void> {
    await wailsPromoteTask(id);
    await refresh();
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
    on('*', sseHandler);
    interval = setInterval(refresh, 15_000);
    loading.value = false;
  });

  onUnmounted(() => {
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
