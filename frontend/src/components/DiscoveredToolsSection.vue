<!--
  Section of Providers that lists AI coding tools found on this machine (Claude, Copilot, Cursor,
  ...) grouped by kind, with search and a running/installed/detected filter.
  Props: sortMode ('active' lists running tools first, 'name' sorts alphabetically).
-->
<template>
  <section class="mt-8">
    <div class="flex items-center justify-between mb-3">
      <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider">AI Coding Tools</h2>
      <span class="text-xs text-slate-600">detected on this system</span>
    </div>
    <div class="flex items-center gap-2 mb-3">
      <input
        v-model="toolSearch"
        type="text"
        placeholder="Search tools…"
        class="text-xs bg-white/5 border border-white/10 rounded-lg px-2.5 py-1 text-slate-300 placeholder-slate-600 focus:outline-none focus:border-violet-500/40 w-40"
      />
      <button
        v-for="opt in [
          { v: '', l: 'All' },
          { v: 'running', l: 'Running' },
          { v: 'installed', l: 'Installed' },
          { v: 'detected', l: 'Detected' },
        ]"
        :key="'tf-' + opt.v"
        @click="toolStatusFilter = opt.v as any"
        :class="[
          'text-[10px] px-2 py-1 rounded-full border transition-colors font-medium',
          toolStatusFilter === opt.v
            ? 'bg-violet-600/20 text-violet-300 border-violet-500/30'
            : 'border-white/10 text-slate-500 hover:text-slate-300',
        ]"
      >
        {{ opt.l }}
      </button>
    </div>
    <div v-if="agents.length === 0" class="text-xs text-slate-600 italic p-3">
      No AI coding tools detected — install GitHub Copilot, Continue, or run Claude CLI
    </div>
    <div v-else-if="filteredAgents.length === 0" class="text-xs text-slate-500 p-3">
      No matching tools
    </div>
    <div v-else class="space-y-4">
      <div v-for="[kind, kindAgents] in groupedAgents" :key="kind">
        <!-- Group header -->
        <div class="flex items-center gap-2 mb-2">
          <span class="text-xs font-semibold text-slate-400">{{
            agentLabel({ kind } as any)
          }}</span>
          <span class="text-[10px] text-slate-600">({{ kindAgents.length }})</span>
          <div class="flex-1 h-px bg-white/5 ml-1"></div>
        </div>
        <p v-if="kindDescriptions[kind]" class="text-[10px] text-slate-600 mb-2 -mt-1">
          {{ kindDescriptions[kind] }}
        </p>
        <!-- Dense card grid -->
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2">
          <div
            v-for="agent in kindAgents"
            :key="agent.id"
            class="rounded-lg border border-white/[0.07] bg-white/[0.02] p-3 hover:bg-white/[0.04] transition-colors"
          >
            <div class="flex items-center justify-between mb-1">
              <div class="flex items-center gap-1.5 min-w-0">
                <span
                  class="w-1.5 h-1.5 rounded-full flex-shrink-0"
                  :class="agent.isRunning ? 'bg-emerald-400' : 'bg-slate-600'"
                ></span>
                <span class="font-medium text-xs text-white truncate">{{
                  agent.name || agentLabel(agent)
                }}</span>
              </div>
              <span
                :class="agentStatusClass(agent)"
                class="text-[9px] px-1.5 py-0.5 rounded font-semibold shrink-0"
              >
                {{
                  agent.isRunning
                    ? 'Active'
                    : agent.detectionMethod === 'vscode-extension'
                      ? 'Installed'
                      : 'Detected'
                }}
              </span>
            </div>
            <p class="text-[10px] text-slate-500 font-mono truncate">
              {{ agent.configPath || agent.workingDir || agent.detectionMethod }}
            </p>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import type { DiscoveredAgent } from '../types/domain';
import { resolveServerUrl } from '../composables/useServerUrl';

/** The "AI Coding Tools" section of the Providers view: tools detected on this system, searchable and grouped by kind. */
const props = defineProps<{ sortMode: 'active' | 'name' }>();

const toolSearch = ref('');
const toolStatusFilter = ref<'' | 'running' | 'installed' | 'detected'>('');

const sortedAgents = computed(() => {
  const list = [...agents.value];
  if (props.sortMode === 'active') {
    return list.sort(
      (a, b) =>
        (b.isRunning ? 1 : 0) - (a.isRunning ? 1 : 0) ||
        (a.name ?? a.kind).localeCompare(b.name ?? b.kind),
    );
  }
  return list.sort((a, b) => (a.name ?? a.kind).localeCompare(b.name ?? b.kind));
});

const filteredAgents = computed(() => {
  let list = sortedAgents.value;
  if (toolStatusFilter.value === 'running') list = list.filter((a) => a.isRunning);
  else if (toolStatusFilter.value === 'installed')
    list = list.filter((a) => !a.isRunning && a.detectionMethod === 'vscode-extension');
  else if (toolStatusFilter.value === 'detected')
    list = list.filter((a) => !a.isRunning && a.detectionMethod !== 'vscode-extension');
  if (toolSearch.value) {
    const q = toolSearch.value.toLowerCase();
    list = list.filter(
      (a) => (a.name ?? '').toLowerCase().includes(q) || a.kind.toLowerCase().includes(q),
    );
  }
  return list;
});

const kindDescriptions: Record<string, string> = {
  copilot: 'AI pair programmer — inline suggestions, chat, and code generation',
  'claude-cli': 'Terminal AI assistant — task automation, code review, file editing',
  continue: 'Open-source AI code assistant — autocomplete, chat, and model flexibility',
  cline: 'VS Code AI agent — autonomous code changes with approval workflow',
  cursor: 'AI-native code editor — smart rewrites and multi-file edits',
};

const groupedAgents = computed(() => {
  const groups: Record<string, DiscoveredAgent[]> = {};
  for (const agent of filteredAgents.value) {
    const key = agent.kind || 'other';
    if (!groups[key]) groups[key] = [];
    groups[key].push(agent);
  }
  // Sort groups: groups with running agents first, then alphabetical
  return Object.entries(groups).sort(([aKey, aList], [bKey, bList]) => {
    const aHasRunning = aList.some((a) => a.isRunning) ? 1 : 0;
    const bHasRunning = bList.some((b) => b.isRunning) ? 1 : 0;
    if (bHasRunning !== aHasRunning) return bHasRunning - aHasRunning;
    return aKey.localeCompare(bKey);
  });
});

const agents = ref<DiscoveredAgent[]>([]);
async function loadAgents() {
  try {
    const base = await resolveServerUrl();
    const res = await fetch(`${base}/api/ai-sessions/discovered`);
    if (res.ok) agents.value = ((await res.json()) as DiscoveredAgent[] | null) ?? [];
  } catch {
    /* the tools list is informational; keep what we have while the daemon is unreachable */
  }
}

function agentLabel(a: DiscoveredAgent): string {
  const labels: Record<string, string> = {
    copilot: 'GitHub Copilot',
    continue: 'Continue',
    'claude-cli': 'Claude CLI',
    cline: 'Cline',
    cursor: 'Cursor',
  };
  return labels[a.kind] ?? a.name ?? a.kind;
}

function agentStatusClass(a: DiscoveredAgent): string {
  if (a.isRunning) return 'bg-emerald-500/20 text-emerald-300';
  if (a.detectionMethod === 'vscode-extension') return 'bg-amber-500/20 text-amber-300';
  return 'bg-slate-500/20 text-slate-400';
}

onMounted(loadAgents);
</script>
