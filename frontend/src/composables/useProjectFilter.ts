import { computed, ref, onMounted, onUnmounted } from 'vue';
import { useTasks } from './useTasks';
import { currentProject, setProject } from './useProjectState';
import { getAllTasks } from '../types/wails';

export function useProjectFilter() {
  const { tasks } = useTasks();
  const allHistoryPaths = ref<string[]>([]);
  let historyInterval: ReturnType<typeof setInterval> | null = null;

  async function fetchHistoryPaths() {
    try {
      const all = await getAllTasks();
      allHistoryPaths.value = all.map((t) => t.projectPath).filter(Boolean) as string[];
    } catch {
      // non-fatal: live tasks still populate the list
    }
  }

  onMounted(() => {
    fetchHistoryPaths();
    historyInterval = setInterval(fetchHistoryPaths, 60_000);
  });

  onUnmounted(() => {
    if (historyInterval) clearInterval(historyInterval);
  });

  const projectList = computed<string[]>(() => {
    const paths = new Set([
      ...tasks.value.map((t) => t.projectPath).filter(Boolean),
      ...allHistoryPaths.value,
    ]);
    return Array.from(paths).sort();
  });

  return { currentProject, projectList, setProject };
}
