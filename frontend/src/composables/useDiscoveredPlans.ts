import { ref, onMounted, onUnmounted, watch } from 'vue';
import type { Ref } from 'vue';
import type { DiscoveredPlanFile } from '../types/domain';
import { resolveServerUrl } from './useServerUrl';

/** Plan files discovered on the machine, with a promote action that queues them. */
export function useDiscoveredPlans(projectPath: Ref<string | null>) {
  const plans = ref<DiscoveredPlanFile[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  async function fetchPlans() {
    loading.value = true;
    try {
      const base = await resolveServerUrl();
      const params = projectPath.value
        ? `?projectPath=${encodeURIComponent(projectPath.value)}`
        : '';
      const res = await fetch(`${base}/api/plans/discovered${params}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      // A Go nil slice is serialised as JSON null.
      plans.value = ((await res.json()) as DiscoveredPlanFile[] | null) ?? [];
      error.value = null;
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  }

  async function scan() {
    loading.value = true;
    try {
      const base = await resolveServerUrl();
      const params = projectPath.value
        ? `?projectPath=${encodeURIComponent(projectPath.value)}`
        : '';
      const res = await fetch(`${base}/api/plans/discovered/scan${params}`, { method: 'POST' });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      await fetchPlans();
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  }

  let intervalId: ReturnType<typeof setInterval> | undefined;

  onMounted(() => {
    fetchPlans();
    intervalId = setInterval(fetchPlans, 30_000);
  });

  onUnmounted(() => {
    clearInterval(intervalId);
  });

  watch(projectPath, () => fetchPlans());

  return { plans, loading, error, fetchPlans, scan };
}
