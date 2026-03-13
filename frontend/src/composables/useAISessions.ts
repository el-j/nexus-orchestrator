import { ref, onMounted, onUnmounted } from 'vue'
import type { AISession } from '../types/domain'
import { listAISessions, deregisterAISession } from '../types/wails'
import { resolveServerUrl } from './useServerUrl'

export function useAISessions() {
  const sessions = ref<AISession[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  let interval: ReturnType<typeof setInterval> | null = null
  let eventSource: EventSource | null = null

  async function refresh() {
    try {
      sessions.value = (await listAISessions()) ?? []
      error.value = null
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load AI sessions'
    }
  }

  async function deregister(id: string) {
    await deregisterAISession(id)
    await refresh()
  }

  onMounted(async () => {
    loading.value = true
    await refresh()

    if (typeof EventSource !== 'undefined') {
      try {
        const baseUrl = await resolveServerUrl()
        eventSource = new EventSource(`${baseUrl}/api/events`)
        eventSource.onmessage = (event) => {
          const data = JSON.parse(event.data) as { type: string }
          if (data.type === 'ai_session_changed') {
            refresh()
          }
        }
        eventSource.onerror = () => {
          console.warn('SSE connection error — falling back to polling')
          eventSource?.close()
          eventSource = null
          interval = setInterval(refresh, 5000)
        }
      } catch {
        interval = setInterval(refresh, 5000)
      }
    } else {
      interval = setInterval(refresh, 5000)
    }

    loading.value = false
  })

  onUnmounted(() => {
    if (interval) clearInterval(interval)
    if (eventSource) eventSource.close()
  })

  return { sessions, loading, error, refresh, deregister }
}
