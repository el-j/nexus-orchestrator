<!--
  AutoModeCard: starts, monitors, and stops Project Auto-Mode (Brain sync -> Plan -> Execute ->
  Verify/Repair -> Learn) for the currently selected sidebar project.
-->
<template>
  <section
    class="rounded-xl border border-violet-500/20 bg-violet-500/[0.04] p-4 space-y-3"
    data-testid="automode-card"
  >
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <div class="flex items-center gap-2">
        <span
          class="w-2 h-2 rounded-full shrink-0"
          :class="activeRun ? 'bg-emerald-400 animate-pulse' : 'bg-violet-400'"
        ></span>
        <h2 class="text-xs font-semibold text-white uppercase tracking-wider">Project Auto-Mode</h2>
        <span
          v-if="currentProject"
          class="text-[10px] px-1.5 py-0.5 rounded bg-violet-500/15 text-violet-300 font-mono"
        >
          {{ baseName(currentProject) }}
        </span>
      </div>
      <span
        v-if="latestRun"
        data-testid="automode-phase"
        class="text-[10px] px-2 py-0.5 rounded-full font-medium uppercase tracking-wide"
        :class="phaseBadgeClass(latestRun.phase)"
      >
        {{ latestRun.phase }}
      </span>
    </div>

    <p v-if="!currentProject" class="text-xs text-slate-500" data-testid="automode-no-project">
      Select a project in the sidebar to run Auto-Mode (Brain sync → Plan → Execute → Verify →
      Learn).
    </p>

    <template v-else>
      <div v-if="!activeRun" class="flex items-center gap-2 flex-wrap">
        <input
          v-model="goalInput"
          data-testid="automode-goal-input"
          type="text"
          placeholder="Describe a goal (or leave blank to execute backlog)…"
          class="flex-1 min-w-[200px] px-3 py-1.5 text-xs text-white bg-black/30 border border-white/10 rounded-lg focus:outline-none focus:border-violet-500/50 placeholder:text-slate-600"
          @keydown.enter="handleStart"
        />
        <button
          data-testid="automode-start-btn"
          :disabled="busy"
          class="text-xs font-medium px-3 py-1.5 rounded-lg bg-violet-600 hover:bg-violet-500 text-white disabled:opacity-50 transition-colors"
          @click="handleStart"
        >
          {{ busy ? 'Starting…' : 'Start Auto-Mode' }}
        </button>
      </div>

      <div v-else class="flex items-center justify-between gap-3 text-xs">
        <div class="min-w-0 space-y-0.5">
          <p class="text-slate-200 truncate">
            {{ activeRun.goal || 'Executing project backlog' }}
          </p>
          <p class="text-[11px] text-slate-400">
            Tasks: {{ activeRun.completedTasks }}/{{ activeRun.taskIds?.length ?? 0 }} done
            <template v-if="activeRun.failedTasks"> · {{ activeRun.failedTasks }} failed</template>
            <template v-if="activeRun.repairAttempts">
              · repair #{{ activeRun.repairAttempts }}</template
            >
          </p>
        </div>
        <button
          data-testid="automode-stop-btn"
          :disabled="busy"
          class="text-xs font-medium px-3 py-1.5 rounded-lg border border-red-500/30 text-red-300 hover:bg-red-500/10 disabled:opacity-50 transition-colors shrink-0"
          @click="handleStop"
        >
          Stop
        </button>
      </div>

      <p v-if="error" class="text-xs text-red-400" data-testid="automode-error">{{ error }}</p>
      <p
        v-else-if="latestRun && !activeRun && (latestRun.summary || latestRun.error)"
        class="text-[11px]"
        :class="latestRun.phase === 'failed' ? 'text-red-400' : 'text-slate-400'"
        data-testid="automode-summary"
      >
        {{ latestRun.error || latestRun.summary }}
      </p>
    </template>
  </section>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue';
import type { AutoModeRun, AutoPhase } from '../types/domain';
import { startAutoMode, listAutoModeRuns, stopAutoMode } from '../types/wails';
import { currentProject } from '../composables/useProjectState';
import { baseName } from '../utils/path';

const runs = ref<AutoModeRun[]>([]);
const goalInput = ref('');
const busy = ref(false);
const error = ref<string | null>(null);
let pollTimer: ReturnType<typeof setInterval> | undefined;

const TERMINAL = new Set<AutoPhase>(['completed', 'failed', 'stopped']);
const latestRun = computed(() => runs.value[0] ?? null);
const activeRun = computed(() => runs.value.find((r) => !TERMINAL.has(r.phase)) ?? null);

function phaseBadgeClass(phase: AutoPhase): string {
  if (phase === 'completed') return 'bg-emerald-500/20 text-emerald-300';
  if (phase === 'failed') return 'bg-red-500/20 text-red-300';
  if (phase === 'stopped') return 'bg-slate-500/20 text-slate-400';
  return 'bg-violet-500/20 text-violet-300';
}

async function refresh() {
  if (!currentProject.value) {
    runs.value = [];
    return;
  }
  try {
    runs.value = await listAutoModeRuns(currentProject.value);
    error.value = null;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load Auto-Mode status';
  }
}

async function handleStart() {
  if (busy.value) return;
  busy.value = true;
  error.value = null;
  try {
    const started = await startAutoMode({
      projectPath: currentProject.value as string,
      goal: goalInput.value.trim() || undefined,
    });
    runs.value = [started, ...runs.value.filter((r) => r.id !== started.id)];
    goalInput.value = '';
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to start Auto-Mode';
  } finally {
    busy.value = false;
  }
}

async function handleStop() {
  busy.value = true;
  error.value = null;
  try {
    await stopAutoMode((activeRun.value as AutoModeRun).id);
    await refresh();
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to stop Auto-Mode';
  } finally {
    busy.value = false;
  }
}

watch(currentProject, () => void refresh());
onMounted(() => {
  void refresh();
  pollTimer = setInterval(() => {
    if (activeRun.value) void refresh();
  }, 3000);
});
onUnmounted(() => clearInterval(pollTimer));
</script>
