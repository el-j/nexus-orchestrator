<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <header class="flex items-center justify-between px-5 py-3 border-b border-white/5 bg-[#0a0a10] flex-shrink-0">
      <div>
        <h1 class="text-sm font-bold text-white">Task History</h1>
        <p class="text-xs text-slate-500">
          <span class="text-white font-semibold">{{ filteredTasks.length }}</span>
          {{ filteredTasks.length === 1 ? 'task' : 'tasks' }}
        </p>
      </div>

      <!-- Status filter -->
      <div class="flex items-center gap-1" role="group" aria-label="Filter tasks by status">
        <button
          v-for="f in filters"
          :key="f.value"
          @click="selectedFilter = f.value"
          :aria-pressed="selectedFilter === f.value"
          :class="[
            'px-3 py-1 rounded-lg text-xs font-medium transition-all',
            selectedFilter === f.value
              ? 'bg-violet-600/20 text-violet-300 border border-violet-500/30'
              : 'text-slate-500 hover:text-slate-300 hover:bg-white/5 border border-transparent',
          ]"
        >
          {{ f.label }}
        </button>
      </div>
    </header>

    <!-- Loading skeleton -->
    <div v-if="loading" class="flex-1 overflow-auto p-4 space-y-2">
      <div v-for="i in 5" :key="i" class="h-14 rounded-xl bg-white/[0.03] animate-pulse" />
    </div>

    <!-- Empty state -->
    <div
      v-else-if="filteredTasks.length === 0"
      class="flex flex-col items-center justify-center flex-1 py-20 text-center px-6"
    >
      <div
        class="w-16 h-16 rounded-2xl bg-violet-600/10 border border-violet-500/20
               flex items-center justify-center mb-4"
      >
        <i class="pi pi-clock text-2xl text-violet-400"></i>
      </div>
      <h3 class="font-semibold text-white mb-2">No task history yet.</h3>
      <p class="text-sm text-slate-500 max-w-xs">
        Completed, failed, and cancelled tasks will appear here.
      </p>
    </div>

    <!-- Task table -->
    <div v-else class="flex-1 overflow-auto p-4">
      <!-- Table header -->
      <div
        class="grid grid-cols-[6rem_1fr_1fr_7rem_8rem] gap-3 px-4 py-2 mb-1
               text-xs font-semibold text-slate-600 uppercase tracking-wider"
      >
        <span>ID</span>
        <span>Project</span>
        <span>Target file</span>
        <span>Status</span>
        <span>Finished</span>
      </div>

      <!-- Rows -->
      <div
        v-for="task in filteredTasks"
        :key="task.ID"
        @click="openDetail(task)"
        class="grid grid-cols-[6rem_1fr_1fr_7rem_8rem] gap-3 items-center px-4 py-3 mb-1
               rounded-xl border border-white/5 bg-[#0d0d14]
               hover:border-violet-500/20 hover:bg-[#14141f] cursor-pointer transition-all"
      >
        <!-- Short ID -->
        <span class="font-mono text-xs text-slate-400 truncate" :title="task.ID">
          {{ shortId(task.ID) }}
        </span>

        <!-- Project (last path segment) -->
        <span class="text-xs text-slate-300 truncate font-mono" :title="task.ProjectPath">
          {{ projectName(task.ProjectPath) }}
        </span>

        <!-- Target file (filename only) -->
        <span class="text-xs text-slate-400 truncate font-mono" :title="task.TargetFile">
          {{ fileName(task.TargetFile) || '—' }}
        </span>

        <!-- Status badge -->
        <TaskStatusBadge :status="task.Status" />

        <!-- Finished at (UpdatedAt as proxy) -->
        <span class="text-xs text-slate-500">
          {{ task.UpdatedAt ? formatDate(task.UpdatedAt) : '—' }}
        </span>
      </div>
    </div>

    <!-- Detail drawer -->
    <TaskDetailDrawer
      v-model="detailOpen"
      :task="selectedTask"
      @cancelled="refresh"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useTasks } from '../composables/useTasks'
import type { Task, TaskStatus } from '../types/domain'
import TaskStatusBadge from '../components/TaskStatusBadge.vue'
import TaskDetailDrawer from '../components/TaskDetailDrawer.vue'

const { tasks, loading, refresh } = useTasks()

type FilterValue = 'ALL' | 'COMPLETED' | 'FAILED' | 'CANCELLED'

const filters: { label: string; value: FilterValue }[] = [
  { label: 'All', value: 'ALL' },
  { label: 'Completed', value: 'COMPLETED' },
  { label: 'Failed', value: 'FAILED' },
  { label: 'Cancelled', value: 'CANCELLED' },
]

const selectedFilter = ref<FilterValue>('ALL')

const historyStatuses = new Set<TaskStatus>(['COMPLETED', 'FAILED', 'CANCELLED'])

const historyTasks = computed(() =>
  tasks.value.filter((t) => historyStatuses.has(t.Status)),
)

const filteredTasks = computed(() => {
  if (selectedFilter.value === 'ALL') return historyTasks.value
  return historyTasks.value.filter((t) => t.Status === selectedFilter.value)
})

const detailOpen = ref(false)
const selectedTask = ref<Task | null>(null)

function openDetail(task: Task) {
  selectedTask.value = task
  detailOpen.value = true
}

function shortId(id: string): string {
  // e.g. "TASK-189" → show as-is; UUID → first 8 chars
  if (id.length <= 12) return id
  return id.slice(0, 8)
}

function projectName(path: string): string {
  if (!path) return '—'
  const parts = path.replace(/\\/g, '/').split('/')
  return parts[parts.length - 1] || path
}

function fileName(filePath: string): string {
  if (!filePath) return ''
  const parts = filePath.replace(/\\/g, '/').split('/')
  return parts[parts.length - 1] || filePath
}

function formatDate(iso: string): string {
  try {
    const d = new Date(iso)
    const diff = Date.now() - d.getTime()
    if (diff < 60_000) return 'just now'
    if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`
    if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`
    return d.toLocaleDateString()
  } catch {
    return iso ?? '—'
  }
}
</script>
