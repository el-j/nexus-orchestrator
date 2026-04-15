<template>
  <div class="mb-5 rounded-xl border border-violet-500/30 bg-violet-500/[0.04] p-3 text-white">
    <div class="flex items-start gap-3">
      <span class="text-lg mt-0.5" :class="ingesting ? 'animate-pulse' : ''">🧠</span>
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

        <!-- Ingest progress / result feedback -->
        <p v-if="ingestProgress" class="text-[11px] text-violet-300 font-mono mt-1">
          {{ ingestProgress }}
        </p>
        <p v-if="ingestResult" class="text-[11px] text-emerald-300 font-mono mt-1">
          {{ ingestResult }}
        </p>
        <p v-if="ingestError" class="text-[11px] text-red-400 font-mono mt-1">
          {{ ingestError }}
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

        <!-- Init button — shown when not initialized -->
        <button
          v-if="!status?.initialized && !loading"
          @click="handleInit"
          :disabled="ingesting"
          class="px-2 py-1 bg-amber-500/20 hover:bg-amber-500/40 text-amber-300 text-xs rounded border border-amber-500/50 transition-all flex items-center gap-1"
        >
          <i v-if="ingesting" class="pi pi-spin pi-spinner text-[10px]"></i>
          <span>{{ ingesting ? 'Initializing…' : 'Init Brain' }}</span>
        </button>

        <!-- Multi-file ingest -->
        <input
          ref="fileInputRef"
          type="file"
          multiple
          accept=".md,.txt,.go,.ts,.vue,.py,.rs,.json,.yaml,.yml"
          class="hidden"
          @change="onFilesSelected"
        />
        <button
          v-if="isIngestible"
          @click="fileInputRef?.click()"
          :disabled="ingesting"
          class="px-2 py-1 bg-violet-500/20 hover:bg-violet-500/40 text-violet-300 text-xs rounded border border-violet-500/50 transition-all flex items-center justify-center gap-1 min-w-[70px]"
        >
          <i v-if="ingesting" class="pi pi-spin pi-spinner text-[10px]"></i>
          <span>{{ ingesting ? ingestProgress || 'Syncing…' : 'Sync Brain' }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import type { DiscoveredPlanFile, BrainStatus } from '../types/domain';
import { getBrainStatus, ingestKnowledge, initProject } from '../types/wails';

const props = defineProps<{
  projectPath: string;
  nexusFile?: DiscoveredPlanFile;
}>();

const status = ref<BrainStatus | null>(null);
const loading = ref(true);
const ingesting = ref(false);
const ingestProgress = ref('');
const ingestResult = ref('');
const ingestError = ref('');
const fileInputRef = ref<HTMLInputElement | null>(null);

const isIngestible = computed(() => !!props.nexusFile);

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

async function handleInit() {
  ingesting.value = true;
  ingestError.value = '';
  ingestResult.value = '';
  try {
    const claudeMDPath = props.nexusFile?.path ?? '';
    status.value = await initProject(props.projectPath, claudeMDPath);
    ingestResult.value = 'Brain initialized.';
  } catch (err) {
    ingestError.value = err instanceof Error ? err.message : String(err);
  } finally {
    ingesting.value = false;
  }
}

async function onFilesSelected(e: Event) {
  const files = (e.target as HTMLInputElement).files;
  if (!files || files.length === 0) return;
  ingesting.value = true;
  ingestProgress.value = '';
  ingestResult.value = '';
  ingestError.value = '';
  let totalSections = 0;
  const errors: string[] = [];
  for (let i = 0; i < files.length; i++) {
    ingestProgress.value = `${i + 1}/${files.length}`;
    try {
      const count = await ingestKnowledge(props.projectPath, files[i].name);
      totalSections += count;
    } catch (err) {
      errors.push(files[i].name);
    }
  }
  ingesting.value = false;
  ingestProgress.value = '';
  ingestResult.value = `Ingested ${totalSections} sections from ${files.length - errors.length} file${files.length - errors.length !== 1 ? 's' : ''}.`;
  if (errors.length > 0) {
    ingestError.value = `Failed: ${errors.join(', ')}`;
  }
  if (fileInputRef.value) fileInputRef.value.value = '';
  await fetchStatus();
}

onMounted(() => {
  fetchStatus();
});

watch(() => props.projectPath, fetchStatus);
</script>
