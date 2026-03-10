import { ref, onMounted, onUnmounted } from 'vue'
import type { Task } from '../types/domain'
import { getQueue } from '../types/wails'

export function useTasks() {
  const tasks = ref<Task[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  let interval: ReturnType<typeof setInterval> | null = null

  async function refresh() {
    try {
      tasks.value = await getQueue()
      error.value = null
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load tasks'
    }
  }

  onMounted(async () => {
    loading.value = true
    await refresh()
    loading.value = false
    interval = setInterval(refresh, 2000)
  })

  onUnmounted(() => {
    if (interval) clearInterval(interval)
  })

  return { tasks, loading, error, refresh }
}
