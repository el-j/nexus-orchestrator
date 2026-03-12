import { ref, onMounted, onUnmounted } from 'vue'
import type { LogEntry } from '../types/domain'
import { resolveServerUrl } from './useServerUrl'

const MAX_LOGS = 2000

export function useLogs() {
  const logs = ref<LogEntry[]>([])
  const connected = ref(false)
  let es: EventSource | null = null

  async function fetchInitial() {
    try {
      const baseUrl = await resolveServerUrl()
      const res = await fetch(`${baseUrl}/api/logs`)
      if (res.ok) {
        const data: LogEntry[] = await res.json()
        logs.value = data
      }
    } catch (err) {
      // Daemon not running yet — non-fatal at startup.
      console.warn('useLogs: fetchInitial failed:', err)
    }
  }

  async function connect() {
    const baseUrl = await resolveServerUrl()
    es = new EventSource(`${baseUrl}/api/events`)
    es.addEventListener('log', (event: MessageEvent) => {
      try {
        const entry: LogEntry = JSON.parse(event.data as string)
        logs.value.push(entry)
        if (logs.value.length > MAX_LOGS) {
          logs.value.splice(0, logs.value.length - MAX_LOGS)
        }
      } catch (err) {
        console.warn('useLogs: malformed log event:', err)
      }
    })
    es.onopen = () => { connected.value = true }
    es.onerror = () => { connected.value = false }
  }

  function clear() {
    logs.value = []
  }

  onMounted(() => {
    void fetchInitial()
    void connect()
  })

  onUnmounted(() => {
    es?.close()
    es = null
  })

  return { logs, connected, clear }
}
