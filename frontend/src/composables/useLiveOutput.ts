import { reactive } from 'vue';
import { useGlobalSSE } from './useGlobalSSE';

/** Most characters kept per task; the newest text wins. */
export const MAX_LIVE_CHARS = 20_000;

/** Task events after which the live text is no longer needed: the result is in the task. */
const FINISHED = new Set([
  'task.completed',
  'task.failed',
  'task.cancelled',
  'task.too_large',
  'task.no_provider',
  'task.queued',
]);

const texts = reactive<Record<string, string>>({});
let listening = false;

function listen(): void {
  listening = true;
  const sse = useGlobalSSE();
  sse.on('*', (event) => {
    const id = typeof event.taskId === 'string' ? event.taskId : '';
    if (!id) return;
    if (event.type === 'task.restart') {
      texts[id] = '';
    } else if (event.type === 'task.chunk' && typeof event.text === 'string') {
      texts[id] = ((texts[id] ?? '') + event.text).slice(-MAX_LIVE_CHARS);
    } else if (FINISHED.has(event.type)) {
      delete texts[id];
    }
  });
  void sse.connect();
}

/**
 * The text a running task has produced so far, from the daemon's `task.chunk` events. A
 * `task.restart` clears it (a retry, the next provider or the next agent turn starts), and a
 * task that finished or was re-queued drops it: its result is in the task itself.
 */
export function useLiveOutput(): { liveText: (taskId: string) => string } {
  if (!listening) listen();
  return { liveText: (taskId) => texts[taskId] ?? '' };
}
