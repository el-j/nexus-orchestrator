<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <header
      class="flex items-center justify-between px-5 py-3 border-b border-white/5 bg-[#0a0a10] flex-shrink-0"
    >
      <div>
        <h1 class="text-sm font-bold text-white">Providers</h1>
        <p class="text-xs text-slate-500">
          <span
            class="font-semibold"
            :class="activeCount > 0 ? 'text-emerald-400' : 'text-slate-500'"
            >{{ activeCount }}</span
          >
          active of {{ providers.length }} registered
          <span class="text-slate-700 mx-1">·</span>
          <span class="text-cyan-400 font-semibold">{{ discovered.length }}</span> discovered
        </p>
      </div>
      <select
        v-model="sortMode"
        class="text-xs bg-white/5 border border-white/10 rounded-lg px-2 py-1.5 text-slate-300 focus:outline-none focus:border-violet-500/40"
      >
        <option value="active">Active first</option>
        <option value="name">Name A-Z</option>
      </select>
      <button
        class="text-xs text-slate-400 hover:text-white px-3 py-1.5 rounded-lg border border-white/10 hover:border-violet-500/40 transition-all"
        @click="
          refresh();
          discoveryRefresh();
        "
      >
        ⟳ Refresh
      </button>
      <RefreshIndicator :last-refreshed="lastRefreshed" />
    </header>

    <!-- Content -->
    <div class="flex-1 overflow-auto p-5 space-y-6">
      <!-- Active providers grid -->
      <section>
        <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">
          Active Providers
        </h2>
        <div class="flex items-center gap-2 mb-3">
          <input
            v-model="providerSearch"
            type="text"
            placeholder="Search providers…"
            class="text-xs bg-white/5 border border-white/10 rounded-lg px-2.5 py-1 text-slate-300 placeholder-slate-600 focus:outline-none focus:border-violet-500/40 w-40"
          />
          <button
            v-for="opt in providerFilterOptions"
            :key="'pf-' + opt.v"
            @click="providerStatusFilter = opt.v"
            :class="[
              'text-[10px] px-2 py-1 rounded-full border transition-colors font-medium',
              providerStatusFilter === opt.v
                ? 'bg-violet-600/20 text-violet-300 border-violet-500/30'
                : 'border-white/10 text-slate-500 hover:text-slate-300',
            ]"
          >
            {{ opt.l }}
          </button>
        </div>
        <div v-if="providers.length === 0" class="text-sm text-slate-600 py-8 text-center">
          No providers detected — start LM Studio or Ollama
        </div>
        <div
          v-else-if="filteredProviders.length === 0"
          class="text-sm text-slate-600 py-8 text-center"
        >
          No matching providers
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div
            v-for="p in filteredProviders"
            :key="p.name"
            class="rounded-xl border p-3 transition-all"
            :class="
              p.active ? 'border-emerald-500/30 bg-emerald-500/5' : 'border-red-500/20 bg-red-500/5'
            "
          >
            <div class="flex items-start justify-between gap-2 mb-2">
              <div class="flex items-center gap-2">
                <span
                  :class="[
                    'w-2 h-2 rounded-full flex-shrink-0 mt-0.5',
                    p.active ? 'bg-emerald-400 animate-pulse' : 'bg-red-500',
                  ]"
                ></span>
                <span class="font-semibold text-sm text-white">{{ p.name }}</span>
              </div>
              <span
                class="text-[10px] px-1.5 py-0.5 rounded font-medium"
                :class="
                  p.active ? 'bg-emerald-500/20 text-emerald-300' : 'bg-red-500/20 text-red-400'
                "
              >
                {{ p.active ? 'Active' : 'Unreachable' }}
              </span>
            </div>
            <div class="text-[11px] text-slate-500 font-mono mb-2">{{ p.baseURL }}</div>
            <div v-if="p.active && p.activeModel" class="text-xs text-slate-400 mb-1">
              Active model: <span class="text-violet-300 font-mono">{{ p.activeModel }}</span>
            </div>
            <div v-if="p.active && p.models?.length" class="flex flex-wrap gap-1 mt-2">
              <span
                v-for="m in p.models"
                :key="m"
                class="text-[10px] px-1.5 py-0.5 rounded font-mono"
                :class="
                  m === p.activeModel
                    ? 'border border-violet-500/30 bg-violet-500/10 text-violet-300'
                    : 'bg-white/5 text-slate-400'
                "
                >{{ m }}</span
              >
            </div>
            <div v-if="!p.active && p.error" class="text-[11px] text-amber-500/80 mt-2 font-mono">
              {{ p.error }}
            </div>
          </div>
        </div>
      </section>

      <!-- Discovered providers (auto-detected from system) -->
      <section>
        <DiscoveredProvidersPanel
          :providers="discovered"
          :loading="discoveryLoading"
          :scanning="scanning"
          @scan="scanNow"
          @promote="handlePromote"
        />
      </section>

      <!-- Configured providers (CRUD) -->
      <section>
        <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">
          Configured Providers
        </h2>
        <ProviderStatus :providers="providers" :refresh="refresh" :hide-discovered="true" />
      </section>

      <!-- AI Coding Tools -->
      <section class="mt-8">
        <div class="flex items-center justify-between mb-3">
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider">
            AI Coding Tools
          </h2>
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
            v-for="opt in toolFilterOptions"
            :key="'tf-' + opt.v"
            @click="toolStatusFilter = opt.v"
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
              <span class="text-xs font-semibold text-slate-400">{{ kindLabel(kind) }}</span>
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

      <!-- Configuration (merged from Settings) -->
      <section class="mt-8">
        <button
          @click="configExpanded = !configExpanded"
          class="flex items-center gap-2 mb-3 group"
        >
          <i
            :class="[
              'pi text-xs text-slate-500',
              configExpanded ? 'pi-chevron-down' : 'pi-chevron-right',
            ]"
          ></i>
          <h2
            class="text-xs font-semibold text-slate-500 uppercase tracking-wider group-hover:text-slate-400"
          >
            Configuration
          </h2>
          <span class="text-[10px] text-slate-600">({{ configs.length }} saved)</span>
        </button>
        <div v-show="configExpanded">
          <div class="flex items-center justify-end mb-3">
            <button
              class="text-[10px] text-violet-400 hover:text-violet-300 transition-colors px-2 py-1 rounded-lg border border-violet-500/20 hover:border-violet-500/40 hover:bg-violet-500/5"
              @click="openAdd"
            >
              ＋ Add Provider
            </button>
          </div>

          <div
            v-if="configs.length === 0"
            class="text-sm text-slate-600 py-8 text-center rounded-xl border border-white/5 bg-white/[0.02]"
          >
            No provider configurations saved yet
          </div>

          <div v-else class="space-y-2">
            <div
              v-for="cfg in configs"
              :key="cfg.id"
              class="flex items-center justify-between gap-3 px-4 py-3 rounded-xl border border-white/5 bg-white/[0.02] hover:bg-white/[0.04] transition-colors group"
            >
              <div class="flex items-center gap-3 min-w-0">
                <span
                  :class="[
                    'w-2 h-2 rounded-full flex-shrink-0',
                    cfg.enabled ? 'bg-emerald-400' : 'bg-slate-600',
                  ]"
                ></span>
                <span class="text-sm font-medium text-white truncate">{{ cfg.name }}</span>
                <span
                  class="text-[10px] px-1.5 py-0.5 rounded bg-violet-500/15 text-violet-300 font-medium flex-shrink-0"
                  >{{ cfg.kind }}</span
                >
                <span class="text-xs text-slate-500 font-mono truncate max-w-[200px]">{{
                  cfg.baseUrl
                }}</span>
              </div>
              <div
                class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0"
              >
                <button
                  class="text-slate-400 hover:text-violet-400 transition-colors text-xs px-2 py-1 rounded hover:bg-white/5"
                  @click="openEdit(cfg)"
                >
                  Edit
                </button>
                <button
                  class="text-slate-400 hover:text-red-400 transition-colors text-xs px-2 py-1 rounded hover:bg-white/5"
                  @click="handleDeleteConfig(cfg)"
                >
                  Remove
                </button>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>

    <!-- ProviderConfigForm modal -->
    <ProviderConfigForm
      v-if="showForm"
      :model-value="editingConfig"
      :on-close="closeForm"
      :on-save="handleSave"
    />

    <AppConfirmDialog
      :open="confirmDialog.open"
      :title="confirmDialog.title"
      :message="confirmDialog.message"
      :danger="true"
      @confirm="
        confirmDialog.onConfirm();
        confirmDialog.open = false;
      "
      @cancel="confirmDialog.open = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, reactive } from 'vue';
import { useProviders } from '../composables/useProviders';
import { useDiscovery } from '../composables/useDiscovery';
import ProviderStatus from '../components/ProviderStatus.vue';
import DiscoveredProvidersPanel from '../components/DiscoveredProvidersPanel.vue';
import ProviderConfigForm from '../components/ProviderConfigForm.vue';
import AppConfirmDialog from '../components/AppConfirmDialog.vue';
import RefreshIndicator from '../components/RefreshIndicator.vue';
import type { DiscoveredProvider } from '../types/discovery';
import type { DiscoveredAgent, ProviderConfig } from '../types/domain';
import { resolveServerUrl } from '../composables/useServerUrl';
import {
  listProviderConfigs,
  addProviderConfig,
  updateProviderConfig,
  removeProviderConfig,
} from '../types/wails';

const { providers, refresh: originalRefresh } = useProviders();
const {
  discovered,
  loading: discoveryLoading,
  scanning,
  refresh: discoveryRefresh,
  scanNow,
} = useDiscovery();

const lastRefreshed = ref<Date | null>(null);
async function refresh() {
  await originalRefresh();
  lastRefreshed.value = new Date();
}

const activeCount = computed(() => providers.value.filter((p) => p.active).length);

const sortMode = ref<'active' | 'name'>('active');

const sortedProviders = computed(() => {
  const list = [...providers.value];
  if (sortMode.value === 'active') {
    return list.sort(
      (a, b) => (b.active ? 1 : 0) - (a.active ? 1 : 0) || a.name.localeCompare(b.name),
    );
  }
  return list.sort((a, b) => a.name.localeCompare(b.name));
});

const sortedAgents = computed(() => {
  const list = [...agents.value];
  if (sortMode.value === 'active') {
    return list.sort(
      (a, b) =>
        (b.isRunning ? 1 : 0) - (a.isRunning ? 1 : 0) ||
        (a.name ?? a.kind).localeCompare(b.name ?? b.kind),
    );
  }
  return list.sort((a, b) => (a.name ?? a.kind).localeCompare(b.name ?? b.kind));
});

const providerSearch = ref('');
const providerStatusFilter = ref<'' | 'active' | 'unreachable'>('');
const providerFilterOptions: { v: '' | 'active' | 'unreachable'; l: string }[] = [
  { v: '', l: 'All' },
  { v: 'active', l: 'Active' },
  { v: 'unreachable', l: 'Unreachable' },
];
const toolSearch = ref('');
const toolStatusFilter = ref<'' | 'running' | 'installed' | 'detected'>('');
const toolFilterOptions: { v: '' | 'running' | 'installed' | 'detected'; l: string }[] = [
  { v: '', l: 'All' },
  { v: 'running', l: 'Running' },
  { v: 'installed', l: 'Installed' },
  { v: 'detected', l: 'Detected' },
];

const filteredProviders = computed(() => {
  let list = sortedProviders.value;
  if (providerStatusFilter.value === 'active') list = list.filter((p) => p.active);
  else if (providerStatusFilter.value === 'unreachable') list = list.filter((p) => !p.active);
  if (providerSearch.value) {
    const q = providerSearch.value.toLowerCase();
    list = list.filter(
      (p) => p.name.toLowerCase().includes(q) || (p.baseURL ?? '').toLowerCase().includes(q),
    );
  }
  return list;
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
  const base = await resolveServerUrl();
  const res = await fetch(`${base}/api/ai-sessions/discovered`);
  if (res.ok) agents.value = await res.json();
}
onMounted(() => {
  loadAgents();
  lastRefreshed.value = new Date();
});

function kindLabel(kind: string): string {
  const labels: Record<string, string> = {
    copilot: 'GitHub Copilot',
    continue: 'Continue',
    'claude-cli': 'Claude CLI',
    cline: 'Cline',
    cursor: 'Cursor',
  };
  return labels[kind] ?? kind;
}

function agentLabel(a: DiscoveredAgent): string {
  return a.name ?? kindLabel(a.kind);
}

function agentStatusClass(a: DiscoveredAgent): string {
  if (a.isRunning) return 'bg-emerald-500/20 text-emerald-300';
  if (a.detectionMethod === 'vscode-extension') return 'bg-amber-500/20 text-amber-300';
  return 'bg-slate-500/20 text-slate-400';
}

// ── Provider Configuration CRUD ──────────────────────────────────────────────

const configs = ref<ProviderConfig[]>([]);
const showForm = ref(false);
const editingConfig = ref<ProviderConfig | null>(null);
const configExpanded = ref(false);

const confirmDialog = reactive({
  open: false,
  title: '',
  message: '',
  onConfirm: () => {},
});
function showConfirm(title: string, message: string, onConfirm: () => void) {
  confirmDialog.title = title;
  confirmDialog.message = message;
  confirmDialog.onConfirm = onConfirm;
  confirmDialog.open = true;
}

async function loadConfigs() {
  try {
    configs.value = (await listProviderConfigs()) ?? [];
  } catch {
    /* silent fail */
  }
}
onMounted(loadConfigs);

function openAdd() {
  editingConfig.value = null;
  showForm.value = true;
}

function openEdit(cfg: ProviderConfig) {
  editingConfig.value = cfg;
  showForm.value = true;
}

function closeForm() {
  showForm.value = false;
  editingConfig.value = null;
}

async function handleSave(cfg: Partial<ProviderConfig>) {
  if (editingConfig.value) {
    await updateProviderConfig({ ...editingConfig.value, ...cfg } as ProviderConfig);
  } else {
    await addProviderConfig(cfg);
  }
  closeForm();
  await loadConfigs();
}

function handleDeleteConfig(cfg: ProviderConfig) {
  showConfirm(
    `Remove provider "${cfg.name}"?`,
    'This will permanently delete this provider configuration.',
    async () => {
      await removeProviderConfig(cfg.id);
      await loadConfigs();
    },
  );
}

function handlePromote(provider: DiscoveredProvider) {
  // Pre-fill the configured provider form with discovered data
  editingConfig.value = {
    id: '',
    name: provider.name,
    kind: (provider.kind as ProviderConfig['kind']) || 'openaicompat',
    baseUrl: provider.baseUrl || '',
    apiKey: '',
    model: '',
    enabled: true,
    createdAt: '',
    updatedAt: '',
  };
  showForm.value = true;
}
</script>
