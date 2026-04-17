<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <header
      class="flex items-center justify-between px-5 py-3 border-b border-white/5 bg-[#0a0a10] shrink-0"
    >
      <div>
        <h1 class="text-sm font-bold text-white">Project Brain</h1>
        <p class="text-xs text-slate-500">
          <span class="text-white font-semibold">{{ status?.entryCount ?? 0 }}</span> entries ·
          <span class="text-white font-semibold">{{
            status?.totalTokens?.toLocaleString() ?? 0
          }}</span>
          tokens
        </p>
      </div>
      <div class="flex items-center gap-2">
        <input
          v-model="projectPath"
          type="text"
          placeholder="Project path…"
          class="text-xs bg-white/5 border border-white/10 rounded px-2 py-1 text-white placeholder-slate-600 w-56 focus:outline-none focus:border-violet-500/50"
          @keydown.enter="onPathChange"
          @blur="onPathChange"
        />
        <button
          v-if="!status?.initialized"
          @click="handleInit"
          :disabled="loading || !projectPath"
          class="px-3 py-1 text-xs rounded bg-violet-500/20 hover:bg-violet-500/40 text-violet-300 border border-violet-500/40 disabled:opacity-40"
        >
          Initialize
        </button>
        <button
          @click="refresh"
          :disabled="loading"
          class="px-3 py-1 text-xs rounded bg-white/5 hover:bg-white/10 text-slate-300 border border-white/10 disabled:opacity-40"
        >
          Refresh
        </button>
      </div>
    </header>

    <!-- Error banner -->
    <div
      v-if="error"
      class="mx-4 mt-3 px-3 py-2 rounded bg-red-500/10 border border-red-500/30 text-xs text-red-300"
    >
      {{ error }}
    </div>

    <!-- Status bar -->
    <div
      v-if="status?.kindCounts"
      class="flex gap-4 px-5 py-2 border-b border-white/5 shrink-0 overflow-x-auto"
    >
      <div
        v-for="(count, kind) in status.kindCounts"
        :key="kind"
        class="flex flex-col items-center"
      >
        <span class="text-[10px] text-slate-500 uppercase font-bold tracking-wider">{{
          kind
        }}</span>
        <span class="text-xs font-mono text-cyan-300">{{ count }}</span>
      </div>
    </div>

    <!-- Main content: tabs -->
    <div class="flex border-b border-white/5 shrink-0 px-4">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        @click="activeTab = tab.key"
        class="px-3 py-2 text-xs font-medium transition-colors"
        :class="
          activeTab === tab.key
            ? 'text-violet-300 border-b-2 border-violet-400'
            : 'text-slate-500 hover:text-slate-300'
        "
      >
        {{ tab.label }}
      </button>
    </div>

    <div class="flex-1 overflow-y-auto p-4 space-y-3">
      <!-- Search tab -->
      <template v-if="activeTab === 'search'">
        <div class="flex gap-2">
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Search knowledge…"
            class="flex-1 text-xs bg-white/5 border border-white/10 rounded px-3 py-2 text-white placeholder-slate-600 focus:outline-none focus:border-violet-500/50"
            @keydown.enter="handleSearch"
          />
          <button
            @click="handleSearch"
            :disabled="loading || !searchQuery"
            class="px-3 py-2 text-xs rounded bg-violet-500/20 hover:bg-violet-500/40 text-violet-300 border border-violet-500/40 disabled:opacity-40"
          >
            Search
          </button>
        </div>
        <div
          v-if="searchResults.length === 0 && !loading"
          class="text-xs text-slate-600 text-center py-8"
        >
          No results
        </div>
        <div
          v-for="(r, i) in searchResults"
          :key="i"
          class="rounded border border-white/5 bg-white/[0.02] p-3"
        >
          <div class="flex items-center gap-2 mb-1">
            <span class="text-[10px] px-1.5 py-0.5 rounded bg-violet-500/20 text-violet-300">{{
              r.kind
            }}</span>
            <span class="text-xs font-semibold text-white">{{ r.topic }}</span>
            <span class="ml-auto text-[10px] text-slate-600 font-mono">{{ r.tokens }}t</span>
          </div>
          <p class="text-xs text-slate-400 line-clamp-3 font-mono whitespace-pre-wrap">
            {{ r.content }}
          </p>
        </div>
      </template>

      <!-- Entries tab -->
      <template v-if="activeTab === 'entries'">
        <div class="flex gap-2 mb-2">
          <select
            v-model="kindFilter"
            @change="handleListEntries"
            class="text-xs bg-white/5 border border-white/10 rounded px-2 py-1 text-white focus:outline-none"
          >
            <option value="">All kinds</option>
            <option value="architecture">architecture</option>
            <option value="convention">convention</option>
            <option value="file_map">file_map</option>
            <option value="learning">learning</option>
            <option value="glossary">glossary</option>
            <option value="task_summary">task_summary</option>
          </select>
        </div>
        <div
          v-if="entries.length === 0 && !loading"
          class="text-xs text-slate-600 text-center py-8"
        >
          No entries
        </div>
        <div
          v-for="entry in entries"
          :key="entry.id"
          class="rounded border border-white/5 bg-white/[0.02] p-3 flex gap-3"
        >
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2 mb-1">
              <span class="text-[10px] px-1.5 py-0.5 rounded bg-violet-500/20 text-violet-300">{{
                entry.kind
              }}</span>
              <span class="text-xs font-semibold text-white truncate">{{ entry.topic }}</span>
              <span class="ml-auto text-[10px] text-slate-600 font-mono shrink-0"
                >{{ entry.tokenCount }}t</span
              >
            </div>
            <p class="text-xs text-slate-500 line-clamp-2 font-mono">{{ entry.content }}</p>
          </div>
          <button
            @click="handleDelete(entry.id)"
            class="text-slate-600 hover:text-red-400 text-xs shrink-0 self-start"
            title="Delete"
          >
            ✕
          </button>
        </div>
      </template>

      <!-- Ingest tab -->
      <template v-if="activeTab === 'ingest'">
        <div class="space-y-3">
          <p class="text-xs text-slate-500">
            Select one or more files to ingest into the project brain.
          </p>
          <!-- Wails desktop mode: use native file dialog for real paths -->
          <template v-if="wailsMode">
            <button
              @click="handleWailsIngest"
              :disabled="ingesting || !projectPath"
              class="px-4 py-2 text-xs rounded bg-violet-500/20 hover:bg-violet-500/40 text-violet-300 border border-violet-500/40 disabled:opacity-40"
            >
              Select File…
            </button>
          </template>
          <!-- Browser dev mode: ingest requires desktop app -->
          <template v-else>
            <div
              class="px-3 py-2 rounded bg-amber-500/10 border border-amber-500/30 text-xs text-amber-300"
            >
              File ingest requires the desktop app — full file-system paths are not accessible from
              the browser. Launch the Wails desktop build to use this feature.
            </div>
          </template>
          <div v-if="ingestProgress" class="text-xs text-slate-400">
            {{ ingestProgress }}
          </div>
          <div v-if="ingestResult" class="text-xs text-emerald-300">
            {{ ingestResult }}
          </div>
        </div>
      </template>

      <!-- File map tab -->
      <template v-if="activeTab === 'filemap'">
        <button
          @click="handleFetchFileMap"
          :disabled="loading"
          class="px-3 py-1 text-xs rounded bg-white/5 hover:bg-white/10 text-slate-300 border border-white/10 disabled:opacity-40 mb-2"
        >
          Load File Map
        </button>
        <div
          v-if="fileMap.length === 0 && !loading"
          class="text-xs text-slate-600 text-center py-8"
        >
          No file map entries
        </div>
        <div
          v-for="(path, i) in fileMap"
          :key="i"
          class="text-xs font-mono text-slate-400 py-0.5 border-b border-white/5 last:border-0"
        >
          {{ i + 1 }}. {{ path }}
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, shallowRef } from 'vue';
import { useBrain } from '../composables/useBrain';
import { currentProject } from '../composables/useProjectState';

const isWails = (): boolean => !!window.go?.main?.App;

const projectPath = ref('');
const activeTab = ref<'search' | 'entries' | 'ingest' | 'filemap'>('search');
const searchQuery = ref('');
const kindFilter = ref('');
const ingesting = ref(false);
const ingestProgress = ref('');
const ingestResult = ref('');
const wailsMode = ref(isWails());

const tabs = [
  { key: 'search' as const, label: 'Search' },
  { key: 'entries' as const, label: 'Entries' },
  { key: 'ingest' as const, label: 'Ingest' },
  { key: 'filemap' as const, label: 'File Map' },
];

// brain ref is swapped when projectPath changes; computed proxies ensure the
// template always reads from the current instance's reactive state.
const brain = shallowRef(useBrain(''));

// Computed proxies — necessary because destructuring brain.value captures refs
// from the initial instance and won't update when brain.value is reassigned.
const status = computed(() => brain.value.status.value);
const entries = computed(() => brain.value.entries.value);
const fileMap = computed(() => brain.value.fileMap.value);
const searchResults = computed(() => brain.value.searchResults.value);
const loading = computed(() => brain.value.loading.value);
const error = computed(() => brain.value.error.value);

function onPathChange() {
  if (projectPath.value) {
    localStorage.setItem('nexus:projectPath', projectPath.value);
    brain.value = useBrain(projectPath.value);
    void brain.value.fetchStatus();
  }
}

async function refresh() {
  if (!projectPath.value) return;
  await brain.value.fetchStatus();
}

async function handleInit() {
  await brain.value.init();
}

async function handleSearch() {
  if (!searchQuery.value) return;
  await brain.value.search(searchQuery.value);
}

async function handleListEntries() {
  await brain.value.listEntries(kindFilter.value);
}

async function handleDelete(id: string) {
  await brain.value.deleteEntry(id);
}

async function handleFetchFileMap() {
  await brain.value.fetchFileMap();
}

async function handleWailsIngest() {
  if (!projectPath.value || !window.go?.main?.App) return;
  ingesting.value = true;
  ingestProgress.value = 'Opening file dialog…';
  ingestResult.value = '';
  try {
    const filePath = await window.go.main.App.OpenFileDialog('Select file to ingest', [
      {
        displayName: 'Knowledge Files',
        pattern: '*.md;*.txt;*.go;*.ts;*.vue;*.py;*.rs;*.json;*.yaml;*.yml',
      },
    ]);
    if (!filePath) {
      ingestProgress.value = '';
      return;
    }
    ingestProgress.value = `Ingesting: ${filePath}…`;
    const count = await brain.value.ingest(filePath);
    ingestResult.value = `Ingested ${count} section${count !== 1 ? 's' : ''} from ${filePath}.`;
  } catch (e) {
    ingestResult.value = e instanceof Error ? e.message : String(e);
  } finally {
    ingesting.value = false;
    ingestProgress.value = '';
  }
}

onMounted(() => {
  // Seed project path from localStorage, then fall back to the global
  // currentProject selection (set via the sidebar project picker).
  const saved = localStorage.getItem('nexus:projectPath') ?? currentProject.value;
  if (saved) {
    projectPath.value = saved;
    onPathChange();
  }
});
</script>
