import { ref, onMounted, onUnmounted } from 'vue';
import type { DiscoveredProvider } from '../types/discovery';
import { getDiscoveredProviders, triggerScan } from '../types/wails';
import { useGlobalSSE } from './useGlobalSSE';

export function useDiscovery() {
  const discovered = ref<DiscoveredProvider[]>([]);
  const loading = ref(false);
  const scanning = ref(false);
  const error = ref<string | null>(null);
  let interval: ReturnType<typeof setInterval> | null = null;

  const { on, off } = useGlobalSSE();

  function sseHandler() {
    refresh();
  }

  async function refresh() {
    try {
      discovered.value = (await getDiscoveredProviders()) ?? [];
      error.value = null;
    } catch (e) {
      error.value = String(e);
    }
  }

  async function scanNow() {
    scanning.value = true;
    try {
      await triggerScan();
      // Give the scanner a moment, then refresh
      await new Promise((r) => setTimeout(r, 1500));
      await refresh();
    } catch (e) {
      error.value = String(e);
    } finally {
      scanning.value = false;
    }
  }

  onMounted(async () => {
    loading.value = true;
    await refresh();
    on('provider_discovered', sseHandler);
    interval = setInterval(refresh, 15_000);
    loading.value = false;
  });

  onUnmounted(() => {
    off('provider_discovered', sseHandler);
    if (interval) clearInterval(interval);
  });

  return { discovered, loading, scanning, error, refresh, scanNow };
}
