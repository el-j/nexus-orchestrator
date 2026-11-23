import { ref, isRef, watch, onMounted, onUnmounted, type Ref } from 'vue';
import type { UsageReport } from '../types/domain';
import { getUsage } from '../types/wails';

/** Token usage of the last `since` (default 24h) from the daemon's usage ledger, refreshed every minute, optionally scoped to a project. */
export function useUsage(
  sinceParam: Ref<string> | string = '24h',
  projectParam?: Ref<string | null> | string | null,
) {
  const sinceRef = isRef(sinceParam) ? sinceParam : ref(sinceParam);
  const projectRef =
    projectParam === undefined
      ? ref<string | null>(null)
      : isRef(projectParam)
        ? projectParam
        : ref(projectParam);
  const report = ref<UsageReport | null>(null);
  const loading = ref(false);
  const error = ref<string | null>(null);
  let interval: ReturnType<typeof setInterval> | null = null;
  let disposed = false; // set on unmount so a fetch that finishes late starts nothing

  async function refresh() {
    try {
      const r = await getUsage(sinceRef.value, projectRef.value);
      // A daemon without the ledger, or a stand-in server, may answer with something else: show an error
      // instead of failing to render.
      if (!r || !r.totals || !Array.isArray(r.usage)) throw new Error('Unexpected usage answer');
      report.value = r;
      error.value = null;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load usage';
    }
  }

  watch([sinceRef, projectRef], async () => {
    loading.value = true;
    await refresh();
    loading.value = false;
  });

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
