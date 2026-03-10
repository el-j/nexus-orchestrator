import { ref, onMounted, onUnmounted } from 'vue'
import type { ProviderInfo } from '../types/domain'
import { getProviders } from '../types/wails'

export function useProviders() {
  const providers = ref<ProviderInfo[]>([])
  const loading = ref(false)
  let interval: ReturnType<typeof setInterval> | null = null

  async function refresh() {
    try {
      providers.value = await getProviders()
    } catch { /* silent fail */ }
  }

  onMounted(async () => {
    loading.value = true
    await refresh()
    loading.value = false
    interval = setInterval(refresh, 5000)
  })

  onUnmounted(() => {
    if (interval) clearInterval(interval)
  })

  return { providers, loading, refresh }
}
