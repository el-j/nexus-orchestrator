<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <header
      class="flex items-center justify-between px-5 py-3 border-b border-white/5 bg-[#0a0a10] shrink-0"
    >
      <div>
        <h1 class="text-sm font-bold text-white">Backlog</h1>
        <p class="text-xs text-slate-500">
          <span class="text-white font-semibold">{{ backlogTasks.length }}</span>
          {{ backlogTasks.length === 1 ? 'item' : 'items' }}
        </p>
      </div>
      <RefreshIndicator :last-refreshed="lastRefreshed" />
    </header>

    <!-- Error state -->
    <div
      v-if="error"
      class="mx-4 mt-3 text-sm text-red-400 bg-red-500/10 border border-red-500/20 rounded-xl p-4"
    >
      <div class="flex items-center justify-between gap-3">
        <span>{{ error }}</span>
        <button
          class="text-[11px] px-2.5 py-1 rounded border border-red-400/30 text-red-200 hover:bg-red-500/10 disabled:opacity-60 disabled:cursor-not-allowed"
          :disabled="loading"
          @click="refresh"
        >
          {{ loading ? 'Retrying…' : 'Retry' }}
        </button>
      </div>
    </div>

    <!-- List (scrollable) -->
    <div class="flex-1 overflow-auto p-4">
      <BacklogList :items="backlogTasks" @promoted="refresh" @dismissed="refresh" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue';
import type { Task } from '../types/domain';
import { getBacklog } from '../types/wails';
import { currentProject } from '../composables/useProjectState';
import { useGlobalSSE } from '../composables/useGlobalSSE';
import BacklogList from '../components/BacklogList.vue';
import RefreshIndicator from '../components/RefreshIndicator.vue';

const backlogTasks = ref<Task[]>([]);
const loading = ref(false);
const lastRefreshed = ref<Date | null>(null);
const error = ref<string | null>(null);

let interval: ReturnType<typeof setInterval> | null = null;
const { on, off, connected } = useGlobalSSE();

function sseHandler(data: { type: string; [key: string]: unknown }) {
  if (data.type === 'connected' || data.type === 'log') return;
  void refresh();
}

function startPolling() {
  if (interval) return;
  interval = setInterval(() => {
    void refresh();
  }, 2000);
}

function stopPolling() {
  if (!interval) return;
  clearInterval(interval);
  interval = null;
}

function syncPolling(isConnected: boolean) {
  if (isConnected) {
    stopPolling();
    return;
  }
  startPolling();
}

async function refresh() {
  if (loading.value) return;
  loading.value = true;
  try {
    backlogTasks.value = (await getBacklog(currentProject.value ?? '')) ?? [];
    lastRefreshed.value = new Date();
    error.value = null;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load backlog';
  } finally {
    loading.value = false;
  }
}

watch(currentProject, () => {
  void refresh();
});

watch(connected, (isConnected) => {
  syncPolling(isConnected);
  if (!isConnected) {
    void refresh();
  }
});

onMounted(async () => {
  await refresh();
  on('*', sseHandler);
  syncPolling(connected.value);
});

onUnmounted(() => {
  off('*', sseHandler);
  stopPolling();
});
</script>
