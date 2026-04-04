import { ref, onMounted, onUnmounted } from 'vue';
import { resolveServerUrl } from './useServerUrl';

const POLL_INTERVAL_MS = 10_000;
const FETCH_TIMEOUT_MS = 3_000;
const FAILURE_THRESHOLD = 3;

export function useDaemonHealth() {
  const connected = ref(false);
  const lastChecked = ref<Date | null>(null);
  let timer: ReturnType<typeof setInterval> | null = null;
  let consecutiveFailures = 0;

  async function ping() {
    lastChecked.value = new Date();
    const controller = new AbortController();
    const id = setTimeout(() => controller.abort(), FETCH_TIMEOUT_MS);
    try {
      const baseUrl = await resolveServerUrl();
      const res = await fetch(`${baseUrl}/health`, { signal: controller.signal });
      if (res.ok) {
        consecutiveFailures = 0;
        connected.value = true;
        return;
      }
      consecutiveFailures += 1;
      if (consecutiveFailures >= FAILURE_THRESHOLD) {
        connected.value = false;
      }
    } catch {
      consecutiveFailures += 1;
      if (consecutiveFailures >= FAILURE_THRESHOLD) {
        connected.value = false;
      }
    } finally {
      clearTimeout(id);
    }
  }

  onMounted(() => {
    void ping();
    timer = setInterval(() => {
      void ping();
    }, POLL_INTERVAL_MS);
  });

  onUnmounted(() => {
    if (timer) clearInterval(timer);
  });

  return { connected, lastChecked };
}
