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
          <span class="font-semibold text-cyan-400">{{ Object.keys(projectGroups).length }}</span>
          project{{ Object.keys(projectGroups).length !== 1 ? 's' : '' }}
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
        <section v-for="(data, projPath) in projectGroups" :key="projPath" class="mb-10">
          <!-- Project path header -->
          <h2
            class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-4 font-mono truncate"
            :title="String(projPath)"
          >
            {{ String(projPath) }}
          </h2>

          <!-- Project Brain card (nexus) -->
          <div
            v-if="data.nexus"
            class="mb-5 rounded-xl border border-violet-500/30 bg-violet-500/[0.04] p-4"
          >
            <div class="flex items-start gap-3">
              <span class="text-lg mt-0.5">🧠</span>
              <div class="flex-1">
                <div class="flex items-center gap-2 mb-1">
                  <span class="font-bold text-sm text-violet-300">Project Brain</span>
                  <span
                    class="text-[10px] px-1.5 py-0.5 rounded bg-violet-500/20 text-violet-300 font-semibold"
                    >nexus</span
                  >
                  <span
                    v-if="data.nexus.isActive"
                    class="text-[10px] px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-300"
                    >active</span
                  >
                </div>
                <p class="text-[11px] text-slate-300 font-mono mb-1">
                  {{ basename(data.nexus.path) }}
                </p>
                <p v-if="data.nexus.summary" class="text-xs text-slate-400">
                  {{ data.nexus.summary }}
                </p>
              </div>
              <span class="text-[10px] text-slate-600">{{
                formatDate(data.nexus.lastModified)
              }}</span>
            </div>
          </div>

          <!-- Plan groups with task files -->
          <div v-if="data.planGroups.length > 0" class="mb-5">
            <div v-for="[planId, taskFiles] in data.planGroups" :key="planId" class="mb-4">
              <!-- Plan group header -->
              <div class="flex items-center gap-2 mb-2">
                <span class="text-xs font-semibold text-slate-400 font-mono">{{
                  planId === 'ungrouped' ? 'Other Tasks' : planId
                }}</span>
                <span class="text-[10px] text-slate-600"
                  >{{ taskFiles.length }} task{{ taskFiles.length !== 1 ? 's' : '' }}</span
                >
                <div class="flex-1 h-px bg-white/5 ml-1"></div>
              </div>
              <!-- Task file cards (compact) -->
              <div class="grid grid-cols-1 gap-2">
                <div
                  v-for="plan in taskFiles"
                  :key="plan.id"
                  class="rounded-lg border border-white/[0.07] bg-white/[0.015] px-3 py-2 hover:bg-white/[0.03] transition-all flex items-start gap-2"
                >
                  <span
                    class="w-1.5 h-1.5 rounded-full flex-shrink-0 mt-1.5"
                    :class="plan.isActive ? 'bg-emerald-400' : 'bg-slate-700'"
                  ></span>
                  <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2">
                      <span class="text-xs font-medium text-white truncate">{{
                        basename(plan.path)
                      }}</span>
                      <span class="text-[10px] text-slate-600 font-mono ml-auto flex-shrink-0">{{
                        formatDate(plan.lastModified)
                      }}</span>
                    </div>
                    <p v-if="plan.summary" class="text-[11px] text-slate-500 truncate mt-0.5">
                      {{ truncate(plan.summary.replace(/^#+\s*/, ''), 100) }}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Other files (non-nexus, non-claude-task) -->
          <div v-if="data.others.length > 0">
            <div class="flex items-center gap-2 mb-2">
              <span class="text-xs font-semibold text-slate-600 uppercase tracking-wider"
                >Other Files</span
              >
              <div class="flex-1 h-px bg-white/5 ml-1"></div>
            </div>
            <div class="grid grid-cols-1 gap-2">
              <div
                v-for="plan in data.others"
                :key="plan.id"
                class="rounded-lg border border-white/[0.07] bg-white/[0.015] px-3 py-2 hover:bg-white/[0.03] transition-all flex items-start gap-2"
              >
                <span
                  class="w-1.5 h-1.5 rounded-full flex-shrink-0 mt-1.5"
                  :class="plan.isActive ? 'bg-emerald-400' : 'bg-slate-700'"
                ></span>
                <div class="flex-1 min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="text-xs font-medium text-white truncate">{{
                      basename(plan.path)
                    }}</span>
                    <span
                      :class="kindBadgeClass(plan.kind)"
                      class="text-[10px] px-1.5 py-0.5 rounded font-semibold flex-shrink-0"
                      >{{ plan.kind }}</span
                    >
                    <span class="text-[10px] text-slate-600 font-mono ml-auto flex-shrink-0">{{
                      formatDate(plan.lastModified)
                    }}</span>
                  </div>
                  <p v-if="plan.summary" class="text-[11px] text-slate-500 truncate mt-0.5">
                    {{ truncate(plan.summary, 80) }}
                  </p>
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
import { computed } from 'vue';
import { useDiscoveredPlans } from '../composables/useDiscoveredPlans';
import { currentProject } from '../composables/useProjectState';
import type { PlanFileKind, DiscoveredPlanFile } from '../types/domain';

const { plans, loading, error, scan } = useDiscoveredPlans(currentProject);

function extractParentPlan(file: DiscoveredPlanFile): string | null {
  const m = file.summary?.match(/\*\*Plan:\*\*\s*(PLAN-\d+)/);
  if (m) return m[1];
  return null;
}

const projectGroups = computed(() => {
  const map: Record<
    string,
    {
      nexus?: DiscoveredPlanFile;
      planGroups: [string, DiscoveredPlanFile[]][];
      others: DiscoveredPlanFile[];
    }
  > = {};
  const allProjectPaths = [...new Set(plans.value.map((p) => p.projectPath || '(unknown)'))];
  for (const proj of allProjectPaths) {
    const projectPlans = plans.value.filter((p) => p.projectPath === proj);
    const nexus = projectPlans.find((p) => p.kind === 'nexus');
    const taskFiles = projectPlans.filter((p) => p.kind === 'claude-task');
    const others = projectPlans.filter((p) => p.kind !== 'nexus' && p.kind !== 'claude-task');

    const groups: Record<string, DiscoveredPlanFile[]> = {};
    for (const f of taskFiles) {
      const planId = extractParentPlan(f) ?? 'ungrouped';
      if (!groups[planId]) groups[planId] = [];
      groups[planId].push(f);
    }
    for (const key of Object.keys(groups)) {
      groups[key].sort((a, b) => {
        const numA = parseInt(a.path.match(/TASK-(\d+)/)?.[1] ?? '0');
        const numB = parseInt(b.path.match(/TASK-(\d+)/)?.[1] ?? '0');
        return numB - numA;
      });
    }
    const sortedGroups = Object.entries(groups).sort(([a], [b]) => {
      if (a === 'ungrouped') return 1;
      if (b === 'ungrouped') return -1;
      const numA = parseInt(a.match(/PLAN-(\d+)/)?.[1] ?? '0');
      const numB = parseInt(b.match(/PLAN-(\d+)/)?.[1] ?? '0');
      return numB - numA;
    });
    map[proj] = { nexus, planGroups: sortedGroups, others };
  }
  return map;
});

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
