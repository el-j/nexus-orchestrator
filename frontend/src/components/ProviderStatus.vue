<template>
  <div class="flex flex-col gap-1">
    <!-- Live provider status -->
    <div class="flex items-center justify-between gap-3">
      <span class="text-xs text-slate-500 font-medium">Providers:</span>
      <button
        class="text-[10px] text-slate-500 hover:text-slate-300 transition-colors px-1.5 py-0.5 rounded hover:bg-white/5 leading-none"
        @click="refresh?.()"
      >⟳ Refresh</button>
    </div>
    <div class="flex items-start gap-2 flex-wrap">
      <div v-for="p in providers" :key="p.Name"
           class="flex items-start gap-1.5 px-2.5 py-1.5 rounded-md border text-xs transition-all"
           :class="p.Active
             ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-400'
             : 'border-red-500/20 bg-red-500/5 text-slate-500'">
        <span :class="['w-1.5 h-1.5 rounded-full mt-[3px] flex-shrink-0', p.Active ? 'bg-emerald-400 animate-pulse' : 'bg-red-500']"></span>
        <div class="flex flex-col min-w-0">
          <div class="flex items-center gap-1">
            <span>{{ p.Name }}</span>
            <span v-if="p.Active && p.ActiveModel" class="text-slate-500 font-mono text-[10px] max-w-[80px] truncate">
              {{ p.ActiveModel }}
            </span>
          </div>
          <span v-if="p.baseURL" class="text-slate-600 text-[10px] font-mono mt-0.5">{{ p.baseURL }}</span>
          <span v-if="!p.Active && p.error" class="text-amber-500/80 text-[10px] mt-0.5">
            {{ p.error.length > 80 ? p.error.slice(0, 80) + '\u2026' : p.error }}
          </span>
        </div>
      </div>
      <div v-if="providers.length === 0" class="text-xs text-slate-600">
        No providers detected &#8212; start LM Studio or Ollama
      </div>
    </div>

    <!-- Configured providers management -->
    <div class="mt-2 pt-2 border-t border-white/5">
      <div class="flex items-center justify-between mb-1.5">
        <span class="text-xs text-slate-500 font-medium">Configured Providers</span>
        <button
          class="text-[10px] text-purple-400 hover:text-purple-300 transition-colors px-1.5 py-0.5 rounded hover:bg-purple-500/10 leading-none"
          @click="openAddForm"
        >＋ Add Provider</button>
      </div>

      <div v-if="configs.length === 0" class="text-xs text-slate-600">
        No configured providers
      </div>

      <div
        v-for="cfg in configs"
        :key="cfg.id"
        class="flex items-center justify-between gap-2 px-2 py-1 rounded hover:bg-white/5 group"
      >
        <div class="flex items-center gap-2 min-w-0">
          <span
            :class="['w-1.5 h-1.5 rounded-full flex-shrink-0', cfg.enabled ? 'bg-emerald-400' : 'bg-slate-600']"
          ></span>
          <span class="text-xs text-slate-300 truncate">{{ cfg.name }}</span>
          <span class="text-[10px] text-slate-600 font-mono truncate max-w-[120px]">{{ cfg.baseURL }}</span>
        </div>
        <div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0">
          <button
            class="text-slate-400 hover:text-purple-400 transition-colors p-0.5 rounded"
            title="Edit"
            @click="openEditForm(cfg)"
          >✎</button>
          <button
            class="text-slate-400 hover:text-red-400 transition-colors p-0.5 rounded"
            title="Delete"
            @click="handleDelete(cfg)"
          >🗑</button>
        </div>
      </div>
    </div>

    <!-- Config form modal -->
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
import type { ProviderInfo, ProviderConfig } from '../types/domain'
import { listProviderConfigs, addProviderConfig, updateProviderConfig, removeProviderConfig } from '../types/wails'
import ProviderConfigForm from './ProviderConfigForm.vue'

const props = defineProps<{ providers: ProviderInfo[], refresh?: () => void }>()

const configs = ref<ProviderConfig[]>([])
const showForm = ref(false)
const editingConfig = ref<ProviderConfig | null>(null)

async function loadConfigs() {
  try {
    configs.value = await listProviderConfigs()
  } catch { /* silent fail */ }
}

onMounted(loadConfigs)

function openAddForm() {
  editingConfig.value = null
  showForm.value = true
}

function openEditForm(cfg: ProviderConfig) {
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
  props.refresh?.()
}

async function handleDelete(cfg: ProviderConfig) {
  if (!window.confirm(`Delete provider "${cfg.name}"?`)) return
  await removeProviderConfig(cfg.id)
  await loadConfigs()
  props.refresh?.()
}
</script>

