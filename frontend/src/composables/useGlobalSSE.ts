import { ref } from 'vue';
import { resolveServerUrl } from './useServerUrl';

type SSEHandler = (data: { type: string; [key: string]: unknown }) => void;

const handlers = new Map<string, Set<SSEHandler>>();
let eventSource: EventSource | null = null;
const connected = ref(false);
let connecting = false;
let reconnectDelay = 3000;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;

function dispatch(data: { type: string; [key: string]: unknown }) {
  const globalSet = handlers.get('*');
  if (globalSet) globalSet.forEach((h) => h(data));

  const typeSet = handlers.get(data.type);
  if (typeSet) typeSet.forEach((h) => h(data));
}

function scheduleReconnect() {
  if (reconnectTimer) return;
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    void connect();
  }, reconnectDelay);
  reconnectDelay = Math.min(reconnectDelay * 2, 30_000);
}

async function connect() {
  if (eventSource || connecting) return;
  connecting = true;
  try {
    const baseUrl = await resolveServerUrl();
    eventSource = new EventSource(`${baseUrl}/api/events`);
    eventSource.onopen = () => {
      connected.value = true;
      reconnectDelay = 3000;
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
        reconnectTimer = null;
      }
    };
    eventSource.onmessage = (event) => {
      try {
        const raw = JSON.parse(event.data) as Record<string, unknown>;
        const type = typeof raw.type === 'string' ? raw.type : 'message';
        dispatch({ ...raw, type });
      } catch {
        // ignore malformed SSE frame
      }
    };
    eventSource.addEventListener('log', (event: MessageEvent) => {
      try {
        const raw = JSON.parse(event.data as string) as Record<string, unknown>;
        dispatch({ ...raw, type: 'log' });
      } catch {
        // ignore malformed SSE frame
      }
    });
    eventSource.onerror = () => {
      connected.value = false;
      eventSource?.close();
      eventSource = null;
      scheduleReconnect();
    };
  } catch {
    connected.value = false;
    scheduleReconnect();
  } finally {
    connecting = false;
  }
}

function disconnect() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  eventSource?.close();
  eventSource = null;
  connected.value = false;
  connecting = false;
}

export function useGlobalSSE() {
  function on(type: string, handler: SSEHandler) {
    if (!handlers.has(type)) handlers.set(type, new Set());
    handlers.get(type)!.add(handler);
  }

  function off(type: string, handler: SSEHandler) {
    handlers.get(type)?.delete(handler);
  }

  return { connected, on, off, connect, disconnect };
}
