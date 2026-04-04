import { ref, onMounted, onUnmounted } from 'vue';
import type { ProviderInfo } from '../types/domain';
import { getProviders } from '../types/wails';

export function useProviders() {
  const providers = ref<ProviderInfo[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);
  let interval: ReturnType<typeof setInterval> | null = null;

  async function refresh() {
    try {
      providers.value = (await getProviders()) ?? [];
      error.value = null;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load providers';
    }
  }

  onMounted(async () => {
    loading.value = true;
    await refresh();
    loading.value = false;
    interval = setInterval(refresh, 30_000); // matches backend health cache TTL (30 s)
  });

  onUnmounted(() => {
    if (interval) clearInterval(interval);
  });

  return { providers, loading, error, refresh };
}
