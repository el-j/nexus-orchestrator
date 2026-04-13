<template>
  <div class="mb-5 rounded-xl border border-violet-500/30 bg-violet-500/[0.04] p-3 text-white">
    <div class="flex items-start gap-3">
      <span class="text-lg mt-0.5" :class="isIngestible ? 'animate-pulse' : ''">🧠</span>
      <div class="flex-1">
        <div class="flex items-center gap-2 mb-1">
          <span class="font-bold text-sm text-violet-300">Project Brain</span>
          <span
            class="text-[10px] px-1.5 py-0.5 rounded bg-violet-500/20 text-violet-300 font-semibold"
            v-if="status?.initialized"
            >nexus</span
          >
          <span
            class="text-[10px] px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-300"
            v-if="nexusFile?.isActive"
            >active</span
          >

          <span
            v-if="!status?.initialized && !loading"
            class="text-[10px] px-1.5 py-0.5 rounded bg-amber-500/20 text-amber-300 font-semibold ml-2"
            >uninitialized</span
          >
        </div>

        <p class="text-[11px] text-slate-300 font-mono mb-1 truncate" v-if="nexusFile">
          {{ nexusFile.path.split('/').pop() }}
        </p>

        <p v-if="nexusFile?.summary" class="text-xs text-slate-400 mb-2">
          {{ nexusFile.summary }}
        </p>

        <!-- Brain Stats Section -->
        <div v-if="status?.initialized" class="flex items-center gap-4 mt-2 text-xs">
          <div class="flex flex-col">
            <span class="text-slate-500 text-[10px] uppercase font-bold tracking-wider"
              >Entries</span
            >
            <span class="font-mono text-cyan-300">{{ status.entryCount }}</span>
          </div>
          <div class="flex flex-col">
            <span class="text-slate-500 text-[10px] uppercase font-bold tracking-wider"
              >Tokens</span
            >
            <span class="font-mono text-emerald-300">{{
              status.totalTokens.toLocaleString()
            }}</span>
          </div>
          <div class="flex flex-col" v-if="status.lastUpdated">
            <span class="text-slate-500 text-[10px] uppercase font-bold tracking-wider"
              >Last Sync</span
            >
            <span class="font-mono text-slate-300">{{
              new Date(status.lastUpdated).toLocaleTimeString()
            }}</span>
          </div>
        </div>
      </div>

      <!-- Actions -->
      <div class="flex flex-col items-end gap-2">
        <span class="text-[10px] text-slate-600 font-mono" v-if="nexusFile">{{
          formatDate(nexusFile.lastModified)
        }}</span>

        <button
          v-if="isIngestible"
          @click="handleIngest()"
          :disabled="ingesting"
          class="px-2 py-1 bg-violet-500/20 hover:bg-violet-500/40 text-violet-300 text-xs rounded border border-violet-500/50 transition-all flex items-center justify-center gap-1 min-w-[70px]"
        >
          <i v-if="ingesting" class="pi pi-spin pi-spinner text-[10px]"></i>
          <span>{{ ingesting ? 'Syncing...' : 'Sync Brain' }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import type { DiscoveredPlanFile, BrainStatus } from '../types/domain';
import { getBrainStatus, ingestKnowledge } from '../types/wails';

const props = defineProps<{
  projectPath: string;
  nexusFile?: DiscoveredPlanFile;
}>();

const status = ref<BrainStatus | null>(null);
const loading = ref(true);
const ingesting = ref(false);

const isIngestible = computed(() => {
  return !!props.nexusFile; // If we have a CLAUDE.md file mapped as nexus
});

async function fetchStatus() {
  loading.value = true;
  try {
    status.value = await getBrainStatus(props.projectPath);
  } catch (err) {
    console.warn('Failed to fetch brain status for', props.projectPath, err);
  } finally {
    loading.value = false;
  }
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

async function handleIngest() {
  if (!props.nexusFile) return;
  ingesting.value = true;
  try {
    const changes = await ingestKnowledge(props.projectPath, props.nexusFile.path);
    console.log(`Ingested ${changes} items into brain index`);
    await fetchStatus();
  } catch (err) {
    console.error('Failed to ingest knowledge', err);
  } finally {
    ingesting.value = false;
  }
}

onMounted(() => {
  fetchStatus();
});

watch(() => props.projectPath, fetchStatus);
</script>
