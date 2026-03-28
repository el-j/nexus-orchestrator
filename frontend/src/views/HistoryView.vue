<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <header class="flex flex-col border-b border-white/5 bg-[#0a0a10] shrink-0">
      <!-- Title / count row -->
      <div class="flex items-center justify-between px-5 py-3">
        <div>
          <h1 class="text-sm font-bold text-white">Task History</h1>
          <p class="text-xs text-slate-500">
            <span class="text-white font-semibold">{{ filteredTasks.length }}</span>
            {{ filteredTasks.length === 1 ? 'task' : 'tasks' }}
          </p>
        </div>
      </div>

      <!-- Status distribution summary bar -->
      <div
        v-if="summaryBadges.length > 0"
        class="flex flex-wrap gap-2 px-5 py-2 border-b border-white/5"
      >
        <button
          v-for="badge in summaryBadges"
          :key="badge.status"
          @click="toggleStatusFilter(badge.status)"
          :aria-pressed="selectedFilter === badge.status"
          :class="[
            'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium transition-all cursor-pointer border',
            badge.color,
            selectedFilter === badge.status
              ? 'border-current opacity-100'
              : 'border-transparent opacity-75 hover:opacity-100',
          ]"
        >
          {{ badge.label }}&nbsp;<span class="font-mono">{{ badge.count }}</span>
        </button>
      </div>

      <!-- Status filter tabs -->
      <div
        class="flex items-center gap-1 px-5 py-2"
        role="group"
        aria-label="Filter tasks by status"
      >
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
      <div v-for="i in 5" :key="i" class="h-14 rounded-xl bg-white/3 animate-pulse" />
    </div>

    <!-- Empty state -->
    <div
      v-else-if="filteredTasks.length === 0"
      class="flex flex-col items-center justify-center flex-1 py-20 text-center px-6"
    >
      <div
        class="w-16 h-16 rounded-2xl bg-violet-600/10 border border-violet-500/20 flex items-center justify-center mb-4"
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
        class="grid grid-cols-[6rem_1fr_1fr_7rem_8rem] gap-3 px-4 py-2 mb-1 text-xs font-semibold text-slate-600 uppercase tracking-wider"
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
        :key="task.id"
        @click="openDetail(task)"
        class="grid grid-cols-[6rem_1fr_1fr_7rem_8rem] gap-3 items-center px-4 py-3 mb-1 rounded-xl border border-white/5 bg-nexus-800 hover:border-violet-500/20 hover:bg-nexus-700 cursor-pointer transition-all"
      >
        <!-- Short ID -->
        <span class="font-mono text-xs text-slate-400 truncate" :title="task.id">
          {{ shortId(task.id) }}
        </span>

        <!-- Project (last path segment) -->
        <span class="text-xs text-slate-300 truncate font-mono" :title="task.projectPath">
          {{ projectName(task.projectPath) }}
        </span>

        <!-- Target file (filename only) -->
        <span class="text-xs text-slate-400 truncate font-mono" :title="task.targetFile">
          {{ fileName(task.targetFile) || '—' }}
        </span>

        <!-- Status badge -->
        <TaskStatusBadge :status="task.status" />

        <!-- Finished at (updatedAt as proxy) -->
        <span class="text-xs text-slate-500">
          {{ task.updatedAt ? formatDate(task.updatedAt) : '—' }}
        </span>
      </div>
    </div>

    <!-- Detail drawer -->
    <TaskDetailDrawer v-model="detailOpen" :task="selectedTask" @cancelled="refresh" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue';
import type { Task, TaskStatus } from '../types/domain';
import { currentProject } from '../composables/useProjectState';
import { resolveServerUrl } from '../composables/useServerUrl';
import TaskStatusBadge from '../components/TaskStatusBadge.vue';
import TaskDetailDrawer from '../components/TaskDetailDrawer.vue';
import { formatDate } from '../utils/time';

const tasks = ref<Task[]>([]);
const loading = ref(false);
let interval: ReturnType<typeof setInterval> | null = null;
let eventSource: EventSource | null = null;

type FilterValue = 'ALL' | 'COMPLETED' | 'FAILED' | 'CANCELLED';

const filters: { label: string; value: FilterValue }[] = [
  { label: 'All', value: 'ALL' },
  { label: 'Completed', value: 'COMPLETED' },
  { label: 'Failed', value: 'FAILED' },
  { label: 'Cancelled', value: 'CANCELLED' },
];

const selectedFilter = ref<TaskStatus | 'ALL'>('ALL');

const historyStatuses = new Set<TaskStatus>(['COMPLETED', 'FAILED', 'CANCELLED']);

const projectTasks = computed(() =>
  currentProject.value === null
    ? tasks.value
    : tasks.value.filter((t) => t.projectPath === currentProject.value),
);

const statusColorMap: Record<string, { color: string; label: string }> = {
  QUEUED: { color: 'bg-slate-500/20 text-slate-400', label: 'Queued' },
  PROCESSING: { color: 'bg-yellow-500/20 text-yellow-300', label: 'Processing' },
  COMPLETED: { color: 'bg-emerald-500/20 text-emerald-300', label: 'Completed' },
  FAILED: { color: 'bg-red-500/20 text-red-300', label: 'Failed' },
  CANCELLED: { color: 'bg-orange-500/20 text-orange-300', label: 'Cancelled' },
  BACKLOG: { color: 'bg-violet-500/20 text-violet-300', label: 'Backlog' },
  DRAFT: { color: 'bg-blue-500/20 text-blue-300', label: 'Draft' },
};

const statusSummary = computed<Partial<Record<TaskStatus, number>>>(() => {
  const counts: Partial<Record<TaskStatus, number>> = {};
  for (const task of projectTasks.value) {
    counts[task.status] = (counts[task.status] ?? 0) + 1;
  }
  return counts;
});

const summaryBadges = computed(() =>
  Object.entries(statusColorMap)
    .filter(([status]) => (statusSummary.value[status as TaskStatus] ?? 0) > 0)
    .map(([status, { color, label }]) => ({
      status: status as TaskStatus,
      label,
      color,
      count: statusSummary.value[status as TaskStatus] ?? 0,
    })),
);

const historyTasks = computed(() =>
  tasks.value.filter(
    (t) =>
      historyStatuses.has(t.status) &&
      (currentProject.value === null || t.projectPath === currentProject.value),
  ),
);

const filteredTasks = computed(() => {
  if (selectedFilter.value === 'ALL') return historyTasks.value;
  return historyTasks.value.filter((t) => t.status === selectedFilter.value);
});

const detailOpen = ref(false);
const selectedTask = ref<Task | null>(null);

async function refresh() {
  const baseUrl = await resolveServerUrl();
  const params = new URLSearchParams();
  if (currentProject.value) {
    params.set('projectPath', currentProject.value);
  }
  const query = params.toString();
  const res = await fetch(`${baseUrl}/api/tasks/all${query ? `?${query}` : ''}`);
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}`);
  }
  tasks.value = ((await res.json()) as Task[]) ?? [];
}

watch(currentProject, () => {
  void refresh();
});

onMounted(async () => {
  loading.value = true;
  await refresh();

  if (typeof EventSource !== 'undefined') {
    try {
      const baseUrl = await resolveServerUrl();
      eventSource = new EventSource(`${baseUrl}/api/events`);
      eventSource.onmessage = (event) => {
        const data = JSON.parse(event.data);
        if (data.type !== 'connected') {
          void refresh();
        }
      };
      eventSource.onerror = () => {
        eventSource?.close();
        eventSource = null;
        if (!interval) {
          interval = setInterval(() => {
            void refresh();
          }, 2000);
        }
      };
    } catch {
      interval = setInterval(() => {
        void refresh();
      }, 2000);
    }
  } else {
    interval = setInterval(() => {
      void refresh();
    }, 2000);
  }

  loading.value = false;
});

onUnmounted(() => {
  if (interval) clearInterval(interval);
  if (eventSource) eventSource.close();
});

function openDetail(task: Task) {
  selectedTask.value = task;
  detailOpen.value = true;
}

function toggleStatusFilter(status: TaskStatus) {
  selectedFilter.value = selectedFilter.value === status ? 'ALL' : status;
}

function shortId(id: string): string {
  // e.g. "TASK-189" → show as-is; UUID → first 8 chars
  if (id.length <= 12) return id;
  return id.slice(0, 8);
}

function projectName(path: string): string {
  if (!path) return '—';
  const parts = path.replace(/\\/g, '/').split('/');
  return parts[parts.length - 1] || path;
}

function fileName(filePath: string): string {
  if (!filePath) return '';
  const parts = filePath.replace(/\\/g, '/').split('/');
  return parts[parts.length - 1] || filePath;
}
</script>
