import { ref, onMounted, onUnmounted, watch } from 'vue';
import type { LogEntry } from '../types/domain';
import { resolveServerUrl } from './useServerUrl';
import { useGlobalSSE } from './useGlobalSSE';

const MAX_LOGS = 2000;
const POLL_INTERVAL_MS = 3_000;
const DISCONNECTED_WARNING =
  'Realtime log stream disconnected. Polling fallback active while reconnecting.';

export function useLogs() {
  const logs = ref<LogEntry[]>([]);
  const error = ref<string | null>(null);
  let pollTimer: ReturnType<typeof setInterval> | null = null;

  const { connected, on, off } = useGlobalSSE();

  function append(entry: LogEntry) {
    logs.value.push(entry);
    if (logs.value.length > MAX_LOGS) {
      logs.value.splice(0, logs.value.length - MAX_LOGS);
    }
  }

  function handleLogEvent(data: { type: string; [key: string]: unknown }) {
    if (data.type !== 'log') return;
    const entry = {
      timestamp: data.timestamp,
      level: data.level,
      source: data.source,
      message: data.message,
    } as LogEntry;
    append(entry);
  }

  async function fetchLogs() {
    try {
      const baseUrl = await resolveServerUrl();
      const res = await fetch(`${baseUrl}/api/logs`);
      if (res.ok) {
        const data: LogEntry[] = await res.json();
        logs.value = data.slice(-MAX_LOGS);
        error.value = connected.value ? null : DISCONNECTED_WARNING;
      }
    } catch (err) {
      if (!connected.value) {
        error.value = DISCONNECTED_WARNING;
        return;
      }
      error.value = err instanceof Error ? err.message : 'Failed to load logs';
    }
  }

  function startPolling() {
    if (pollTimer) return;
    pollTimer = setInterval(() => {
      void fetchLogs();
    }, POLL_INTERVAL_MS);
  }

  function stopPolling() {
    if (!pollTimer) return;
    clearInterval(pollTimer);
    pollTimer = null;
  }

  function clear() {
    logs.value = [];
  }

  watch(connected, (isConnected) => {
    if (isConnected) {
      error.value = null;
      stopPolling();
      void fetchLogs();
      return;
    }
    error.value = DISCONNECTED_WARNING;
    startPolling();
    void fetchLogs();
  });

  onMounted(() => {
    on('log', handleLogEvent);
    void fetchLogs();
    if (!connected.value) {
      error.value = DISCONNECTED_WARNING;
      startPolling();
    }
  });

  onUnmounted(() => {
    off('log', handleLogEvent);
    stopPolling();
  });

  return { logs, connected, error, clear };
}
