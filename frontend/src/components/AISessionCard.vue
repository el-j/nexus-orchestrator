<template>
  <div
    class="rounded-xl border bg-white/[0.03] p-4 transition-all border-l-4"
    :style="{ borderLeftColor: cardBorderColour }"
  >
    <!-- Header: status dot + agent name + status badge -->
    <div class="flex items-center gap-2 mb-2">
      <span
        class="w-2 h-2 rounded-full flex-shrink-0"
        :style="{ backgroundColor: cardBorderColour }"
      ></span>
      <span class="font-semibold text-sm text-white truncate flex-1">{{ session.agentName }}</span>
      <span
        class="text-[10px] px-1.5 py-0.5 rounded font-medium flex-shrink-0"
        :class="{
          'bg-emerald-500/20 text-emerald-300': session.status === 'active',
          'bg-yellow-500/20 text-yellow-300': session.status === 'idle',
          'bg-slate-700/50 text-slate-400': session.status === 'disconnected',
        }"
        >{{ session.status }}</span
      >
    </div>

    <!-- Source badge + Delegated chip + Model badge -->
    <div class="flex items-center gap-2 mb-2 flex-wrap">
      <span class="text-[11px] px-2 py-0.5 rounded bg-white/5 text-slate-400">
        <template v-if="session.source === 'mcp'">🤖 MCP</template>
        <template v-else-if="session.source === 'vscode'">🔵 VS Code</template>
        <template v-else>🌐 HTTP</template>
      </span>
      <span
        v-if="session.delegatedToNexus"
        class="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300 font-medium"
        >✓ Delegated</span
      >
      <span
        v-if="session.modelId"
        class="bg-slate-100 text-slate-700 rounded px-1.5 text-xs font-mono"
        >{{ shortModelName(session.modelId) }}</span
      >
    </div>

    <!-- Project path: last 2 segments -->
    <div
      v-if="session.projectPath"
      class="text-[11px] text-slate-500 font-mono mb-2 truncate"
      :title="session.projectPath"
    >
      {{ shortPath }}
    </div>

    <!-- Last activity -->
    <div class="text-[11px] text-slate-600 mb-2">
      {{ relativeTime(session.lastActivity) }}
    </div>

    <!-- Capability chips -->
    <div v-if="session.agentCapabilities?.length" class="flex flex-wrap gap-1 mb-3">
      <span
        v-for="cap in session.agentCapabilities"
        :key="cap"
        class="text-[10px] px-1.5 py-0.5 rounded bg-violet-500/10 text-violet-400"
        >{{ cap }}</span
      >
    </div>

    <!-- Delegate button -->
    <div class="mb-3">
      <button
        v-if="!session.delegatedToNexus"
        class="text-xs text-violet-400 hover:text-violet-300 px-3 py-1 rounded-lg border border-violet-500/30 hover:border-violet-500/60 transition-all"
        @click="$emit('delegate', session)"
      >
        Delegate →
      </button>
    </div>

    <!-- Collapsible task timeline (TASK-287) -->
    <details @toggle="handleToggle">
      <summary
        class="text-[11px] text-slate-500 cursor-pointer hover:text-slate-300 select-none list-none outline-none"
      >
        {{ open ? 'Hide tasks ▲' : 'Show tasks ▼' }}
      </summary>
      <div v-if="open" class="mt-2">
        <div v-if="loadingTasks" class="text-[11px] text-slate-500 py-2">Loading…</div>
        <div v-else-if="sessionTasks.length === 0" class="text-[11px] text-slate-600 py-2">
          No tasks yet
        </div>
        <div v-else class="space-y-0.5">
          <div
            v-for="task in sessionTasks.slice(0, 50)"
            :key="task.id"
            class="flex items-start gap-2 py-1 border-b border-white/5 last:border-0"
          >
            <span
              :class="`text-[9px] px-1.5 py-0.5 rounded font-medium flex-shrink-0 ${statusChipClass(task.status)}`"
            >
              {{ task.status }}
            </span>
            <span class="text-[10px] text-slate-400 font-mono truncate flex-1 min-w-0">{{
              task.targetFile
            }}</span>
            <span class="text-[10px] text-slate-500 truncate max-w-[120px] flex-shrink-0">
              {{ (task.instruction ?? '').slice(0, 80) }}…
            </span>
            <span class="text-[10px] text-slate-600 flex-shrink-0 whitespace-nowrap">{{
              relativeTime(task.updatedAt)
            }}</span>
          </div>
        </div>
      </div>
    </details>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue';
import type { AISession, Task, TaskStatus } from '../types/domain';
import { resolveServerUrl } from '../composables/useServerUrl';
import { relativeTime } from '../utils/time';

const props = defineProps<{ session: AISession }>();
defineEmits<{ (e: 'delegate', session: AISession): void }>();

const open = ref(false);
const loadingTasks = ref(false);
const sessionTasks = ref<Task[]>([]);
let refreshTimer: ReturnType<typeof setInterval> | null = null;
let disposed = false;
const TIMELINE_REFRESH_MS = 5000;

const cardBorderColour = computed((): string => {
  if (props.session.status === 'active' && props.session.delegatedToNexus) return '#4ade80';
  if (props.session.status === 'active') return '#facc15';
  if (props.session.status === 'idle') return '#fb923c';
  return '#6b7280';
});

const shortPath = computed((): string => {
  if (!props.session.projectPath) return '';
  const parts = props.session.projectPath.replace(/\\/g, '/').split('/').filter(Boolean);
  return parts.slice(-2).join('/');
});

function shortModelName(id: string): string {
  return id.replace(/^claude-/, '');
}

function statusChipClass(status: TaskStatus | undefined): string {
  switch (status) {
    case 'COMPLETED':
      return 'bg-emerald-500/20 text-emerald-300';
    case 'PROCESSING':
      return 'bg-blue-500/20 text-blue-300';
    case 'QUEUED':
      return 'bg-violet-500/20 text-violet-300';
    case 'FAILED':
      return 'bg-red-500/20 text-red-300';
    case 'CANCELLED':
      return 'bg-slate-700/50 text-slate-400';
    case 'DRAFT':
    case 'BACKLOG':
      return 'bg-yellow-500/15 text-yellow-300';
    default:
      return 'bg-slate-700/50 text-slate-400';
  }
}

/** Loads the tasks claimed by THIS session (not every task of its project). */
async function loadSessionTasks(): Promise<void> {
  try {
    const base = await resolveServerUrl();
    const r = await fetch(`${base}/api/ai-sessions/${encodeURIComponent(props.session.id)}/tasks`);
    if (r.ok) sessionTasks.value = ((await r.json()) as Task[] | null) ?? [];
  } catch {
    /* non-critical: keep showing what we have */
  }
}

async function openTimeline() {
  loadingTasks.value = true;
  await loadSessionTasks();
  loadingTasks.value = false;
  // The panel may have been closed (or the card unmounted) while the request was
  // in flight; starting the poll now would leak a timer nobody clears.
  if (!open.value || disposed) return;
  // /api/ai-sessions/{id}/tasks is a plain JSON endpoint (not an event stream),
  // so keep the list fresh by re-reading it while the panel is open.
  closeTimeline();
  refreshTimer = setInterval(() => void loadSessionTasks(), TIMELINE_REFRESH_MS);
}

function closeTimeline() {
  if (refreshTimer) clearInterval(refreshTimer);
  refreshTimer = null;
}

function handleToggle(e: Event) {
  open.value = (e.target as HTMLDetailsElement).open;
  if (open.value) {
    openTimeline();
  } else {
    closeTimeline();
  }
}

onUnmounted(() => {
  disposed = true;
  closeTimeline();
});
</script>
