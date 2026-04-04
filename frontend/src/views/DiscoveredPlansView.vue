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
          @click="allCollapsed ? expandAll() : collapseAll()"
        >
          {{ allCollapsed ? '⊞ Expand All' : '⊟ Collapse All' }}
        </button>
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
      <!-- Kind filter chips — only when multiple kinds available -->
      <div
        v-if="availableKinds.length > 1"
        class="flex flex-wrap gap-1.5 mb-4 pb-3 border-b border-white/5"
      >
        <button
          v-for="kind in availableKinds"
          :key="kind"
          :class="[
            'text-[10px] px-2.5 py-1 rounded-full border transition-colors font-medium',
            activeFilters.has(kind)
              ? kindBadgeClass(kind)
              : 'border-white/10 text-slate-500 hover:text-slate-300 hover:border-white/20',
          ]"
          @click="toggleFilter(kind)"
        >
          {{ kindLabel(kind) }}
        </button>
        <button
          v-if="activeFilters.size > 0"
          class="text-[10px] px-2.5 py-1 rounded-full border border-white/10 text-slate-500 hover:text-slate-300 transition-colors"
          @click="activeFilters = new Set()"
        >
          Clear filters
        </button>
      </div>

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
        <p class="text-sm font-medium text-slate-400">No plan files found.</p>
        <p class="text-xs text-slate-600 mt-1 max-w-sm">
          Create a <span class="font-mono text-slate-400">.claude/plans/PLAN-NNN.md</span> file to
          get started.
        </p>
      </div>

      <!-- Grouped by project -->
      <template v-else>
        <section v-for="(data, projPath) in filteredProjectGroups" :key="projPath" class="mb-6">
          <!-- Project path header (clickable, collapsible) -->
          <button
            class="flex items-center gap-2 w-full text-left mb-3 group"
            @click="toggleProject(String(projPath))"
          >
            <span
              class="text-xs text-slate-600 transition-transform"
              :class="collapsedProjects.has(String(projPath)) ? '' : 'rotate-90'"
              >▶</span
            >
            <h2
              class="text-xs font-semibold text-slate-500 uppercase tracking-wider font-mono truncate"
            >
              {{ String(projPath).split('/').filter(Boolean).slice(-2).join('/') }}
            </h2>
            <span class="text-[10px] text-slate-600 shrink-0">
              {{ projectStats(data).planCount }} plans · {{ projectStats(data).taskCount }} tasks
              <template v-if="projectStats(data).otherCount">
                · {{ projectStats(data).otherCount }} other</template
              >
            </span>
            <span
              v-if="projectStats(data).hasActive"
              class="w-1.5 h-1.5 rounded-full bg-emerald-400 shrink-0"
            ></span>
            <div class="flex-1 h-px bg-white/5 ml-1"></div>
          </button>

          <!-- Collapsible content -->
          <div v-show="!collapsedProjects.has(String(projPath))">
            <!-- Project Brain card (nexus) -->
            <div
              v-if="data.nexus"
              class="mb-5 rounded-xl border border-violet-500/30 bg-violet-500/[0.04] p-3"
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
              <div
                v-for="[planId, taskFiles] in data.planGroups"
                :key="planId"
                :class="[
                  'mb-4 pl-2',
                  taskFiles.some((f) => ['todo', 'in-progress'].includes(parseStatus(f) ?? ''))
                    ? 'border-l-2 border-violet-500/50'
                    : 'border-l-2 border-transparent',
                ]"
              >
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
                      <div class="flex items-center gap-2 flex-wrap">
                        <span class="text-xs font-medium text-white truncate">{{
                          basename(plan.path)
                        }}</span>
                        <span
                          v-if="parseStatus(plan)"
                          :class="statusBadgeClass(parseStatus(plan)!)"
                          class="text-[10px] px-1.5 py-0.5 rounded font-semibold flex-shrink-0"
                          >{{ parseStatus(plan) }}</span
                        >
                        <span class="text-[10px] text-slate-600 font-mono ml-auto flex-shrink-0">{{
                          formatDate(plan.lastModified)
                        }}</span>
                      </div>
                      <p v-if="plan.summary" class="text-[11px] text-slate-500 truncate mt-0.5">
                        {{ truncate(plan.summary.replace(/^#+\s*/, ''), 100) }}
                      </p>
                    </div>
                    <button
                      v-if="parseStatus(plan) === 'todo'"
                      class="text-[10px] px-2 py-1 rounded border border-violet-500/40 text-violet-400 hover:bg-violet-500/20 transition-colors flex-shrink-0 mt-0.5"
                      :disabled="promoting === plan.id"
                      @click="handlePromote(plan)"
                    >
                      {{ promoting === plan.id ? '…' : 'Promote' }}
                    </button>
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
          </div>
        </section>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useDiscoveredPlans } from '../composables/useDiscoveredPlans';
import { currentProject } from '../composables/useProjectState';
import { createDraft } from '../types/wails';
import type { PlanFileKind, DiscoveredPlanFile } from '../types/domain';

const { plans, loading, error, scan } = useDiscoveredPlans(currentProject);

// --- Collapse state ---
const collapsedProjects = ref<Set<string>>(new Set());

function toggleProject(projPath: string) {
  if (collapsedProjects.value.has(projPath)) {
    collapsedProjects.value.delete(projPath);
  } else {
    collapsedProjects.value.add(projPath);
  }
  collapsedProjects.value = new Set(collapsedProjects.value);
}

function collapseAll() {
  collapsedProjects.value = new Set(Object.keys(filteredProjectGroups.value));
}

function expandAll() {
  collapsedProjects.value = new Set();
}

const allCollapsed = computed(() => {
  const keys = Object.keys(filteredProjectGroups.value);
  return keys.length > 0 && keys.every((k) => collapsedProjects.value.has(k));
});

function projectStats(data: {
  nexus?: DiscoveredPlanFile;
  planGroups: [string, DiscoveredPlanFile[]][];
  others: DiscoveredPlanFile[];
}) {
  const planCount = data.planGroups.length;
  const taskCount = data.planGroups.reduce((sum, [, files]) => sum + files.length, 0);
  const otherCount = data.others.length;
  const hasActive =
    data.nexus?.isActive ||
    data.planGroups.some(([, files]) => files.some((f) => f.isActive)) ||
    data.others.some((f) => f.isActive);
  return { planCount, taskCount, otherCount, hasActive };
}

// --- Kind filtering ---
const activeFilters = ref<Set<PlanFileKind>>(new Set());

function toggleFilter(kind: PlanFileKind) {
  if (activeFilters.value.has(kind)) {
    activeFilters.value.delete(kind);
  } else {
    activeFilters.value.add(kind);
  }
  // Trigger Vue reactivity on Set change
  activeFilters.value = new Set(activeFilters.value);
}

function kindLabel(kind: PlanFileKind): string {
  const labels: Record<PlanFileKind, string> = {
    nexus: 'Nexus',
    'claude-task': 'Tasks',
    claude: 'Instructions',
    cursor: 'Cursor',
    'mcp-config': 'Config',
    crewai: 'CrewAI',
    markdown: 'Markdown',
  };
  return labels[kind] ?? kind;
}

const availableKinds = computed<PlanFileKind[]>(() => {
  const kinds = new Set<PlanFileKind>();
  for (const plan of plans.value) kinds.add(plan.kind);
  return [
    ...[...kinds].filter((k) => k === 'nexus'),
    ...[...kinds].filter((k) => k === 'claude-task'),
    ...[...kinds].filter((k) => k === 'claude'),
    ...[...kinds].filter((k) => k === 'cursor'),
    ...[...kinds].filter((k) => k === 'mcp-config'),
    ...[...kinds].filter((k) => k === 'crewai'),
    ...[...kinds].filter((k) => k === 'markdown'),
  ];
});

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

// Apply active filters to projectGroups
const filteredProjectGroups = computed(() => {
  if (activeFilters.value.size === 0) return projectGroups.value;
  const filtered: typeof projectGroups.value = {};
  for (const [proj, group] of Object.entries(projectGroups.value)) {
    const nexus = group.nexus && activeFilters.value.has('nexus') ? group.nexus : undefined;
    const planGroups = group.planGroups
      .map(
        ([planId, files]) =>
          [planId, files.filter((f) => activeFilters.value.has('claude-task'))] as [
            string,
            DiscoveredPlanFile[],
          ],
      )
      .filter(([_, files]) => files.length > 0);
    const others = group.others.filter((f) => activeFilters.value.has(f.kind));
    if (nexus || planGroups.length > 0 || others.length > 0) {
      filtered[proj] = { nexus, planGroups, others };
    }
  }
  return filtered;
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

// Status parsing from plan file summary (YAML frontmatter or body-level)
function parseStatus(plan: DiscoveredPlanFile): string | null {
  if (!plan.summary) return null;
  // YAML frontmatter: "status: todo"
  const yamlMatch = plan.summary.match(/\bstatus:\s*([^\n\r]+)/i);
  if (yamlMatch) return yamlMatch[1].trim().toLowerCase();
  // Body-level: "**Status:** todo"
  const bodyMatch = plan.summary.match(/\*\*Status:\*\*\s*([^\n\r]+)/i);
  if (bodyMatch) return bodyMatch[1].trim().toLowerCase();
  return null;
}

const statusBadgeColors: Record<string, string> = {
  todo: 'bg-yellow-500/20 text-yellow-300',
  'in-progress': 'bg-blue-500/20 text-blue-300',
  done: 'bg-emerald-500/20 text-emerald-300',
  completed: 'bg-emerald-500/20 text-emerald-300',
  blocked: 'bg-red-500/20 text-red-300',
};

function statusBadgeClass(status: string): string {
  return statusBadgeColors[status] ?? 'bg-white/5 text-slate-400';
}

// Promote plan task to nexus queue
const promoting = ref<string | null>(null);

async function handlePromote(plan: DiscoveredPlanFile) {
  promoting.value = plan.id;
  try {
    await createDraft({
      projectPath: plan.projectPath,
      instruction: plan.summary ?? `Execute task from ${plan.path}`,
      targetFile: plan.path,
    });
  } finally {
    promoting.value = null;
  }
}

// Auto-collapse projects without active items on first load
let initialized = false;
watch(
  filteredProjectGroups,
  (groups) => {
    if (initialized || Object.keys(groups).length === 0) return;
    initialized = true;
    for (const [projPath, data] of Object.entries(groups)) {
      const stats = projectStats(data);
      if (!stats.hasActive) {
        collapsedProjects.value.add(projPath);
      }
    }
    collapsedProjects.value = new Set(collapsedProjects.value);
  },
  { immediate: true },
);
</script>
