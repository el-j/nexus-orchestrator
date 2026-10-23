<!--
  Agents: registered AI sessions and the AI tools discovered on this machine (with their
  sub-agents), plus the activity tab. Sort, filter and search; sessions can be deregistered or purged.
-->
<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <header
      class="flex items-center justify-between px-5 py-3 border-b border-white/5 bg-[#0a0a10] flex-shrink-0"
    >
      <div>
        <h1 class="text-sm font-bold text-white">Agents</h1>
        <p class="text-xs text-slate-500">
          <span
            class="font-semibold"
            :class="activeCount > 0 ? 'text-emerald-400' : 'text-slate-500'"
            >{{ activeCount }}</span
          >
          active ·
          <span class="text-cyan-400 font-semibold">{{ agents.length }}</span>
          discovered
        </p>
      </div>
      <div class="flex items-center gap-2">
        <select
          v-model="sortMode"
          class="text-xs bg-white/5 border border-white/10 rounded-lg px-2 py-1.5 text-slate-300 focus:outline-none focus:border-violet-500/40"
        >
          <option v-for="opt in sortOptions" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </option>
        </select>
        <button
          class="text-xs text-slate-400 hover:text-white px-3 py-1.5 rounded-lg border border-white/10 hover:border-violet-500/40 transition-all"
          @click="refreshAll"
        >
          ⟳ Refresh
        </button>
        <button
          v-if="sessions.some((s) => s.status === 'disconnected')"
          class="text-xs text-red-400 hover:text-red-300 px-3 py-1.5 rounded-lg border border-red-500/20 hover:border-red-500/40 transition-all"
          @click="purgeDisconnected"
        >
          🗑 Clear Disconnected
        </button>
      </div>
    </header>

    <!-- Tab bar -->
    <div class="flex items-center gap-1 px-5 py-2 border-b border-white/5 bg-[#0a0a10] shrink-0">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        @click="activeTab = tab.id"
        :class="[
          'px-3 py-1 rounded-lg text-xs font-medium transition-all',
          activeTab === tab.id
            ? 'bg-violet-600/20 text-violet-300 border border-violet-500/30'
            : 'text-slate-500 hover:text-slate-300 hover:bg-white/5 border border-transparent',
        ]"
      >
        {{ tab.label }}
      </button>
    </div>

    <!-- Filter bar -->
    <div class="flex items-center gap-2 px-5 py-2 border-b border-white/5 bg-[#0a0a10] shrink-0">
      <input
        v-model="searchQuery"
        type="text"
        placeholder="Search…"
        class="text-xs bg-white/5 border border-white/10 rounded-lg px-3 py-1.5 text-slate-300 placeholder-slate-600 focus:outline-none focus:border-violet-500/40 w-48"
      />
      <template v-if="activeTab === 'sessions'">
        <button
          v-for="opt in [
            { v: '', l: 'All' },
            { v: 'active', l: 'Active' },
            { v: 'idle', l: 'Idle' },
            { v: 'disconnected', l: 'Off' },
          ]"
          :key="'ss-' + opt.v"
          @click="sessionStatusFilter = opt.v as any"
          :class="[
            'text-[10px] px-2 py-1 rounded-full border transition-colors font-medium',
            sessionStatusFilter === opt.v
              ? 'bg-violet-600/20 text-violet-300 border-violet-500/30'
              : 'border-white/10 text-slate-500 hover:text-slate-300 hover:border-white/20',
          ]"
        >
          {{ opt.l }}
        </button>
      </template>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-auto p-5">
      <!-- ===== SESSIONS TAB ===== -->
      <template v-if="activeTab === 'sessions'">
        <!-- Loading -->
        <div v-if="loading" class="flex items-center justify-center py-16">
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
          v-else-if="sessions.length === 0 && agents.length === 0"
          class="flex flex-col items-center justify-center py-20 text-center"
        >
          <div class="text-4xl mb-4 opacity-40">🤖</div>
          <p class="text-sm font-medium text-slate-400">No AI agents detected</p>
          <p class="text-xs text-slate-600 mt-1 max-w-sm">
            Connect VS Code Copilot, an MCP client, or another AI agent to see it here.
          </p>
        </div>

        <template v-else>
          <!-- Filtered empty state -->
          <div
            v-if="
              (sessions.length > 0 || agents.length > 0) &&
              filteredSessions.length === 0 &&
              filteredAgentRoots.length === 0
            "
            class="flex flex-col items-center justify-center py-12 text-center"
          >
            <p class="text-sm text-slate-500">No matches found</p>
            <p class="text-xs text-slate-600 mt-1">Try adjusting your search or filters</p>
          </div>

          <!-- Instruction returned by Delegate: the agent has to be told to follow it -->
          <div
            v-if="delegationInstruction"
            data-testid="delegation-note"
            class="mb-4 rounded-xl border border-violet-500/30 bg-violet-500/5 p-3 text-xs text-violet-100"
          >
            <div class="flex items-center justify-between mb-1">
              <span class="font-semibold">Give this to the agent</span>
              <button class="text-slate-400 hover:text-white" @click="delegationInstruction = ''">
                Dismiss
              </button>
            </div>
            <pre class="whitespace-pre-wrap font-mono text-[11px]">{{ delegationInstruction }}</pre>
          </div>

          <!-- Registered Sessions -->
          <section v-if="filteredSessions.length > 0" class="mb-6">
            <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">
              Registered Sessions
            </h2>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <SessionCard
                v-for="s in filteredSessions"
                :key="s.id"
                :session="s"
                @open="openDrawer(s.agentName)"
                @disconnect="deregister(s.id)"
                @delegate="onDelegate(s.id)"
                @terminate="terminate(s.id)"
              />
            </div>
          </section>

          <!-- Discovered Agents -->
          <section v-if="filteredAgentRoots.length > 0">
            <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">
              Discovered Agents
            </h2>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <template v-for="agent in filteredAgentRoots" :key="agent.id">
                <div class="rounded-xl border border-white/10 bg-white/[0.02] p-3">
                  <div class="flex items-center gap-2 mb-1">
                    <span
                      class="w-2 h-2 rounded-full flex-shrink-0"
                      :class="agent.isRunning ? 'bg-emerald-400' : 'bg-slate-600'"
                    ></span>
                    <span class="font-semibold text-sm text-white truncate">{{ agent.name }}</span>
                    <span
                      class="text-[10px] px-1.5 py-0.5 rounded bg-white/5 text-slate-400 ml-auto flex-shrink-0"
                      >{{ agent.kind }}</span
                    >
                    <span
                      v-if="agent.modelId"
                      class="bg-slate-100 text-slate-700 rounded px-1.5 text-xs font-mono flex-shrink-0"
                      >{{ shortModelName(agent.modelId) }}</span
                    >
                  </div>
                  <div class="text-[11px] text-slate-500 mb-1">{{ agent.detectionMethod }}</div>
                  <div v-if="agent.workingDir" class="text-slate-500 text-xs mb-1">
                    {{ shortWorkingDir(agent.workingDir) }}
                  </div>
                  <div
                    v-if="agent.mcpEndpoint"
                    class="text-[10px] text-slate-600 font-mono truncate"
                  >
                    {{ agent.mcpEndpoint }}
                  </div>
                  <div class="flex items-center gap-2 mt-1">
                    <span class="text-[10px] text-slate-700">
                      Last seen {{ relativeTime(agent.lastSeen) }}
                    </span>
                    <span
                      v-if="agent.isRunning"
                      class="text-[10px] px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-300 font-medium"
                      >Running</span
                    >
                  </div>

                  <!-- Sub-agents -->
                  <div
                    v-if="agentTree.childrenOf[agent.id]?.length"
                    class="mt-3 ml-6 border-l-2 border-slate-200/20 pl-2 space-y-2"
                  >
                    <div
                      v-for="sub in agentTree.childrenOf[agent.id]"
                      :key="sub.id"
                      class="rounded-lg border border-white/5 bg-white/[0.015] p-3"
                    >
                      <div class="flex items-center gap-2 mb-1">
                        <span
                          class="w-1.5 h-1.5 rounded-full flex-shrink-0"
                          :class="sub.isRunning ? 'bg-emerald-400' : 'bg-slate-600'"
                        ></span>
                        <span class="font-medium text-sm text-white truncate">{{ sub.name }}</span>
                        <span
                          class="text-[9px] px-1 py-0.5 rounded bg-violet-500/10 text-violet-400 flex-shrink-0"
                          >sub-agent</span
                        >
                        <span
                          class="text-[10px] px-1.5 py-0.5 rounded bg-white/5 text-slate-400 ml-auto flex-shrink-0"
                          >{{ sub.kind }}</span
                        >
                        <span
                          v-if="sub.modelId"
                          class="bg-slate-100 text-slate-700 rounded px-1.5 text-xs font-mono flex-shrink-0"
                          >{{ shortModelName(sub.modelId) }}</span
                        >
                      </div>
                      <div class="text-[10px] text-slate-500 mb-1">{{ sub.detectionMethod }}</div>
                      <div v-if="sub.workingDir" class="text-slate-500 text-xs mb-1">
                        {{ shortWorkingDir(sub.workingDir) }}
                      </div>
                      <div class="text-[10px] text-slate-700 mt-1">
                        Last seen {{ relativeTime(sub.lastSeen) }}
                      </div>
                    </div>
                  </div>
                </div>
              </template>
            </div>
          </section>
        </template>
      </template>

      <AgentActivityTab v-if="activeTab === 'activity'" />
    </div>
  </div>

  <AgentDetailDrawer
    :agent-name="selectedAgent"
    :model-visible="drawerOpen"
    @update:model-visible="drawerOpen = $event"
  />
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { useAISessions } from '../composables/useAISessions';
import AgentDetailDrawer from '../components/AgentDetailDrawer.vue';
import AgentActivityTab from '../components/AgentActivityTab.vue';
import SessionCard from '../components/SessionCard.vue';
import type { DiscoveredAgent } from '../types/domain';
import { resolveServerUrl } from '../composables/useServerUrl';
import { relativeTime } from '../utils/time';
import { currentProject } from '../composables/useProjectState';

const { sessions, loading, error, refresh, deregister, delegate, terminate, purgeDisconnected } =
  useAISessions();

const delegationInstruction = ref('');

async function onDelegate(id: string) {
  const res = await delegate(id);
  delegationInstruction.value = res?.instruction ?? '';
}

// --- Sort ---
const sortMode = ref<'active' | 'recent' | 'name'>('active');
const sortOptions = [
  { value: 'active', label: 'Active first' },
  { value: 'recent', label: 'Recent first' },
  { value: 'name', label: 'Name A-Z' },
] as const;

// --- Filter ---
const searchQuery = ref('');
const sessionStatusFilter = ref<'' | 'active' | 'idle' | 'disconnected'>('');

// --- Tabs ---
const tabs = [
  { id: 'sessions', label: 'Sessions' },
  { id: 'activity', label: 'Activity' },
];
const activeTab = ref('sessions');

const agents = ref<DiscoveredAgent[]>([]);
let discoveredInterval: ReturnType<typeof setInterval> | undefined;

const activeCount = computed(() => sessions.value.filter((s) => s.status === 'active').length);

const statusOrder: Record<string, number> = { active: 0, idle: 1, disconnected: 2 };

const sortedSessions = computed(() => {
  const list = [...sessions.value];
  switch (sortMode.value) {
    case 'active':
      return list.sort(
        (a, b) =>
          (statusOrder[a.status] ?? 9) - (statusOrder[b.status] ?? 9) ||
          new Date(b.lastActivity).getTime() - new Date(a.lastActivity).getTime(),
      );
    case 'recent':
      return list.sort(
        (a, b) => new Date(b.lastActivity).getTime() - new Date(a.lastActivity).getTime(),
      );
    default:
      return list.sort((a, b) => a.agentName.localeCompare(b.agentName));
  }
});

const agentTree = computed(() => {
  const roots: DiscoveredAgent[] = [];
  const childrenOf: Record<string, DiscoveredAgent[]> = {};
  const byId: Record<string, DiscoveredAgent> = {};

  for (const agent of agents.value) {
    byId[agent.id] = agent;
  }

  for (const agent of agents.value) {
    if (agent.parentAgentId && byId[agent.parentAgentId]) {
      if (!childrenOf[agent.parentAgentId]) childrenOf[agent.parentAgentId] = [];
      childrenOf[agent.parentAgentId].push(agent);
    } else {
      roots.push(agent);
    }
  }

  return { roots, childrenOf };
});

const sortedAgentRoots = computed(() => {
  const list = [...agentTree.value.roots];
  switch (sortMode.value) {
    case 'active':
      return list.sort(
        (a, b) =>
          (b.isRunning ? 1 : 0) - (a.isRunning ? 1 : 0) ||
          new Date(b.lastSeen).getTime() - new Date(a.lastSeen).getTime(),
      );
    case 'recent':
      return list.sort((a, b) => new Date(b.lastSeen).getTime() - new Date(a.lastSeen).getTime());
    default:
      return list.sort((a, b) => a.name.localeCompare(b.name));
  }
});

const filteredSessions = computed(() => {
  let list = sortedSessions.value;
  if (currentProject.value !== null) {
    list = list.filter((s) => s.projectPath === currentProject.value);
  }
  if (sessionStatusFilter.value) {
    list = list.filter((s) => s.status === sessionStatusFilter.value);
  }
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase();
    list = list.filter(
      (s) =>
        s.agentName.toLowerCase().includes(q) ||
        (s.externalId && s.externalId.toLowerCase().includes(q)) ||
        (s.projectPath && s.projectPath.toLowerCase().includes(q)),
    );
  }
  return list;
});

const filteredAgentRoots = computed(() => {
  let list = sortedAgentRoots.value;
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase();
    list = list.filter(
      (a) =>
        a.name.toLowerCase().includes(q) ||
        a.kind.toLowerCase().includes(q) ||
        (a.workingDir && a.workingDir.toLowerCase().includes(q)),
    );
  }
  return list;
});

async function loadAgents() {
  try {
    const base = await resolveServerUrl();
    const res = await fetch(`${base}/api/ai-sessions/discovered`);
    if (res.ok) agents.value = (await res.json()) ?? [];
  } catch {
    /* non-critical */
  }
}

async function refreshAll() {
  await Promise.all([refresh(), loadAgents()]);
}

function shortModelName(id: string): string {
  return id.replace(/^claude-/, '');
}

function shortWorkingDir(dir: string): string {
  return dir.split('/').filter(Boolean).slice(-2).join('/');
}

let agentsDisposed = false; // set on unmount so a load that finishes late starts no polling
onMounted(async () => {
  await loadAgents();
  if (agentsDisposed) return;
  discoveredInterval = setInterval(loadAgents, 30_000);
});

onUnmounted(() => {
  agentsDisposed = true;
  clearInterval(discoveredInterval);
});

const selectedAgent = ref('');
const drawerOpen = ref(false);
function openDrawer(agentName: string) {
  selectedAgent.value = agentName;
  drawerOpen.value = true;
}
</script>
