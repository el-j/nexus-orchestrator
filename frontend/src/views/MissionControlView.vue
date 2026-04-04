<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- STATUS BAR -->
    <header class="flex items-center gap-4 px-5 py-3 border-b border-white/5 bg-[#0a0a10] shrink-0">
      <h1 class="text-sm font-bold text-white mr-2">Mission Control</h1>
      <div class="flex items-center gap-4 text-xs text-slate-400">
        <span>
          <span class="inline-block w-2 h-2 rounded-full bg-emerald-400 mr-1"></span>
          <span class="text-white font-semibold">{{ activeAgentCount }}</span> Agents
        </span>
        <span class="text-slate-700">·</span>
        <span>
          <span class="inline-block w-2 h-2 rounded-full bg-violet-400 mr-1"></span>
          <span class="text-white font-semibold">{{ queuedCount }}</span> Queued
        </span>
        <span>
          <span class="inline-block w-2 h-2 rounded-full bg-blue-400 mr-1 animate-pulse"></span>
          <span class="text-white font-semibold">{{ processingCount }}</span> Processing
        </span>
        <span class="text-slate-700">·</span>
        <span>
          <span class="inline-block w-2 h-2 rounded-full bg-slate-400 mr-1"></span>
          <span class="text-white font-semibold">{{ activeProviderCount }}</span> Providers
        </span>
      </div>
    </header>

    <!-- SCROLLABLE CONTENT -->
    <div class="flex-1 min-h-0 overflow-y-auto p-4 space-y-4">
      <!-- ACTIVE WORK -->
      <section>
        <div class="flex items-center justify-between mb-3">
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider">Active Work</h2>
          <div class="flex items-center gap-1">
            <button
              v-for="f in workFilters"
              :key="f.value"
              @click="workFilter = f.value"
              :class="[
                'px-2.5 py-1 rounded-lg text-xs font-medium transition-all',
                workFilter === f.value
                  ? 'bg-violet-600/20 text-violet-300 border border-violet-500/30'
                  : 'text-slate-500 hover:text-slate-300 hover:bg-white/5 border border-transparent',
              ]"
            >
              {{ f.label }}
            </button>
          </div>
        </div>
        <div class="rounded-xl border border-white/5 bg-white/2 overflow-hidden">
          <div v-if="tasksLoading" class="p-4 space-y-2">
            <Skeleton v-for="i in 3" :key="i" height="2rem" class="mb-2" />
          </div>
          <div
            v-else-if="filteredWorkTasks.length === 0"
            class="p-6 text-center text-sm text-slate-500"
          >
            No {{ workFilter === 'drafts' ? 'draft/backlog' : 'active' }} tasks
          </div>
          <table v-else class="w-full text-sm">
            <thead>
              <tr class="text-xs text-slate-500 uppercase tracking-wider border-b border-white/5">
                <th class="text-left px-4 py-2 font-semibold">Status</th>
                <th class="text-left px-4 py-2 font-semibold">ID</th>
                <th class="text-left px-4 py-2 font-semibold">Instruction</th>
                <th class="text-left px-4 py-2 font-semibold">Provider</th>
                <th class="text-left px-4 py-2 font-semibold">Duration</th>
                <th class="text-left px-4 py-2 font-semibold"></th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="task in filteredWorkTasks"
                :key="task.id"
                class="border-b border-white/5 hover:bg-white/2 cursor-pointer transition-colors"
                @click="openDetail(task)"
              >
                <td class="px-4 py-2.5">
                  <TaskStatusBadge :status="task.status" />
                </td>
                <td class="px-4 py-2.5 font-mono text-xs text-slate-400">{{ task.id }}</td>
                <td class="px-4 py-2.5 text-white max-w-75 truncate">
                  {{ task.instruction }}
                </td>
                <td class="px-4 py-2.5 text-slate-400 text-xs">
                  {{ task.providerName || '—' }}
                </td>
                <td class="px-4 py-2.5 text-slate-400 text-xs font-mono">
                  {{ formatDuration(task.durationMs) || '—' }}
                </td>
                <td class="px-4 py-2.5" @click.stop>
                  <button
                    v-if="task.status === 'DRAFT' || task.status === 'BACKLOG'"
                    class="text-xs text-violet-400 hover:text-violet-300 px-2 py-1 rounded border border-violet-500/20 hover:border-violet-500/40 transition-all"
                    :disabled="promoting === task.id"
                    @click="handlePromote(task.id)"
                  >
                    {{ promoting === task.id ? '…' : 'Promote ↑' }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <!-- HISTORY -->
      <section>
        <div class="flex items-center justify-between mb-3">
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider">
            Recent Completions
          </h2>
          <button
            @click="showAllHistory = !showAllHistory"
            class="text-xs text-slate-400 hover:text-white px-2.5 py-1 rounded-lg border border-white/10 hover:border-violet-500/40 transition-all"
          >
            {{ showAllHistory ? 'Show Recent' : 'Show All History' }}
          </button>
        </div>

        <!-- History filter tabs (only when expanded) -->
        <div v-if="showAllHistory" class="flex items-center gap-1 mb-3">
          <button
            v-for="f in historyFilters"
            :key="f.value"
            @click="historyFilter = f.value"
            :class="[
              'px-2.5 py-1 rounded-lg text-xs font-medium transition-all',
              historyFilter === f.value
                ? 'bg-violet-600/20 text-violet-300 border border-violet-500/30'
                : 'text-slate-500 hover:text-slate-300 hover:bg-white/5 border border-transparent',
            ]"
          >
            {{ f.label }}
          </button>
        </div>

        <div class="rounded-xl border border-white/5 bg-white/2 overflow-hidden">
          <div v-if="displayedHistory.length === 0" class="p-4 text-center text-sm text-slate-500">
            No completed tasks yet
          </div>
          <div v-else class="divide-y divide-white/5">
            <div
              v-for="task in displayedHistory"
              :key="task.id"
              class="flex items-center gap-3 px-4 py-2.5 hover:bg-white/2 cursor-pointer transition-colors"
              @click="openDetail(task)"
            >
              <TaskStatusBadge :status="task.status" />
              <span class="font-mono text-xs text-slate-400 shrink-0">{{ task.id }}</span>
              <span class="text-xs text-white truncate flex-1">{{ task.instruction }}</span>
              <span class="text-xs text-slate-500 shrink-0">{{
                formatRelativeTime(task.updatedAt)
              }}</span>
            </div>
          </div>
        </div>
      </section>

      <!-- AGENTS + PROVIDERS GRID -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <!-- AGENTS PANEL -->
        <section>
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Agents</h2>
          <div class="space-y-2">
            <div v-if="sessionsLoading" class="space-y-2">
              <Skeleton v-for="i in 2" :key="i" height="3rem" />
            </div>
            <div
              v-else-if="sessions.length === 0"
              class="rounded-xl border border-white/5 bg-white/2 p-4 text-sm text-slate-500 text-center"
            >
              No agents connected
            </div>
            <div
              v-for="session in sessions"
              :key="session.id"
              class="rounded-xl border border-white/5 bg-white/2 p-4 flex items-center gap-3"
            >
              <span
                :class="[
                  'w-2 h-2 rounded-full shrink-0',
                  session.status === 'active'
                    ? 'bg-emerald-400'
                    : session.status === 'idle'
                      ? 'bg-yellow-500'
                      : 'bg-slate-500',
                ]"
              ></span>
              <div class="flex-1 min-w-0">
                <p class="text-sm text-white font-medium truncate">{{ session.agentName }}</p>
                <p class="text-xs text-slate-500 truncate">
                  {{ session.projectPath ? basename(session.projectPath) : '—' }}
                  <span v-if="session.modelId" class="text-slate-600 ml-1">
                    · {{ session.modelId }}
                  </span>
                </p>
              </div>
              <span
                :class="[
                  'text-xs px-2 py-0.5 rounded-full',
                  session.status === 'active'
                    ? 'bg-emerald-500/20 text-emerald-300'
                    : session.status === 'idle'
                      ? 'bg-yellow-500/20 text-yellow-300'
                      : 'bg-slate-500/20 text-slate-400',
                ]"
              >
                {{ formatSessionStatus(session) }}
              </span>
            </div>
          </div>
        </section>

        <!-- PROVIDERS PANEL -->
        <section>
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">
            Providers
          </h2>
          <div class="space-y-2">
            <div v-if="providersLoading" class="space-y-2">
              <Skeleton v-for="i in 2" :key="i" height="3rem" />
            </div>
            <div
              v-else-if="providers.length === 0"
              class="rounded-xl border border-white/5 bg-white/2 p-4 text-sm text-slate-500 text-center"
            >
              No providers available
            </div>
            <div
              v-for="provider in providers"
              :key="provider.name"
              class="rounded-xl border border-white/5 bg-white/2 p-4 flex items-center gap-3"
            >
              <span
                :class="[
                  'w-2 h-2 rounded-full shrink-0',
                  provider.active ? 'bg-emerald-400' : 'bg-red-500',
                ]"
              ></span>
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <p class="text-sm text-white font-medium truncate">{{ provider.name }}</p>
                  <span
                    v-if="provider.consecutiveFails && provider.consecutiveFails > 0"
                    class="text-[10px] px-1.5 py-0.5 rounded-full bg-red-500/20 text-red-400 font-semibold"
                  >
                    {{ provider.consecutiveFails }} fail{{
                      provider.consecutiveFails > 1 ? 's' : ''
                    }}
                  </span>
                </div>
                <p class="text-xs text-slate-500 truncate">
                  {{ provider.activeModel || 'no model' }}
                  <span v-if="provider.contextLimit" class="text-slate-600 ml-1">
                    · {{ formatContextLimit(provider.contextLimit) }}
                  </span>
                  <span v-if="provider.baseURL" class="text-slate-600 ml-1">
                    · {{ provider.baseURL }}
                  </span>
                </p>
              </div>
              <span v-if="provider.lastChecked" class="text-xs text-slate-500">
                {{ formatRelativeTime(provider.lastChecked) }}
              </span>
            </div>
          </div>
        </section>
      </div>
    </div>

    <!-- SUBMIT FORM (pinned bottom) -->
    <TaskSubmitForm class="shrink-0" @submitted="handleSubmitted" />

    <!-- DETAIL DRAWER -->
    <TaskDetailDrawer v-model="detailOpen" :task="selectedTask" @cancelled="onCancelled" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import Skeleton from 'primevue/skeleton';
import { useToast } from 'primevue/usetoast';
import { useTasks } from '../composables/useTasks';
import { useProviders } from '../composables/useProviders';
import { useAISessions } from '../composables/useAISessions';
import { getAllTasks } from '../types/wails';
import type { Task, AISession } from '../types/domain';
import TaskStatusBadge from '../components/TaskStatusBadge.vue';
import TaskDetailDrawer from '../components/TaskDetailDrawer.vue';
import TaskSubmitForm from '../components/TaskSubmitForm.vue';

// --- Composables ---
const toast = useToast();
const { tasks, loading: tasksLoading, refresh: refreshTasks, promoteTask } = useTasks();
const { providers, loading: providersLoading } = useProviders();
const { sessions, loading: sessionsLoading } = useAISessions();

// --- Work filter (Active Work section) ---
type WorkFilterValue = 'all' | 'active' | 'drafts';
const workFilters: { label: string; value: WorkFilterValue }[] = [
  { label: 'All', value: 'all' },
  { label: 'Active', value: 'active' },
  { label: 'Drafts/Backlog', value: 'drafts' },
];
const workFilter = ref<WorkFilterValue>('all');
const promoting = ref<string | null>(null);

async function handlePromote(id: string) {
  if (promoting.value) return;
  promoting.value = id;
  try {
    await promoteTask(id);
    await refreshHistory();
  } catch (e) {
    toast.add({ severity: 'error', summary: 'Promote Failed', detail: String(e), life: 5000 });
  } finally {
    promoting.value = null;
  }
}

// --- Recent history ---
const recentHistory = ref<Task[]>([]);
const allHistory = ref<Task[]>([]);
const showAllHistory = ref(false);

type HistoryFilterValue = 'ALL' | 'COMPLETED' | 'FAILED' | 'CANCELLED';
const historyFilters: { label: string; value: HistoryFilterValue }[] = [
  { label: 'All', value: 'ALL' },
  { label: 'Completed', value: 'COMPLETED' },
  { label: 'Failed', value: 'FAILED' },
  { label: 'Cancelled', value: 'CANCELLED' },
];
const historyFilter = ref<HistoryFilterValue>('ALL');

let historyInterval: ReturnType<typeof setInterval> | null = null;
const historyStatuses = new Set(['COMPLETED', 'FAILED', 'CANCELLED']);

async function refreshHistory() {
  try {
    const all = (await getAllTasks()) ?? [];
    allHistory.value = all.filter((t) => historyStatuses.has(t.status));
    const oneHourAgo = Date.now() - 3_600_000;
    recentHistory.value = all.filter(
      (t) =>
        (t.status === 'COMPLETED' || t.status === 'FAILED') &&
        new Date(t.updatedAt).getTime() > oneHourAgo,
    );
  } catch {
    // silently ignore — queue tasks are the primary source
  }
}

onMounted(() => {
  refreshHistory();
  historyInterval = setInterval(refreshHistory, 30_000);
});
onUnmounted(() => {
  if (historyInterval) clearInterval(historyInterval);
});

// --- Status bar counts ---
const activeAgentCount = computed(
  () => sessions.value.filter((s) => s.status === 'active' || s.status === 'idle').length,
);
const queuedCount = computed(() => tasks.value.filter((t) => t.status === 'QUEUED').length);
const processingCount = computed(() => tasks.value.filter((t) => t.status === 'PROCESSING').length);
const activeProviderCount = computed(() => providers.value.filter((p) => p.active).length);

// --- Sorted task list (queue + recent history) ---
const statusPriority: Record<string, number> = {
  PROCESSING: 0,
  QUEUED: 1,
  DRAFT: 2,
  BACKLOG: 3,
  FAILED: 4,
  COMPLETED: 5,
  CANCELLED: 6,
  TOO_LARGE: 7,
  NO_PROVIDER: 8,
};

const sortedTasks = computed(() => {
  const ids = new Set<string>();
  const merged: Task[] = [];
  // Queue tasks first
  for (const t of tasks.value) {
    ids.add(t.id);
    merged.push(t);
  }
  // Add recent history not already in queue
  for (const t of recentHistory.value) {
    if (!ids.has(t.id)) {
      merged.push(t);
    }
  }
  return merged.sort((a, b) => (statusPriority[a.status] ?? 99) - (statusPriority[b.status] ?? 99));
});

// --- Filtered work tasks ---
const filteredWorkTasks = computed(() => {
  if (workFilter.value === 'active') {
    return sortedTasks.value.filter((t) => t.status === 'PROCESSING' || t.status === 'QUEUED');
  }
  if (workFilter.value === 'drafts') {
    return sortedTasks.value.filter((t) => t.status === 'DRAFT' || t.status === 'BACKLOG');
  }
  return sortedTasks.value;
});

// --- History display ---
const displayedHistory = computed(() => {
  if (!showAllHistory.value) {
    return recentHistory.value.slice(0, 10);
  }
  if (historyFilter.value === 'ALL') return allHistory.value;
  return allHistory.value.filter((t) => t.status === historyFilter.value);
});

// --- Detail drawer ---
const detailOpen = ref(false);
const selectedTask = ref<Task | null>(null);

function openDetail(task: Task) {
  selectedTask.value = task;
  detailOpen.value = true;
}

function onCancelled() {
  refreshTasks();
  refreshHistory();
}

function handleSubmitted() {
  refreshTasks();
}

// --- Helpers ---
function formatRelativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  if (diff < 60_000) return 'just now';
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  return new Date(iso).toLocaleDateString();
}

function formatDuration(ms?: number): string {
  if (!ms) return '';
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

function formatContextLimit(limit?: number): string {
  if (!limit) return '';
  return `${Math.round(limit / 1024)}k ctx`;
}

function formatSessionStatus(session: AISession): string {
  if (session.status === 'active') return 'active';
  if (session.status === 'disconnected') return 'disconnected';
  // idle — show how long
  const diff = Date.now() - new Date(session.lastActivity).getTime();
  if (diff < 60_000) return 'active';
  return `idle ${Math.floor(diff / 60_000)}m`;
}

function basename(path: string): string {
  return path.split('/').filter(Boolean).pop() || path;
}
</script>
