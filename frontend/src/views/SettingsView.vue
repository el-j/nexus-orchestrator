<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <header class="flex items-center justify-between px-5 py-3 border-b border-white/5 bg-[#0a0a10] flex-shrink-0">
      <div>
        <h1 class="text-sm font-bold text-white">Settings</h1>
        <p class="text-xs text-slate-500">Provider connections, queue limits, and server addresses</p>
      </div>
    </header>

    <!-- Content -->
    <div class="flex-1 overflow-auto p-5 space-y-6">

      <!-- 1. Provider Connections -->
      <section>
        <div class="flex items-center justify-between mb-3">
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider">Provider Connections</h2>
          <button
            class="text-[10px] text-violet-400 hover:text-violet-300 transition-colors px-2 py-1 rounded-lg border border-violet-500/20 hover:border-violet-500/40 hover:bg-violet-500/5"
            @click="openAdd"
          >＋ Add Provider</button>
        </div>

        <div v-if="configs.length === 0" class="text-sm text-slate-600 py-8 text-center rounded-xl border border-white/5 bg-white/[0.02]">
          No provider configurations saved yet
        </div>

        <div v-else class="space-y-2">
          <div
            v-for="cfg in configs"
            :key="cfg.id"
            class="flex items-center justify-between gap-3 px-4 py-3 rounded-xl border border-white/5 bg-white/[0.02] hover:bg-white/[0.04] transition-colors group"
          >
            <div class="flex items-center gap-3 min-w-0">
              <span :class="['w-2 h-2 rounded-full flex-shrink-0', cfg.enabled ? 'bg-emerald-400' : 'bg-slate-600']"></span>
              <span class="text-sm font-medium text-white truncate">{{ cfg.name }}</span>
              <span class="text-[10px] px-1.5 py-0.5 rounded bg-violet-500/15 text-violet-300 font-medium flex-shrink-0">{{ cfg.kind }}</span>
              <span class="text-xs text-slate-500 font-mono truncate max-w-[200px]">{{ cfg.baseURL }}</span>
            </div>
            <div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0">
              <button
                class="text-slate-400 hover:text-violet-400 transition-colors text-xs px-2 py-1 rounded hover:bg-white/5"
                @click="openEdit(cfg)"
              >Edit</button>
              <button
                class="text-slate-400 hover:text-red-400 transition-colors text-xs px-2 py-1 rounded hover:bg-white/5"
                @click="handleDelete(cfg)"
              >Remove</button>
            </div>
          </div>
        </div>
      </section>

      <!-- 2. Queue Settings -->
      <section>
        <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Queue Settings</h2>
        <div class="rounded-xl border border-white/5 bg-white/[0.02] p-4">
          <div class="flex items-center gap-4 mb-2">
            <span class="text-sm text-slate-300">Queue Cap</span>
            <span class="text-xl font-bold text-violet-300 tabular-nums">50</span>
          </div>
          <p class="text-xs text-slate-500">
            Maximum tasks held in queue. Change with the
            <code class="font-mono text-slate-400 bg-white/5 px-1 py-0.5 rounded">NEXUS_QUEUE_CAP</code>
            environment variable and daemon restart.
          </p>
        </div>
      </section>

      <!-- 3. Server Addresses -->
      <section>
        <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Server Addresses</h2>
        <div class="space-y-2">
          <div
            v-for="addr in serverAddresses"
            :key="addr.label"
            class="flex items-center justify-between gap-3 px-4 py-3 rounded-xl border border-white/5 bg-white/[0.02]"
          >
            <div class="flex items-center gap-3 min-w-0">
              <span class="text-xs font-medium text-slate-400 flex-shrink-0 w-24">{{ addr.label }}</span>
              <span class="text-xs font-mono text-slate-300 truncate">{{ addr.url }}</span>
            </div>
            <button
              class="text-[10px] flex-shrink-0 px-2.5 py-1 rounded border transition-colors"
              :class="copied === addr.url
                ? 'border-emerald-500/30 text-emerald-400 bg-emerald-500/5'
                : 'border-white/10 text-slate-400 hover:text-violet-400 hover:border-violet-500/30'"
              @click="copyToClipboard(addr.url)"
            >{{ copied === addr.url ? '✓ Copied' : 'Copy' }}</button>
          </div>
        </div>
        <p class="text-xs text-slate-600 mt-2">
          Changes require daemon restart. Configure via
          <code class="font-mono text-slate-500">NEXUS_LISTEN_ADDR</code> /
          <code class="font-mono text-slate-500">NEXUS_MCP_ADDR</code>.
        </p>
      </section>

    </div>

    <!-- ProviderConfigForm modal -->
    <ProviderConfigForm
      v-if="showForm"
      :model-value="editingConfig"
      :on-close="closeForm"
      :on-save="handleSave"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { ProviderConfig } from '../types/domain'
import { listProviderConfigs, addProviderConfig, updateProviderConfig, removeProviderConfig } from '../types/wails'
import ProviderConfigForm from '../components/ProviderConfigForm.vue'

// ── Provider Connections ─────────────────────────────────────────────────────

const configs = ref<ProviderConfig[]>([])
const showForm = ref(false)
const editingConfig = ref<ProviderConfig | null>(null)

async function loadConfigs() {
  try {
    configs.value = await listProviderConfigs()
  } catch { /* silent fail */ }
}

onMounted(loadConfigs)

function openAdd() {
  editingConfig.value = null
  showForm.value = true
}

function openEdit(cfg: ProviderConfig) {
  editingConfig.value = cfg
  showForm.value = true
}

function closeForm() {
  showForm.value = false
  editingConfig.value = null
}

async function handleSave(cfg: Partial<ProviderConfig>) {
  if (editingConfig.value) {
    await updateProviderConfig({ ...editingConfig.value, ...cfg } as ProviderConfig)
  } else {
    await addProviderConfig(cfg)
  }
  closeForm()
  await loadConfigs()
}

async function handleDelete(cfg: ProviderConfig) {
  if (!window.confirm(`Remove provider "${cfg.name}"?`)) return
  await removeProviderConfig(cfg.id)
  await loadConfigs()
}

// ── Server Addresses ──────────────────────────────────────────────────────────

const serverAddresses = [
  { label: 'HTTP API', url: 'http://127.0.0.1:9999' },
  { label: 'MCP Server', url: 'http://127.0.0.1:9998/mcp' },
]

const copied = ref<string | null>(null)
let copyTimer: ReturnType<typeof setTimeout> | null = null

async function copyToClipboard(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    copied.value = text
    if (copyTimer) clearTimeout(copyTimer)
    copyTimer = setTimeout(() => { copied.value = null }, 2000)
  } catch { /* clipboard unavailable */ }
}
</script>
