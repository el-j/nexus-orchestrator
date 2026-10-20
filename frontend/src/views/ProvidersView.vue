<!--
  Providers: registered backends (status, models, search and filter), discovered backends and
  the saved provider configurations (add, edit, remove, promote a discovered one).
-->
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
            v-for="opt in [
              { v: '', l: 'All' },
              { v: 'active', l: 'Active' },
              { v: 'unreachable', l: 'Unreachable' },
            ]"
            :key="'pf-' + opt.v"
            @click="providerStatusFilter = opt.v as any"
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
              <div class="flex items-center gap-2 flex-wrap">
                <span
                  :class="[
                    'w-2 h-2 rounded-full flex-shrink-0 mt-0.5',
                    p.active ? 'bg-emerald-400 animate-pulse' : 'bg-red-500',
                  ]"
                ></span>
                <span class="font-semibold text-sm text-white">{{ p.name }}</span>
                <ProviderLimitBadges :provider="p" />
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

      <!-- Routing Preview -->
      <RoutingPreviewCard />

      <!-- Project policy: ladder, budgets, token caps -->
      <PolicyCard />

      <DiscoveredToolsSection :sort-mode="sortMode" />

      <!-- Configuration (merged from Settings) -->
      <section class="mt-8">
        <button
          @click="configExpanded = !configExpanded"
          class="flex items-center gap-2 mb-3 group"
        >
          <UiIcon
            :name="configExpanded ? 'chevron-down' : 'chevron-right'"
            class="text-xs text-slate-500"
          />
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
        confirmDialog.onConfirm!();
        confirmDialog.open = false;
      "
      @cancel="confirmDialog.open = false"
    />
  </div>
</template>

<script setup lang="ts">
import UiIcon from '../ui/UiIcon.vue';
import { ref, computed, onMounted, reactive } from 'vue';
import { useRoute } from 'vue-router';
import { useProviders } from '../composables/useProviders';
import { useDiscovery } from '../composables/useDiscovery';
import ProviderLimitBadges from '../components/ProviderLimitBadges.vue';
import RoutingPreviewCard from '../components/RoutingPreviewCard.vue';
import PolicyCard from '../components/PolicyCard.vue';
import DiscoveredProvidersPanel from '../components/DiscoveredProvidersPanel.vue';
import ProviderConfigForm from '../components/ProviderConfigForm.vue';
import DiscoveredToolsSection from '../components/DiscoveredToolsSection.vue';
import AppConfirmDialog from '../components/AppConfirmDialog.vue';
import RefreshIndicator from '../components/RefreshIndicator.vue';
import type { DiscoveredProvider } from '../types/discovery';
import type { ProviderConfig } from '../types/domain';
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

const providerSearch = ref('');
const providerStatusFilter = ref<'' | 'active' | 'unreachable'>('');
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

onMounted(() => {
  lastRefreshed.value = new Date();
});

// ── Provider Configuration CRUD ──────────────────────────────────────────────

const configs = ref<ProviderConfig[]>([]);
const showForm = ref(false);
const editingConfig = ref<ProviderConfig | null>(null);
const configExpanded = ref(false);

const confirmDialog = reactive({
  open: false,
  title: '',
  message: '',
  onConfirm: null as (() => void) | null, // set by showConfirm before the dialog opens
});
function showConfirm(title: string, message: string, onConfirm: () => void) {
  confirmDialog.title = title;
  confirmDialog.message = message;
  confirmDialog.onConfirm = onConfirm;
  confirmDialog.open = true;
}

const route = useRoute();

async function loadConfigs() {
  try {
    configs.value = (await listProviderConfigs()) ?? [];
  } catch {
    /* silent fail */
  }
}
onMounted(async () => {
  await loadConfigs();
  if (route?.query?.action === 'add' || route?.query?.name) {
    editingConfig.value = {
      id: '',
      name: (route.query.name as string) || '',
      kind: (route.query.kind as string as ProviderConfig['kind']) || 'openaicompat',
      baseUrl: (route.query.baseURL as string) || (route.query.baseUrl as string) || '',
      apiKey: '',
      model: (route.query.model as string) || '',
      enabled: true,
      createdAt: '',
      updatedAt: '',
    };
    configExpanded.value = true;
    showForm.value = true;
  }
});

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
  // A pre-filled form for a not-yet-saved provider (promote / deep link) has id ''
  // and must be created, not updated.
  if (editingConfig.value?.id) {
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

function defaultCliCommand(cliPath: string): string {
  const base = cliPath
    .replace(/^.*[\\/]/, '')
    .toLowerCase()
    .replace(/\.exe$/, '');
  if (base === 'claude') return `${cliPath} -p --output-format stream-json`;
  if (base === 'copilot') return `${cliPath} -s --no-ask-user`;
  return `${cliPath} -p`;
}

function handlePromote(provider: DiscoveredProvider) {
  // Pre-fill the configured provider form with discovered data
  const baseUrl =
    provider.baseUrl ||
    (provider.kind === 'cli' && provider.cliPath ? defaultCliCommand(provider.cliPath) : '');
  editingConfig.value = {
    id: '',
    name: provider.name,
    kind: (provider.kind as ProviderConfig['kind']) || 'openaicompat',
    baseUrl,
    apiKey: '',
    model: '',
    enabled: true,
    createdAt: '',
    updatedAt: '',
  };
  showForm.value = true;
}
</script>
