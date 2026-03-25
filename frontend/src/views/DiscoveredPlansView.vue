<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <header
      class="flex items-center justify-between px-5 py-3 border-b border-white/5 bg-[#0a0a10] flex-shrink-0"
    >
      <div>
        <h1 class="text-sm font-bold text-white">Plans</h1>
        <p class="text-xs text-slate-500">
          <span class="font-semibold text-cyan-400">{{ plans.length }}</span>
          discovered across
          <span class="font-semibold text-cyan-400">{{ groupCount }}</span>
          project{{ groupCount !== 1 ? 's' : '' }}
        </p>
      </div>
      <div class="flex items-center gap-2">
        <button
          class="text-xs text-slate-400 hover:text-white px-3 py-1.5 rounded-lg border border-white/10 hover:border-violet-500/40 transition-all"
          :disabled="loading"
          @click="scan"
        >
          <i :class="['pi pi-search text-xs mr-1', loading && 'animate-spin']"></i>
          {{ loading ? 'Scanning…' : 'Scan Now' }}
        </button>
      </div>
    </header>

    <!-- Content -->
    <div class="flex-1 overflow-auto p-5">
      <!-- Loading -->
      <div v-if="loading && plans.length === 0" class="flex items-center justify-center py-16">
        <div
          class="w-6 h-6 border-2 border-violet-500 border-t-transparent rounded-full animate-spin"
        ></div>
      </div>

      <!-- Error -->
      <div
        v-else-if="error"
        class="text-sm text-red-400 bg-red-500/10 border border-red-500/20 rounded-xl p-4"
      >
        {{ error }}
      </div>

      <!-- Empty state -->
      <div
        v-else-if="plans.length === 0"
        class="flex flex-col items-center justify-center py-20 text-center"
      >
        <div class="text-4xl mb-4 opacity-40">📂</div>
        <p class="text-sm font-medium text-slate-400">No plan files discovered</p>
        <p class="text-xs text-slate-600 mt-1 max-w-sm">
          Click "Scan Now" to search your projects for plan, task, and orchestration files.
        </p>
      </div>

      <!-- Grouped by project -->
      <template v-else>
        <section v-for="(group, projPath) in grouped" :key="projPath" class="mb-8">
          <h2
            class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3 font-mono truncate"
            :title="String(projPath)"
          >
            {{ String(projPath) }}
          </h2>
          <div class="grid grid-cols-1 gap-3">
            <div
              v-for="plan in group"
              :key="plan.id"
              class="rounded-xl border border-white/10 bg-white/[0.02] p-4 hover:bg-white/[0.04] transition-all"
            >
              <div class="flex items-start gap-3">
                <!-- Active dot -->
                <span
                  class="w-2 h-2 rounded-full flex-shrink-0 mt-1.5"
                  :class="plan.isActive ? 'bg-emerald-400' : 'bg-slate-600'"
                  :title="plan.isActive ? 'Active' : 'Inactive'"
                ></span>

                <div class="flex-1 min-w-0">
                  <!-- Filename + badges -->
                  <div class="flex items-center gap-2 flex-wrap mb-1">
                    <a
                      :href="`file://${plan.path}`"
                      class="font-semibold text-sm text-white truncate hover:text-violet-300 transition-colors"
                      :title="plan.path"
                    >
                      {{ basename(plan.path) }}
                    </a>
                    <!-- Kind badge -->
                    <span
                      class="text-[10px] px-1.5 py-0.5 rounded font-semibold flex-shrink-0"
                      :class="kindBadgeClass(plan.kind)"
                    >
                      {{ plan.kind }}
                    </span>
                    <!-- Format chip -->
                    <span
                      class="text-[10px] px-1.5 py-0.5 rounded bg-white/5 text-slate-400 flex-shrink-0 font-mono"
                    >
                      {{ plan.format }}
                    </span>
                  </div>

                  <!-- Summary -->
                  <p v-if="plan.summary" class="text-[11px] text-slate-400 mb-1 line-clamp-1">
                    {{ truncate(plan.summary, 80) }}
                  </p>

                  <!-- Path + last modified -->
                  <div class="flex items-center gap-3 flex-wrap">
                    <span
                      class="text-[10px] text-slate-600 font-mono truncate max-w-xs"
                      :title="plan.path"
                    >
                      {{ plan.path }}
                    </span>
                    <span class="text-[10px] text-slate-700 flex-shrink-0">
                      {{ formatDate(plan.lastModified) }}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { useDiscoveredPlans } from '../composables/useDiscoveredPlans';
import type { PlanFileKind } from '../types/domain';

const projectPath = ref('');
const { plans, loading, error, scan } = useDiscoveredPlans(projectPath);

const grouped = computed(() => {
  const map: Record<string, typeof plans.value> = {};
  for (const plan of plans.value) {
    const key = plan.projectPath || '(unknown project)';
    if (!map[key]) map[key] = [];
    map[key].push(plan);
  }
  return map;
});

const groupCount = computed(() => Object.keys(grouped.value).length);

function basename(path: string): string {
  return path.split('/').pop() ?? path;
}

function truncate(text: string, max: number): string {
  return text.length > max ? text.slice(0, max) + '…' : text;
}

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  } catch {
    return iso;
  }
}

const kindColors: Record<PlanFileKind, string> = {
  nexus: 'bg-indigo-500/20 text-indigo-300',
  markdown: 'bg-emerald-500/20 text-emerald-300',
  cursor: 'bg-blue-500/20 text-blue-300',
  claude: 'bg-orange-500/20 text-orange-300',
  'mcp-config': 'bg-purple-500/20 text-purple-300',
  crewai: 'bg-pink-500/20 text-pink-300',
  'claude-task': 'bg-amber-500/20 text-amber-300',
};

function kindBadgeClass(kind: PlanFileKind): string {
  return kindColors[kind] ?? 'bg-white/5 text-slate-400';
}
</script>
