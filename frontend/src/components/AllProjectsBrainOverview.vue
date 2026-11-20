<!--
  AllProjectsBrainOverview: workspace-level overview showing all discovered projects,
  their brain indexation status, entry/token stats, and quick links to inspect each project brain.
-->
<template>
  <div class="space-y-4">
    <!-- Workspace summary banner -->
    <div class="rounded-xl border border-violet-500/30 bg-violet-500/[0.04] p-4 text-white">
      <div class="flex items-center justify-between flex-wrap gap-3">
        <div class="flex items-center gap-3">
          <span class="text-2xl">🧠</span>
          <div>
            <h2 class="text-sm font-bold text-violet-300">Workspace Brain Overview</h2>
            <p class="text-xs text-slate-400 mt-0.5">
              <span class="text-white font-semibold">{{ initializedCount }}</span> of
              <span class="text-white font-semibold">{{ projectList.length }}</span> projects
              indexed ·
              <span class="text-cyan-300 font-mono font-semibold">{{ totalEntries }}</span> entries
              ·
              <span class="text-emerald-300 font-mono font-semibold">{{
                totalTokens.toLocaleString()
              }}</span>
              tokens
            </p>
          </div>
        </div>
        <div v-if="loading" class="text-xs text-violet-400 flex items-center gap-1.5 font-mono">
          <span
            class="w-3 h-3 border-2 border-violet-400 border-t-transparent rounded-full animate-spin"
          ></span>
          Scanning projects…
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <div
      v-if="projectList.length === 0 && !loading"
      class="text-xs text-slate-500 text-center py-12 rounded-xl border border-white/5 bg-white/[0.02]"
    >
      No projects discovered in workspace. Start by submitting a task or creating a plan file.
    </div>

    <!-- Projects Grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
      <div
        v-for="proj in projectList"
        :key="proj"
        class="rounded-xl border border-white/10 bg-white/[0.02] hover:bg-white/[0.04] hover:border-violet-500/30 p-3 flex flex-col justify-between transition-all"
        :data-testid="`brain-card-${baseName(proj)}`"
      >
        <div>
          <div class="flex items-center justify-between gap-2 mb-1.5">
            <span class="text-xs font-bold text-white truncate" :title="proj">{{
              baseName(proj)
            }}</span>
            <span
              v-if="statuses[proj]?.initialized"
              class="text-[10px] px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-300 font-mono font-semibold shrink-0"
            >
              indexed
            </span>
            <span
              v-else
              class="text-[10px] px-1.5 py-0.5 rounded bg-white/5 text-slate-500 font-mono shrink-0"
            >
              unindexed
            </span>
          </div>
          <p class="text-[10px] text-slate-500 font-mono truncate mb-2" :title="proj">{{ proj }}</p>

          <div v-if="statuses[proj]?.initialized" class="flex items-center gap-3 text-xs mb-3">
            <span class="text-slate-400 text-[11px] font-mono">
              <strong class="text-cyan-300">{{ statuses[proj]?.entryCount }}</strong> entries
            </span>
            <span class="text-slate-400 text-[11px] font-mono">
              <strong class="text-emerald-300">{{
                statuses[proj]?.totalTokens?.toLocaleString()
              }}</strong>
              t
            </span>
          </div>
          <div v-else class="text-[11px] text-slate-600 mb-3 italic">No knowledge indexed yet</div>
        </div>

        <div class="flex items-center justify-between pt-2 border-t border-white/5">
          <span class="text-[10px] text-slate-600 font-mono" v-if="statuses[proj]?.lastUpdated">
            {{ timeAgo(statuses[proj]!.lastUpdated) }}
          </span>
          <span v-else></span>
          <button
            @click="$emit('selectProject', proj)"
            class="text-xs font-medium text-violet-300 hover:text-white px-2 py-1 rounded bg-violet-500/20 hover:bg-violet-500/40 border border-violet-500/30 transition-colors ml-auto"
          >
            Open Brain →
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue';
import type { BrainStatus } from '../types/domain';
import { getBrainStatus } from '../types/wails';
import { baseName } from '../utils/path';
import { timeAgo } from '../utils/time';

const props = defineProps<{
  projectList: string[];
}>();

defineEmits<{
  (e: 'selectProject', path: string): void;
}>();

const statuses = ref<Record<string, BrainStatus>>({});
const loading = ref(false);

const totalEntries = computed(() =>
  Object.values(statuses.value).reduce((sum, s) => sum + s.entryCount, 0),
);

const totalTokens = computed(() =>
  Object.values(statuses.value).reduce((sum, s) => sum + s.totalTokens, 0),
);

const initializedCount = computed(
  () => Object.values(statuses.value).filter((s) => s.initialized).length,
);

async function loadAllStatuses() {
  if (props.projectList.length === 0) return;
  loading.value = true;
  try {
    const results = await Promise.allSettled(
      props.projectList.map((p) => getBrainStatus(p).catch(() => null)),
    );
    results.forEach((res, i) => {
      const p = props.projectList[i];
      if (res.status === 'fulfilled' && res.value) {
        statuses.value[p] = res.value;
      }
    });
  } finally {
    loading.value = false;
  }
}

watch(() => props.projectList, loadAllStatuses, { deep: true });

onMounted(loadAllStatuses);
</script>
