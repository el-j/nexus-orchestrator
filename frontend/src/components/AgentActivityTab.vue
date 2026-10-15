<!--
  Activity tab of the Agents view: a filterable feed (agent, type, project) of what connected
  agents have been doing.
-->
<template>
  <div>
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
  </div>
</template>
<script setup lang="ts">
import { ref, computed } from 'vue';
import { useActivities } from '../composables/useActivities';
import type { ActivityType } from '../types/domain';
import { shortPath } from '../utils/path';

/** The Activity tab of the Agents view: a filterable timeline of what the AI tools did. */
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
</script>
