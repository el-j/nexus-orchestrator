import { ref, onMounted, onUnmounted, computed } from 'vue';
import type { AIActivity, ActivityType } from '../types/domain';
import { resolveServerUrl } from './useServerUrl';
import { useGlobalSSE } from './useGlobalSSE';

export function useActivities(options?: {
  agentFilter?: string;
  projectFilter?: string;
  typeFilter?: ActivityType;
  limit?: number;
}) {
  const activities = ref<AIActivity[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);
  let interval: ReturnType<typeof setInterval> | null = null;

  const { on, off } = useGlobalSSE();

  function sseHandler(data: { type: string; [key: string]: unknown }) {
    if (data.type === 'ai_activity_new' && data.activity) {
      activities.value = [data.activity as AIActivity, ...activities.value];
    }
  }

  async function refresh() {
    try {
      const baseUrl = await resolveServerUrl();
      const limit = options?.limit ?? 100;
      const since = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();
      const params = new URLSearchParams();
      params.set('limit', String(limit));
      params.set('since', since);
      if (options?.projectFilter) {
        params.set('projectPath', options.projectFilter);
      }
      const res = await fetch(`${baseUrl}/api/activities/timeline?${params.toString()}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = (await res.json()) as AIActivity[];
      activities.value = data ?? [];
      error.value = null;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load activities';
    }
  }

  const filtered = computed(() => {
    let result = activities.value;
    if (options?.agentFilter) {
      result = result.filter((a) => a.agentName === options.agentFilter);
    }
    if (options?.projectFilter) {
      result = result.filter((a) => a.projectPath === options.projectFilter);
    }
    if (options?.typeFilter) {
      result = result.filter((a) => a.activityType === options.typeFilter);
    }
    return result;
  });

  onMounted(async () => {
    loading.value = true;
    await refresh();
    on('ai_activity_new', sseHandler);
    interval = setInterval(refresh, 15_000);
    loading.value = false;
  });

  onUnmounted(() => {
    off('ai_activity_new', sseHandler);
    if (interval) clearInterval(interval);
  });

  return { activities, filtered, loading, error, refresh };
}
