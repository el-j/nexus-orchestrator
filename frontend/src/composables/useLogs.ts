import { ref, onMounted, onUnmounted } from 'vue'
import type { LogEntry } from '../types/domain'

const MAX_LOGS = 2000

export function useLogs() {
  const logs = ref<LogEntry[]>([])
  const connected = ref(false)
  let es: EventSource | null = null

  async function fetchInitial() {
    try {
      const res = await fetch('http://127.0.0.1:9999/api/logs')
      if (res.ok) {
        const data: LogEntry[] = await res.json()
        logs.value = data
      }
    } catch {
      // daemon not running yet
    }
  }

  function connect() {
    es = new EventSource('http://127.0.0.1:9999/api/events')
    es.addEventListener('log', (event: MessageEvent) => {
      try {
        const entry: LogEntry = JSON.parse(event.data)
        logs.value.push(entry)
        if (logs.value.length > MAX_LOGS) {
          logs.value.splice(0, logs.value.length - MAX_LOGS)
        }
      } catch { /* ignore malformed */ }
    })
    es.onopen = () => { connected.value = true }
    es.onerror = () => { connected.value = false }
  }

  function clear() {
    logs.value = []
  }

  onMounted(() => {
    fetchInitial()
    connect()
  })

  onUnmounted(() => {
    es?.close()
    es = null
  })

  return { logs, connected, clear }
}
