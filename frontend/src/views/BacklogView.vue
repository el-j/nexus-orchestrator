<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <header class="flex items-center justify-between px-5 py-3 border-b border-white/5 bg-[#0a0a10] shrink-0">
      <div>
        <h1 class="text-sm font-bold text-white">Backlog</h1>
        <p class="text-xs text-slate-500">
          <span class="text-white font-semibold">{{ backlogTasks.length }}</span>
          {{ backlogTasks.length === 1 ? 'item' : 'items' }}
        </p>
      </div>
    </header>

    <!-- List (scrollable) -->
    <div class="flex-1 overflow-auto p-4">
      <BacklogList :items="backlogTasks" @promoted="refresh" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import type { Task } from '../types/domain'
import { getBacklog } from '../types/wails'
import { currentProject } from '../composables/useProjectState'
import { resolveServerUrl } from '../composables/useServerUrl'
import BacklogList from '../components/BacklogList.vue'

const backlogTasks = ref<Task[]>([])

let interval: ReturnType<typeof setInterval> | null = null
let eventSource: EventSource | null = null

async function refresh() {
  backlogTasks.value = (await getBacklog(currentProject.value ?? '')) ?? []
}

watch(currentProject, () => {
  void refresh()
})

onMounted(async () => {
  await refresh()

  if (typeof EventSource !== 'undefined') {
    try {
      const baseUrl = await resolveServerUrl()
      eventSource = new EventSource(`${baseUrl}/api/events`)
      eventSource.onmessage = (event) => {
        const data = JSON.parse(event.data)
        if (data.type !== 'connected') {
          void refresh()
        }
      }
      eventSource.onerror = () => {
        eventSource?.close()
        eventSource = null
        if (!interval) {
          interval = setInterval(() => {
            void refresh()
          }, 2000)
        }
      }
    } catch {
      interval = setInterval(() => {
        void refresh()
      }, 2000)
    }
  } else {
    interval = setInterval(() => {
      void refresh()
    }, 2000)
  }
})

onUnmounted(() => {
  if (interval) clearInterval(interval)
  if (eventSource) eventSource.close()
})
</script>
