<template>
  <div class="flex flex-col flex-1 min-h-0 overflow-hidden">
    <!-- Header bar -->
    <header class="flex items-center justify-between px-5 py-3 border-b border-white/5 bg-[#0a0a10] flex-shrink-0">
      <div>
        <h1 class="text-sm font-bold text-white">Task Queue</h1>
        <p class="text-xs text-slate-500" role="status" aria-live="polite">
          <span class="text-white font-semibold">{{ activeCount }}</span> active
          <span class="mx-1 text-slate-700">·</span>
          <span class="text-white font-semibold">{{ tasks.length }}</span> total
        </p>
      </div>
      <ProviderStatus :providers="providers" :refresh="refreshProviders" />
    </header>

    <!-- Task list (scrollable) -->
    <div class="flex-1 min-h-0 flex flex-col">
      <TaskQueue
        :tasks="tasks"
        :loading="loading"
        @select="openDetail"
        @cancel="handleCancel"
      />
    </div>

    <!-- Submit form (pinned at bottom) -->
    <TaskSubmitForm class="flex-shrink-0" @submitted="handleSubmitted" />

    <!-- Detail drawer -->
    <TaskDetailDrawer
      v-model="detailOpen"
      :task="selectedTask"
      @cancelled="onCancelled"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useToast } from 'primevue/usetoast'
import { useTasks } from '../composables/useTasks'
import { useProviders } from '../composables/useProviders'
import { cancelTask } from '../types/wails'
import type { Task } from '../types/domain'
import TaskQueue from '../components/TaskQueue.vue'
import TaskSubmitForm from '../components/TaskSubmitForm.vue'
import TaskDetailDrawer from '../components/TaskDetailDrawer.vue'
import ProviderStatus from '../components/ProviderStatus.vue'

const { tasks, loading, refresh } = useTasks()
const { providers, refresh: refreshProviders } = useProviders()
const toast = useToast()

const detailOpen = ref(false)
const selectedTask = ref<Task | null>(null)

const activeCount = computed(() =>
  tasks.value.filter((t) => t.Status === 'QUEUED' || t.Status === 'PROCESSING').length,
)

function openDetail(task: Task) {
  selectedTask.value = task
  detailOpen.value = true
}

async function handleCancel(id: string) {
  try {
    await cancelTask(id)
    toast.add({ severity: 'success', summary: 'Cancelled', life: 2000 })
    await refresh()
  } catch (e) {
    toast.add({
      severity: 'error',
      summary: 'Cancel failed',
      detail: e instanceof Error ? e.message : String(e),
      life: 4000,
    })
  }
}

async function onCancelled(_id: string) {
  await refresh()
}

function handleSubmitted(_id: string) {
  refresh()
}
</script>
