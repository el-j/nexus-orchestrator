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
          <!-- Registered Sessions -->
          <section v-if="sessions.length > 0" class="mb-6">
            <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">
              Registered Sessions
            </h2>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div
                v-for="s in sessions"
                :key="s.id"
                class="rounded-xl border bg-white/[0.03] p-4 transition-all border-l-4"
                :class="{
                  'border-l-emerald-500/60': s.status === 'active',
                  'border-l-yellow-500/60': s.status === 'idle',
                  'border-l-slate-600/60': s.status === 'disconnected',
                }"
              >
                <!-- Header: status dot + agent name + status badge -->
                <div class="flex items-center gap-2 mb-2">
                  <span
                    class="w-2 h-2 rounded-full flex-shrink-0"
                    :class="{
                      'bg-emerald-400': s.status === 'active',
                      'bg-yellow-400': s.status === 'idle',
                      'bg-slate-600': s.status === 'disconnected',
                    }"
                  ></span>
                  <span class="font-semibold text-sm text-white truncate flex-1">{{
                    s.agentName
                  }}</span>
                  <span
                    class="text-[10px] px-1.5 py-0.5 rounded font-medium flex-shrink-0"
                    :class="{
                      'bg-emerald-500/20 text-emerald-300': s.status === 'active',
                      'bg-yellow-500/20 text-yellow-300': s.status === 'idle',
                      'bg-slate-700/50 text-slate-400': s.status === 'disconnected',
                    }"
                    >{{ s.status }}</span
                  >
                </div>

                <!-- Source badge -->
                <div class="flex items-center gap-2 mb-2">
                  <span class="text-[11px] px-2 py-0.5 rounded bg-white/5 text-slate-400">
                    <template v-if="s.source === 'mcp'">🤖 MCP</template>
                    <template v-else-if="s.source === 'vscode'">🔵 VS Code</template>
                    <template v-else>🌐 HTTP</template>
                  </span>
                  <span v-if="s.externalId" class="text-[10px] text-slate-600 font-mono truncate">{{
                    s.externalId
                  }}</span>
                </div>

                <!-- Project path -->
                <div
                  v-if="s.projectPath"
                  class="text-[11px] text-slate-500 font-mono mb-2 truncate"
                  :title="s.projectPath"
                >
                  {{ shortPath(s.projectPath) }}
                </div>

                <!-- Last activity -->
                <div class="text-[11px] text-slate-600 mb-3">
                  Last active: {{ relativeTime(s.lastActivity) }}
                </div>

                <!-- Disconnect button -->
                <button
                  v-if="s.status !== 'disconnected'"
                  class="text-xs text-red-400 hover:text-red-300 px-2.5 py-1 rounded-lg border border-red-500/20 hover:border-red-500/40 transition-all"
                  @click="deregister(s.id)"
                >
                  Disconnect
                </button>
              </div>
            </div>
          </section>

          <!-- Discovered Agents -->
          <section v-if="agents.length > 0">
            <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">
              Discovered Agents
            </h2>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <template v-for="agent in agentTree.roots" :key="agent.id">
                <div class="rounded-xl border border-white/10 bg-white/[0.02] p-4">
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

      <!-- ===== ACTIVITY TAB ===== -->
      <template v-if="activeTab === 'activity'">
        <!-- Filters -->
        <div class="flex items-center gap-3 mb-4 flex-wrap">
          <select
            v-model="activityAgentFilter"
            class="text-xs bg-white/5 border border-white/10 rounded px-2 py-1 text-slate-300 focus:outline-none focus:border-violet-500/40"
          >
            <option value="">All agents</option>
            <option v-for="a in activityAgentOptions" :key="a" :value="a">{{ a }}</option>
          </select>
          <select
            v-model="activityProjectFilter"
            class="text-xs bg-white/5 border border-white/10 rounded px-2 py-1 text-slate-300 focus:outline-none focus:border-violet-500/40"
          >
            <option value="">All projects</option>
            <option v-for="p in activityProjectOptions" :key="p" :value="p">
              {{ shortPath(p) }}
            </option>
          </select>
        </div>

        <!-- Loading -->
        <div
          v-if="activityLoading && activityList.length === 0"
          class="flex items-center justify-center py-12"
        >
          <div
            class="w-5 h-5 border-2 border-violet-500 border-t-transparent rounded-full animate-spin"
          ></div>
        </div>

        <!-- Empty -->
        <div
          v-else-if="filteredActivities.length === 0"
          class="flex flex-col items-center justify-center py-16 text-center"
        >
          <p class="text-sm text-slate-500">No AI activity detected</p>
          <p class="text-xs text-slate-600 mt-1">Start an AI tool and it will appear here</p>
        </div>

        <!-- Activity timeline -->
        <div v-else class="space-y-1.5">
          <div
            v-for="a in filteredActivities"
            :key="a.id"
            class="flex items-center gap-3 px-3 py-2 rounded-lg border border-white/5 bg-white/[0.02] hover:bg-white/[0.04] transition-colors"
          >
            <span class="text-[10px] text-slate-500 shrink-0 w-14 font-mono">{{
              formatActivityTime(a.timestamp)
            }}</span>
            <span
              class="text-xs font-semibold truncate shrink-0 max-w-[8rem]"
              :class="activityAgentColor(a.agentName)"
              >{{ a.agentName }}</span
            >
            <span
              :class="[
                'text-[10px] px-1.5 py-0.5 rounded shrink-0 font-medium',
                activityTypeBadge(a.activityType),
              ]"
              >{{ a.activityType }}</span
            >
            <span class="text-xs text-slate-300 truncate flex-1">{{ a.summary }}</span>
            <span
              v-if="a.projectPath"
              class="text-[10px] text-slate-600 shrink-0 truncate max-w-[8rem]"
              >{{ shortPath(a.projectPath) }}</span
            >
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { useAISessions } from '../composables/useAISessions';
import { useActivities } from '../composables/useActivities';
import type { DiscoveredAgent, ActivityType } from '../types/domain';
import { resolveServerUrl } from '../composables/useServerUrl';
import { relativeTime } from '../utils/time';

const { sessions, loading, error, refresh, deregister, purgeDisconnected } = useAISessions();

// --- Tabs ---
const tabs = [
  { id: 'sessions', label: 'Sessions' },
  { id: 'activity', label: 'Activity' },
];
const activeTab = ref('sessions');

// --- Activity tab ---
const { activities: activityList, loading: activityLoading } = useActivities({ limit: 200 });

const activityAgentFilter = ref('');
const activityProjectFilter = ref('');

const activityAgentOptions = computed(() =>
  [...new Set(activityList.value.map((a) => a.agentName))].sort(),
);
const activityProjectOptions = computed(() =>
  [...new Set(activityList.value.map((a) => a.projectPath).filter(Boolean) as string[])].sort(),
);

const filteredActivities = computed(() => {
  let result = activityList.value;
  if (activityAgentFilter.value)
    result = result.filter((a) => a.agentName === activityAgentFilter.value);
  if (activityProjectFilter.value)
    result = result.filter((a) => a.projectPath === activityProjectFilter.value);
  return result;
});

function formatActivityTime(ts: string): string {
  const d = new Date(ts);
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function activityAgentColor(name: string): string {
  const lower = name.toLowerCase();
  if (lower.includes('claude') || lower.includes('anthropic')) return 'text-violet-400';
  if (lower.includes('copilot') || lower.includes('github')) return 'text-blue-400';
  if (lower.includes('cursor')) return 'text-cyan-400';
  if (lower.includes('continue')) return 'text-emerald-400';
  return 'text-slate-300';
}

function activityTypeBadge(type: ActivityType): string {
  switch (type) {
    case 'message':
      return 'bg-blue-500/20 text-blue-300';
    case 'tool_use':
      return 'bg-green-500/20 text-green-300';
    case 'thinking':
      return 'bg-purple-500/20 text-purple-300';
    case 'file_edit':
      return 'bg-orange-500/20 text-orange-300';
    case 'generation':
      return 'bg-yellow-500/20 text-yellow-300';
    default:
      return 'bg-white/10 text-slate-400';
  }
}

const agents = ref<DiscoveredAgent[]>([]);
let discoveredInterval: ReturnType<typeof setInterval> | null = null;

const activeCount = computed(() => sessions.value.filter((s) => s.status === 'active').length);

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

function shortPath(path: string): string {
  return path.split('/').filter(Boolean).slice(-2).join('/');
}

function shortModelName(id: string): string {
  return id.replace(/^claude-/, '');
}

function shortWorkingDir(dir: string): string {
  return dir.split('/').filter(Boolean).slice(-2).join('/');
}

onMounted(async () => {
  await loadAgents();
  discoveredInterval = setInterval(loadAgents, 30_000);
});

onUnmounted(() => {
  if (discoveredInterval) clearInterval(discoveredInterval);
});
</script>
