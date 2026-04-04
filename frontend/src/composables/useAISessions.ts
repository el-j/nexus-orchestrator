import { ref, onMounted, onUnmounted } from 'vue';
import type { AISession } from '../types/domain';
import { listAISessions, deregisterAISession, purgeDisconnectedSessions } from '../types/wails';
import { useGlobalSSE } from './useGlobalSSE';

export function useAISessions() {
  const sessions = ref<AISession[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);
  let interval: ReturnType<typeof setInterval> | null = null;

  const { on, off } = useGlobalSSE();

  function sseHandler() {
    refresh();
  }

  async function refresh() {
    try {
      sessions.value = (await listAISessions()) ?? [];
      error.value = null;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load AI sessions';
    }
  }

  async function deregister(id: string) {
    await deregisterAISession(id);
    await refresh();
  }

  async function purgeDisconnected() {
    await purgeDisconnectedSessions();
    await refresh();
  }

  onMounted(async () => {
    loading.value = true;
    await refresh();
    on('ai_session_changed', sseHandler);
    interval = setInterval(refresh, 15_000);
    loading.value = false;
  });

  onUnmounted(() => {
    off('ai_session_changed', sseHandler);
    if (interval) clearInterval(interval);
  });

  return { sessions, loading, error, refresh, deregister, purgeDisconnected };
}
