<template>
  <div
    class="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
    @click.self="onClose"
  >
    <div class="bg-slate-800 rounded-xl p-6 w-full max-w-md shadow-2xl border border-white/10">
      <h2 class="text-sm font-bold text-white mb-4">
        {{ modelValue ? 'Edit Provider' : 'Add Provider' }}
      </h2>

      <form @submit.prevent="handleSave" class="flex flex-col gap-3">
        <!-- Type -->
        <div class="flex flex-col gap-1">
          <label class="text-xs text-slate-400">Type</label>
          <select
            v-model="form.kind"
            class="bg-slate-700 rounded px-3 py-2 text-sm w-full text-white border border-white/10 focus:outline-none focus:border-purple-500"
            @change="onTypeChange"
          >
            <option value="lmstudio">LM Studio</option>
            <option value="ollama">Ollama</option>
            <option value="openai">OpenAI</option>
            <option value="anthropic">Anthropic</option>
            <option value="openaicompat">Custom (OpenAI-compatible)</option>
          </select>
        </div>

        <!-- Name -->
        <div class="flex flex-col gap-1">
          <label class="text-xs text-slate-400">Name <span class="text-red-400">*</span></label>
          <input
            v-model="form.name"
            type="text"
            placeholder="My Provider"
            required
            class="bg-slate-700 rounded px-3 py-2 text-sm w-full text-white border border-white/10 focus:outline-none focus:border-purple-500 placeholder:text-slate-600"
          />
        </div>

        <!-- Base URL -->
        <div class="flex flex-col gap-1">
          <label class="text-xs text-slate-400">Base URL <span class="text-red-400">*</span></label>
          <input
            v-model="form.baseURL"
            type="text"
            placeholder="http://127.0.0.1:1234/v1"
            required
            class="bg-slate-700 rounded px-3 py-2 text-sm w-full text-white border border-white/10 focus:outline-none focus:border-purple-500 placeholder:text-slate-600 font-mono text-xs"
          />
        </div>

        <!-- API Key -->
        <div class="flex flex-col gap-1">
          <label class="text-xs text-slate-400">API Key</label>
          <input
            v-model="form.apiKey"
            type="password"
            placeholder="sk-… (optional)"
            autocomplete="new-password"
            class="bg-slate-700 rounded px-3 py-2 text-sm w-full text-white border border-white/10 focus:outline-none focus:border-purple-500 placeholder:text-slate-600"
          />
        </div>

        <!-- Default Model -->
        <div class="flex flex-col gap-1">
          <label class="text-xs text-slate-400">Default Model</label>
          <input
            v-model="form.defaultModel"
            type="text"
            placeholder="e.g. gpt-4o"
            class="bg-slate-700 rounded px-3 py-2 text-sm w-full text-white border border-white/10 focus:outline-none focus:border-purple-500 placeholder:text-slate-600 font-mono text-xs"
          />
        </div>

        <!-- Enabled -->
        <div class="flex items-center gap-2 mt-1">
          <input
            id="cfg-enabled"
            v-model="form.enabled"
            type="checkbox"
            class="accent-purple-500 w-4 h-4 cursor-pointer"
          />
          <label for="cfg-enabled" class="text-xs text-slate-400 cursor-pointer select-none">Enabled</label>
        </div>

        <!-- Error message -->
        <p v-if="error" class="text-xs text-red-400 bg-red-500/10 rounded px-2 py-1.5">{{ error }}</p>

        <!-- Buttons -->
        <div class="flex justify-end gap-2 mt-2 pt-2 border-t border-white/5">
          <button
            type="button"
            class="px-4 py-1.5 text-xs rounded bg-slate-700 text-slate-300 hover:bg-slate-600 transition-colors"
            @click="onClose"
          >Cancel</button>
          <button
            type="submit"
            :disabled="saving"
            class="px-4 py-1.5 text-xs rounded bg-purple-600 text-white hover:bg-purple-500 disabled:opacity-50 transition-colors"
          >{{ saving ? 'Saving…' : 'Save' }}</button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import type { ProviderConfig } from '../types/domain'

const props = defineProps<{
  modelValue: ProviderConfig | null
  onClose: () => void
  onSave: (cfg: Partial<ProviderConfig>) => Promise<void>
}>()

const BASE_URL_PRESETS: Record<string, string> = {
  lmstudio: 'http://127.0.0.1:1234/v1',
  ollama: 'http://127.0.0.1:11434',
  openai: 'https://api.openai.com/v1',
  anthropic: 'https://api.anthropic.com',
  openaicompat: '',
}

function buildDefaultForm(): Partial<ProviderConfig> {
  return {
    kind: 'lmstudio',
    name: '',
    baseURL: BASE_URL_PRESETS['lmstudio'],
    apiKey: '',
    defaultModel: '',
    enabled: true,
  }
}

const form = ref<Partial<ProviderConfig>>(
  props.modelValue ? { ...props.modelValue } : buildDefaultForm(),
)
const saving = ref(false)
const error = ref('')

watch(
  () => props.modelValue,
  (v) => {
    form.value = v ? { ...v } : buildDefaultForm()
    error.value = ''
  },
  { immediate: true },
)

function onTypeChange() {
  const preset = BASE_URL_PRESETS[form.value.kind ?? 'openaicompat']
  if (preset !== undefined) {
    form.value.baseURL = preset
  }
}

async function handleSave() {
  error.value = ''
  saving.value = true
  try {
    await props.onSave(form.value)
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    saving.value = false
  }
}

function handleEscape(e: KeyboardEvent) {
  if (e.key === 'Escape') props.onClose()
}

onMounted(() => window.addEventListener('keydown', handleEscape))
onUnmounted(() => window.removeEventListener('keydown', handleEscape))
</script>