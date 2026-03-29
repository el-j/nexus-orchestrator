<template>
  <div class="flex flex-col h-full overflow-hidden">
    <header
      class="flex items-center justify-between px-5 py-3 border-b border-white/5 bg-[#0a0a10] shrink-0"
    >
      <div>
        <h1 class="text-sm font-bold text-white">Live Activity</h1>
        <p class="text-xs text-slate-500">
          <span class="font-semibold text-emerald-400">{{ activeCount }}</span> agents active ·
          <span class="font-semibold text-violet-400">{{ filtered.length }}</span> activities
        </p>
      </div>
      <button
        class="text-xs text-slate-400 hover:text-white px-3 py-1.5 rounded-lg border border-white/10 hover:border-violet-500/40 transition-all"
        :disabled="scanning"
        @click="triggerScan"
      >
        {{ scanning ? '⏳ Scanning…' : '⟳ Scan Now' }}
      </button>
      <RefreshIndicator :last-refreshed="lastRefreshed" />
    </header>

    <div class="flex items-center gap-3 px-5 py-2 border-b border-white/5 shrink-0 flex-wrap">
      <select
        v-model="selectedAgent"
        class="text-xs bg-white/5 border border-white/10 rounded px-2 py-1 text-slate-300 focus:outline-none focus:border-violet-500/40"
      >
        <option value="">All agents</option>
        <option v-for="a in agentOptions" :key="a" :value="a">{{ a }}</option>
      </select>

      <select
        v-model="selectedProject"
        class="text-xs bg-white/5 border border-white/10 rounded px-2 py-1 text-slate-300 focus:outline-none focus:border-violet-500/40"
      >
        <option value="">All projects</option>
        <option v-for="p in projectOptions" :key="p" :value="p">{{ shortPath(p) }}</option>
      </select>

      <div class="flex items-center gap-1">
        <button
          v-for="t in typeOptions"
          :key="t.type"
          class="text-xs px-2 py-0.5 rounded border transition-all"
          :class="
            selectedTypes.includes(t.type)
              ? 'border-violet-500/60 bg-violet-500/15 text-violet-300'
              : 'border-white/10 text-slate-500 hover:text-slate-300'
          "
          :title="t.type"
          @click="toggleType(t.type)"
        >
          {{ t.emoji }}
        </button>
      </div>
    </div>

    <div class="flex-1 overflow-auto p-4">
      <template v-if="loading && activities.length === 0">
        <div class="flex items-center justify-center py-12">
          <div
            class="w-5 h-5 border-2 border-violet-500 border-t-transparent rounded-full animate-spin"
          ></div>
        </div>
      </template>

      <template v-else-if="filtered.length === 0">
        <div class="flex flex-col items-center justify-center py-16 text-center">
          <p class="text-sm text-slate-500">
            No AI activity detected — start an AI tool and it will appear here
          </p>
          <p class="text-xs text-slate-600 mt-1">Start Claude, Continue, or LM Studio to begin</p>
        </div>
      </template>

      <div
        v-for="group in sessionGroups"
        :key="group.sessionId"
        class="mb-4 rounded-xl border border-white/[0.06] bg-white/[0.01] overflow-hidden"
      >
        <!-- Session header -->
        <div
          class="flex items-center gap-3 px-4 py-2.5 bg-white/[0.03] border-b border-white/[0.06]"
        >
          <div
            class="w-2 h-2 rounded-full shrink-0"
            :class="isRecent(group.latestTime) ? 'bg-emerald-400' : 'bg-slate-600'"
          ></div>
          <span class="text-xs font-semibold truncate" :class="agentColor(group.agentName)">{{
            group.agentName
          }}</span>
          <span v-if="group.model" class="text-[10px] font-mono text-slate-500 shrink-0">{{
            group.model
          }}</span>
          <span v-if="group.totalTokens > 0" class="text-[10px] text-slate-500 shrink-0"
            >🪙 {{ group.totalTokens.toLocaleString() }}</span
          >
          <span class="ml-auto text-[10px] text-slate-500 shrink-0">{{
            timeAgo(group.latestTime)
          }}</span>
          <span v-if="group.projectPath" class="text-[10px] text-slate-600 shrink-0">{{
            shortPath(group.projectPath)
          }}</span>
        </div>
        <!-- Activities within group -->
        <div class="divide-y divide-white/[0.04]">
          <div v-for="a in group.activities" :key="a.id" class="px-3 py-1.5">
            <AIActivityCard :activity="a" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue';
import AIActivityCard from '../components/AIActivityCard.vue';
import RefreshIndicator from '../components/RefreshIndicator.vue';
import { useActivities } from '../composables/useActivities';
import { useDiscovery } from '../composables/useDiscovery';
import type { AIActivity, ActivityType } from '../types/domain';
import { timeAgo } from '../utils/time';

const { scanning, scanNow: originalScan } = useDiscovery();
const { activities, loading } = useActivities({ limit: 100 });

const lastRefreshed = ref<Date | null>(null);

async function triggerScan() {
  await originalScan();
  lastRefreshed.value = new Date();
}

onMounted(() => {
  lastRefreshed.value = new Date();
});

const selectedAgent = ref('');
const selectedProject = ref('');
const selectedTypes = ref<ActivityType[]>([]);

const typeOptions: { type: ActivityType; emoji: string }[] = [
  { type: 'message', emoji: '💬' },
  { type: 'tool_use', emoji: '🔧' },
  { type: 'thinking', emoji: '🧠' },
  { type: 'file_edit', emoji: '📄' },
  { type: 'generation', emoji: '⚡' },
];

function toggleType(t: ActivityType) {
  const idx = selectedTypes.value.indexOf(t);
  if (idx === -1) {
    selectedTypes.value.push(t);
    return;
  }
  selectedTypes.value.splice(idx, 1);
}

const agentOptions = computed(() => [...new Set(activities.value.map((a) => a.agentName))].sort());

const projectOptions = computed(() =>
  [...new Set(activities.value.map((a) => a.projectPath).filter(Boolean) as string[])].sort(),
);

const filtered = computed(() => {
  let result = activities.value;

  if (selectedAgent.value) {
    result = result.filter((a) => a.agentName === selectedAgent.value);
  }

  if (selectedProject.value) {
    result = result.filter((a) => a.projectPath === selectedProject.value);
  }

  if (selectedTypes.value.length > 0) {
    result = result.filter((a) => selectedTypes.value.includes(a.activityType));
  }

  return result;
});

const activeCount = computed(() => new Set(activities.value.map((a) => a.agentName)).size);

function shortPath(projectPath: string): string {
  const parts = projectPath.replace(/\\/g, '/').split('/').filter(Boolean);
  return parts.slice(-2).join('/');
}

interface SessionGroup {
  sessionId: string;
  agentName: string;
  projectPath: string;
  model: string;
  latestTime: string;
  totalTokens: number;
  activities: AIActivity[];
}

const sessionGroups = computed((): SessionGroup[] => {
  const acts = filtered.value;
  const groups = new Map<string, SessionGroup>();
  const openGroups = new Map<string, { groupKey: string; lastTime: number }>();

  // Sort ascending by timestamp for the grouping algorithm
  const sorted = [...acts].sort(
    (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime(),
  );

  for (const activity of sorted) {
    let groupKey: string;

    if (activity.sessionId) {
      groupKey = activity.sessionId;
    } else {
      // Time-based fallback: group consecutive same agent+project within 60 seconds
      const comboKey = `${activity.agentName}|${activity.projectPath ?? ''}`;
      const open = openGroups.get(comboKey);
      const actTime = new Date(activity.timestamp).getTime();

      if (open && actTime - open.lastTime <= 60_000) {
        groupKey = open.groupKey;
        open.lastTime = actTime;
      } else {
        groupKey = `${comboKey}|${activity.timestamp}`;
        openGroups.set(comboKey, { groupKey, lastTime: actTime });
      }
    }

    if (!groups.has(groupKey)) {
      groups.set(groupKey, {
        sessionId: groupKey,
        agentName: activity.agentName,
        projectPath: activity.projectPath ?? '',
        model: activity.model ?? '',
        latestTime: activity.timestamp,
        totalTokens: 0,
        activities: [],
      });
    }

    const group = groups.get(groupKey)!;
    group.activities.push(activity);
    group.totalTokens += (activity.tokensIn ?? 0) + (activity.tokensOut ?? 0);

    if (new Date(activity.timestamp) > new Date(group.latestTime)) {
      group.latestTime = activity.timestamp;
    }
    if (!group.model && activity.model) group.model = activity.model;
  }

  // Within each group: newest first. Groups sorted newest-first.
  return Array.from(groups.values())
    .map((g) => ({ ...g, activities: g.activities.slice().reverse() }))
    .sort((a, b) => new Date(b.latestTime).getTime() - new Date(a.latestTime).getTime());
});

function agentColor(name: string): string {
  const lower = name.toLowerCase();
  if (lower.includes('claude') || lower.includes('anthropic')) return 'text-violet-400';
  if (lower.includes('copilot') || lower.includes('github')) return 'text-blue-400';
  if (lower.includes('cursor')) return 'text-cyan-400';
  if (lower.includes('continue')) return 'text-emerald-400';
  if (lower.includes('lmstudio') || lower.includes('lm studio')) return 'text-orange-400';
  if (lower.includes('ollama')) return 'text-green-400';
  return 'text-slate-300';
}

function isRecent(timestamp: string): boolean {
  return Date.now() - new Date(timestamp).getTime() < 60_000;
}
</script>
