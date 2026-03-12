import { ref, computed, onMounted, onUnmounted } from 'vue'
import type { Task } from '../types/domain'
import { getQueue, createDraft as wailsCreateDraft, promoteTask as wailsPromoteTask, updateTask as wailsUpdateTask } from '../types/wails'
import { currentProject } from './useProjectState'
import { resolveServerUrl } from './useServerUrl'

export function useTasks() {
  const tasks = ref<Task[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  let interval: ReturnType<typeof setInterval> | null = null
  let eventSource: EventSource | null = null

  const backlogTasks = computed(() =>
    tasks.value.filter(t =>
      (t.status === 'DRAFT' || t.status === 'BACKLOG') &&
      (currentProject.value === null || t.projectPath === currentProject.value)
    )
  )

  const queuedTasks = computed(() =>
    tasks.value.filter(t =>
      (t.status === 'QUEUED' || t.status === 'PROCESSING') &&
      (currentProject.value === null || t.projectPath === currentProject.value)
    )
  )

  async function refresh() {
    try {
      tasks.value = await getQueue()
      error.value = null
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load tasks'
    }
  }

  async function createDraft(task: Partial<Task>): Promise<string> {
    const id = await wailsCreateDraft(task)
    await refresh()
    return id
  }

  async function promoteTask(id: string): Promise<void> {
    await wailsPromoteTask(id)
    await refresh()
  }

  async function updateTask(id: string, updates: Partial<Task>): Promise<Task> {
    const updated = await wailsUpdateTask(id, updates)
    await refresh()
    return updated
  }

  onMounted(async () => {
    loading.value = true
    await refresh()

    if (typeof EventSource !== 'undefined') {
      try {
        const baseUrl = await resolveServerUrl()
        eventSource = new EventSource(`${baseUrl}/api/events`)
        eventSource.onmessage = (event) => {
          const data = JSON.parse(event.data)
          if (data.type !== 'connected') {
            refresh()
          }
        }
        eventSource.onerror = () => {
          console.warn('SSE connection error — falling back to polling')
          eventSource?.close()
          eventSource = null
          interval = setInterval(refresh, 2000)
        }
      } catch {
        // EventSource unavailable or failed to connect — fall back to polling
        interval = setInterval(refresh, 2000)
      }
    } else {
      interval = setInterval(refresh, 2000)
    }

    loading.value = false
  })

  onUnmounted(() => {
    if (interval) clearInterval(interval)
    if (eventSource) eventSource.close()
  })

  return { tasks, loading, error, refresh, backlogTasks, queuedTasks, createDraft, promoteTask, updateTask }
}