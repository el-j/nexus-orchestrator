---
id: TASK-149
title: Fix ProviderConfigForm.vue to use kind field
role: api
planId: PLAN-021
status: todo
dependencies: [TASK-146]
createdAt: 2026-03-12T10:00:00.000Z
---

## Context
`ProviderConfigForm.vue` uses `form.value.type` throughout (matching the old incorrect TypeScript
interface). After TASK-146 renames `ProviderConfig.type` → `ProviderConfig.kind`, the form must
be updated to use `form.value.kind` so that:
- The `v-model="form.type"` binding on the `<select>` element sends the correct JSON key `kind`
- `buildDefaultForm()` initialises `kind` not `type`
- `onTypeChange()` reads from `form.value.kind`
- `BASE_URL_PRESETS` lookup uses `form.value.kind`

## Current file content of ProviderConfigForm.vue
```vue
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
            v-model="form.type"
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
    type: 'lmstudio',
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
  const preset = BASE_URL_PRESETS[form.value.type ?? 'openaicompat']
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
```

## Implementation Steps
1. In the `<template>`: change `v-model="form.type"` → `v-model="form.kind"` on the `<select>`.
2. In `buildDefaultForm()`: change `type: 'lmstudio'` → `kind: 'lmstudio'`.
3. In `onTypeChange()`: change `form.value.type` → `form.value.kind` (both the lookup and the nullish coalescing).
4. Output the **complete updated file** — all other template and script content unchanged.

## Acceptance Criteria
- [ ] `v-model="form.kind"` on the type `<select>`
- [ ] `buildDefaultForm()` returns `{ kind: 'lmstudio', ... }`
- [ ] `onTypeChange()` uses `form.value.kind`
- [ ] No other changes made to the component
- [ ] Output is the complete file (template + script, nothing omitted)

## Anti-patterns to Avoid
- Do NOT rename `<option value="lmstudio">` etc. — those HTML values are correct
- Do NOT change any CSS classes or layout
- Do NOT change the form field bindings for name, baseURL, apiKey, defaultModel, or enabled
- Do NOT add new imports
