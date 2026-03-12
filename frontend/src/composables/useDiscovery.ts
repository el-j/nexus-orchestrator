import { ref, onMounted, onUnmounted } from 'vue'
import type { DiscoveredProvider } from '../types/discovery'
import { getDiscoveredProviders, triggerScan } from '../types/wails'
import { resolveServerUrl } from './useServerUrl'

export function useDiscovery() {
  const discovered = ref<DiscoveredProvider[]>([])
  const loading = ref(false)
  const scanning = ref(false)
  let eventSource: EventSource | null = null
  let interval: ReturnType<typeof setInterval> | null = null

  async function refresh() {
    try {
      discovered.value = await getDiscoveredProviders()
    } catch (e) { console.warn('useDiscovery: refresh failed:', e) }
  }

  async function scanNow() {
    scanning.value = true
    try {
      await triggerScan()
      // Give the scanner a moment, then refresh
      await new Promise(r => setTimeout(r, 1500))
      await refresh()
    } catch (e) { console.warn('useDiscovery: scan failed:', e) } finally {
      scanning.value = false
    }
  }

  function connectSSE(baseUrl: string): boolean {
    if (typeof EventSource === 'undefined') return false
    try {
      eventSource = new EventSource(`${baseUrl}/api/events`)
      eventSource.addEventListener('provider_discovered', () => {
        refresh()
      })
      eventSource.onerror = () => {
        eventSource?.close()
        eventSource = null
        // Fall back to polling
        if (!interval) {
          interval = setInterval(refresh, 10000)
        }
      }
      return true
    } catch {
      return false
    }
  }

  onMounted(async () => {
    loading.value = true
    await refresh()
    loading.value = false
    let baseUrl = 'http://127.0.0.1:9999'
    try { baseUrl = await resolveServerUrl() } catch { /* use default */ }
    if (!connectSSE(baseUrl)) {
      interval = setInterval(refresh, 10000)
    }
  })

  onUnmounted(() => {
    if (eventSource) eventSource.close()
    if (interval) clearInterval(interval)
  })

  return { discovered, loading, scanning, refresh, scanNow }
}
