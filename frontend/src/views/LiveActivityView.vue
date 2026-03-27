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

    <div class="flex-1 overflow-auto p-4 space-y-2">
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

      <AIActivityCard v-for="a in filtered" :key="a.id" :activity="a" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import AIActivityCard from '../components/AIActivityCard.vue';
import { useActivities } from '../composables/useActivities';
import { useDiscovery } from '../composables/useDiscovery';
import type { ActivityType } from '../types/domain';

const { scanning, scanNow: triggerScan } = useDiscovery();
const { activities, loading } = useActivities({ limit: 100 });

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
</script>
