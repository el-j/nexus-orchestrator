import { ref } from 'vue';
import { resolveServerUrl } from './useServerUrl';

type SSEHandler = (data: { type: string; [key: string]: unknown }) => void;

const handlers = new Map<string, Set<SSEHandler>>();
let eventSource: EventSource | null = null;
const connected = ref(false);
let reconnectDelay = 3000;

async function connect() {
  if (eventSource) return;
  const baseUrl = await resolveServerUrl();
  eventSource = new EventSource(`${baseUrl}/api/events`);
  eventSource.onopen = () => {
    connected.value = true;
    reconnectDelay = 3000;
  };
  eventSource.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data) as { type: string };
      // Notify all global handlers
      const globalSet = handlers.get('*');
      if (globalSet) globalSet.forEach((h) => h(data));
      // Notify type-specific handlers
      const typeSet = handlers.get(data.type);
      if (typeSet) typeSet.forEach((h) => h(data));
    } catch {
      console.warn('[SSE] Failed to parse event:', event.data);
    }
  };
  eventSource.onerror = () => {
    connected.value = false;
    eventSource?.close();
    eventSource = null;
    setTimeout(connect, reconnectDelay);
    reconnectDelay = Math.min(reconnectDelay * 2, 30_000);
  };
}

function disconnect() {
  eventSource?.close();
  eventSource = null;
  connected.value = false;
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
