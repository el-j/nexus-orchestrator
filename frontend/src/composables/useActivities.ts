import { ref, onMounted, onUnmounted, computed } from 'vue';
import type { AIActivity, ActivityType } from '../types/domain';
import { resolveServerUrl } from './useServerUrl';

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
  let eventSource: EventSource | null = null;

  async function refresh() {
    try {
      const baseUrl = await resolveServerUrl();
      const limit = options?.limit ?? 100;
      const since = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();
      const res = await fetch(
        `${baseUrl}/api/activities/timeline?limit=${limit}&since=${encodeURIComponent(since)}`,
      );
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

    if (typeof EventSource !== 'undefined') {
      try {
        const baseUrl = await resolveServerUrl();
        eventSource = new EventSource(`${baseUrl}/api/events`);
        eventSource.onmessage = (event) => {
          try {
            const data = JSON.parse(event.data) as { type: string; activity?: AIActivity };
            if (data.type === 'ai_activity_new' && data.activity) {
              activities.value = [data.activity, ...activities.value];
            }
          } catch {
            // ignore malformed SSE frames
          }
        };
        eventSource.onerror = () => {
          console.warn('SSE connection error — falling back to polling');
          eventSource?.close();
          eventSource = null;
          interval = setInterval(refresh, 5000);
        };
      } catch {
        interval = setInterval(refresh, 5000);
      }
    } else {
      interval = setInterval(refresh, 5000);
    }

    loading.value = false;
  });

  onUnmounted(() => {
    if (interval) clearInterval(interval);
    if (eventSource) eventSource.close();
  });

  return { activities, filtered, loading, error, refresh };
}
