import { ref, onMounted, onUnmounted } from 'vue';
import type { UsageReport } from '../types/domain';
import { getUsage } from '../types/wails';

/** Token usage of the last `since` (default 24h) from the daemon's usage ledger, refreshed every minute. */
export function useUsage(since = '24h') {
  const report = ref<UsageReport | null>(null);
  const loading = ref(false);
  const error = ref<string | null>(null);
  let interval: ReturnType<typeof setInterval> | null = null;
  let disposed = false; // set on unmount so a fetch that finishes late starts nothing

  async function refresh() {
    try {
      const r = await getUsage(since);
      // A daemon without the ledger, or a stand-in server, may answer with something else: show an error
      // instead of failing to render.
      if (!r || !r.totals || !Array.isArray(r.usage)) throw new Error('Unexpected usage answer');
      report.value = r;
      error.value = null;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load usage';
    }
  }

  onMounted(async () => {
    loading.value = true;
    await refresh();
    loading.value = false;
    if (disposed) return;
    interval = setInterval(refresh, 60_000);
  });

  onUnmounted(() => {
    disposed = true;
    if (interval) clearInterval(interval);
  });

  return { report, loading, error, refresh };
}
