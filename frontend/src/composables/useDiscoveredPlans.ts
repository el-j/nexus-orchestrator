import { ref, onMounted } from 'vue';
import type { Ref } from 'vue';
import type { DiscoveredPlanFile } from '../types/domain';
import { resolveServerUrl } from './useServerUrl';

export function useDiscoveredPlans(projectPath: Ref<string>) {
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
      if (res.ok) plans.value = (await res.json()) as DiscoveredPlanFile[];
    } catch (e) {
      error.value = String(e);
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
      await fetch(`${base}/api/plans/discovered/scan${params}`, { method: 'POST' });
      await fetchPlans();
    } catch (e) {
      error.value = String(e);
    } finally {
      loading.value = false;
    }
  }

  onMounted(() => {
    fetchPlans();
    setInterval(fetchPlans, 30_000);
  });

  return { plans, loading, error, fetchPlans, scan };
}
