<template>
  <div class="border-t border-white/5 bg-[#0a0a10] p-4">
    <form @submit.prevent="handleSubmit" class="flex flex-col gap-3">
      <div class="flex items-center justify-between">
        <span class="text-xs font-semibold text-slate-500 uppercase tracking-wider">Submit Task</span>
        <button
          type="button"
          @click="expanded = !expanded"
          class="p-1 rounded text-slate-500 hover:text-slate-300 transition-colors"
          :aria-label="expanded ? 'Collapse form' : 'Expand form'"
        >
          <i :class="`pi ${expanded ? 'pi-chevron-down' : 'pi-chevron-up'} text-xs`"></i>
        </button>
      </div>

      <div v-show="expanded" class="space-y-3">
        <Textarea
          v-model="form.Instruction"
          placeholder="What should the AI do? e.g. 'Add error handling to the HTTP handler'"
          :rows="2"
          class="w-full text-sm resize-none"
          :invalid="submitted && !form.Instruction.trim()"
        />
        <small v-if="submitted && !form.Instruction.trim()" class="text-red-400 text-xs">
          Instruction is required.
        </small>

        <!-- Project + Target file -->
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
          <InputText
            v-model="form.ProjectPath"
            placeholder="Project path (absolute)"
            class="text-xs"
          />
          <InputText
            v-model="form.TargetFile"
            placeholder="Target file (e.g. handler.go)"
            class="text-xs"
          />
        </div>

        <!-- Command + Priority -->
        <div class="grid grid-cols-2 gap-2">
          <Select
            v-model="form.Command"
            :options="commandOptions"
            option-label="label"
            option-value="value"
            placeholder="Command"
            class="text-xs"
          />
          <Select
            v-model="form.Priority"
            :options="priorityOptions"
            option-label="label"
            option-value="value"
            placeholder="Priority"
            class="text-xs"
          />
        </div>

        <!-- Provider + Model -->
        <div class="grid grid-cols-2 gap-2">
          <Select
            v-model="form.ProviderName"
            :options="providerOptions"
            option-label="label"
            option-value="value"
            placeholder="Auto (best available)"
            class="text-xs"
          />
          <Select
            v-model="form.ModelID"
            :options="modelOptions"
            option-label="label"
            option-value="value"
            placeholder="Default model"
            :disabled="!form.ProviderName"
            class="text-xs"
          />
        </div>

        <!-- Tags -->
        <div>
          <InputText
            v-model="tagInput"
            placeholder="Add tag… (Enter or ,)"
            class="w-full text-xs"
            @keydown.enter.prevent="addTag"
            @keydown="onTagKeydown"
          />
          <div v-if="form.Tags.length > 0" class="flex gap-1 flex-wrap mt-1">
            <span
              v-for="(tag, i) in form.Tags"
              :key="tag"
              class="inline-flex items-center gap-0.5 px-2 py-0.5 rounded-full bg-indigo-900/60 text-indigo-300 text-xs border border-indigo-700/50"
            >
              {{ tag }}
              <button
                type="button"
                @click="removeTag(i)"
                class="ml-0.5 text-indigo-400 hover:text-indigo-100 leading-none"
                :aria-label="`Remove tag ${tag}`"
              >×</button>
            </span>
          </div>
        </div>

        <!-- Actions: Clear + Split submit button -->
        <div class="flex justify-end gap-2">
          <Button
            type="button"
            label="Clear"
            severity="secondary"
            text
            size="small"
            @click="resetForm"
          />
          <!-- Split button: Primary + dropdown toggle -->
          <div class="relative flex" ref="splitRef">
            <Button
              type="submit"
              :label="submitting ? 'Submitting…' : 'Submit to Queue'"
              :loading="submitting"
              size="small"
              icon="pi pi-send"
              class="!rounded-r-none"
            />
            <button
              type="button"
              @click.stop="splitOpen = !splitOpen"
              class="px-2 bg-indigo-600 hover:bg-indigo-500 active:bg-indigo-700 border border-l-0 border-indigo-500 rounded-r-md text-white transition-colors flex items-center"
              :class="{ '!bg-indigo-500': splitOpen }"
              aria-label="More submit options"
              :aria-expanded="splitOpen"
            >
              <i class="pi pi-chevron-down text-[10px]"></i>
            </button>
            <!-- Dropdown -->
            <Transition name="split-menu">
              <div
                v-if="splitOpen"
                role="menu"
                @keydown.esc="splitOpen = false"
                class="absolute right-0 bottom-full mb-1 z-50 min-w-[160px] rounded-md border border-white/10 bg-[#13131f] shadow-xl overflow-hidden"
              >
                <button
                  type="button"
                  role="menuitem"
                  @click="handleSaveDraft"
                  class="w-full text-left px-3 py-2 text-xs text-slate-300 hover:bg-white/5 flex items-center gap-2 transition-colors"
                >
                  <i class="pi pi-file-edit text-slate-400"></i>
                  Save as Draft
                </button>
                <button
                  type="button"
                  role="menuitem"
                  @click="handleSaveBacklog"
                  class="w-full text-left px-3 py-2 text-xs text-slate-300 hover:bg-white/5 flex items-center gap-2 transition-colors"
                >
                  <i class="pi pi-list text-slate-400"></i>
                  Add to Backlog
                </button>
              </div>
            </Transition>
          </div>
        </div>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount } from 'vue'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import Select from 'primevue/select'
import { useToast } from 'primevue/usetoast'
import { submitTask, createDraft } from '../types/wails'
import type { Task } from '../types/domain'
import { useProviders } from '../composables/useProviders'

const emit = defineEmits<{ submitted: [id: string] }>()

const toast = useToast()
const expanded = ref(true)
const submitting = ref(false)
const submitted = ref(false)
const splitOpen = ref(false)
const splitRef = ref<HTMLElement | null>(null)
const tagInput = ref('')

const { providers } = useProviders()

// --- Options ---

const commandOptions = [
  { label: 'Auto', value: 'auto' },
  { label: 'Plan', value: 'plan' },
  { label: 'Execute', value: 'execute' },
]

const priorityOptions = [
  { label: '🔴 High', value: 1 },
  { label: '🟡 Medium', value: 2 },
  { label: '🟢 Low', value: 3 },
]

const providerOptions = computed(() => [
  { label: 'Auto (best available)', value: '' },
  ...providers.value.map(p => ({ label: p.name, value: p.name })),
])

const modelOptions = computed(() => {
  const selected = providers.value.find(p => p.name === form.ProviderName)
  return [
    { label: 'Default model', value: '' },
    ...(selected?.models ?? []).map(m => ({ label: m, value: m })),
  ]
})

// --- Form state ---

interface FormState {
  ProjectPath: string
  TargetFile: string
  Instruction: string
  Command: 'auto' | 'plan' | 'execute'
  ProviderName: string
  ModelID: string
  Priority: number
  Tags: string[]
}

function defaultForm(): FormState {
  return {
    ProjectPath: '',
    TargetFile: '',
    Instruction: '',
    Command: 'auto',
    ProviderName: '',
    ModelID: '',
    Priority: 2,
    Tags: [],
  }
}

const form = reactive<FormState>(defaultForm())

function resetForm() {
  submitted.value = false
  tagInput.value = ''
  splitOpen.value = false
  Object.assign(form, defaultForm())
}

// --- Tag helpers ---

function addTag() {
  const tag = tagInput.value.replace(/,/g, '').trim()
  if (tag && !form.Tags.includes(tag)) {
    form.Tags.push(tag)
  }
  tagInput.value = ''
}

function onTagKeydown(e: KeyboardEvent) {
  if (e.key === ',') {
    e.preventDefault()
    addTag()
  }
}

function removeTag(index: number) {
  form.Tags.splice(index, 1)
}

// --- Payload builder ---

function buildTaskPayload(): Partial<Task> {
  return {
    ProjectPath: form.ProjectPath,
    TargetFile: form.TargetFile,
    Instruction: form.Instruction.trim(),
    Command: form.Command,
    ProviderHint: form.ProviderName || '',
    ModelID: form.ModelID || '',
    Priority: form.Priority,
    Tags: form.Tags.length > 0 ? [...form.Tags] : [],
    ContextFiles: [],
  }
}

// --- Submit handlers ---

async function handleSubmit() {
  submitted.value = true
  if (!form.Instruction?.trim()) return

  submitting.value = true
  try {
    const p = buildTaskPayload()
    const id = await submitTask({
      ProjectPath: p.ProjectPath ?? '',
      TargetFile: p.TargetFile ?? '',
      Instruction: p.Instruction ?? '',
      Command: p.Command,
      ProviderHint: p.ProviderHint ?? '',
      ModelID: p.ModelID ?? '',
      ContextFiles: [],
    })
    toast.add({ severity: 'success', summary: 'Task submitted', detail: `ID: ${id}`, life: 3000 })
    emit('submitted', id)
    resetForm()
  } catch (e) {
    toast.add({
      severity: 'error',
      summary: 'Submit failed',
      detail: e instanceof Error ? e.message : String(e),
      life: 5000,
    })
  } finally {
    submitting.value = false
  }
}

async function handleSaveDraft() {
  splitOpen.value = false
  submitted.value = true
  if (!form.Instruction?.trim()) return

  try {
    const id = await createDraft({ ...buildTaskPayload(), Status: 'DRAFT' })
    toast.add({ severity: 'info', summary: 'Draft saved', detail: `ID: ${id}`, life: 3000 })
    emit('submitted', id)
    resetForm()
  } catch (e) {
    toast.add({
      severity: 'error',
      summary: 'Save failed',
      detail: e instanceof Error ? e.message : String(e),
      life: 5000,
    })
  }
}

async function handleSaveBacklog() {
  splitOpen.value = false
  submitted.value = true
  if (!form.Instruction?.trim()) return

  try {
    const id = await createDraft({ ...buildTaskPayload(), Status: 'BACKLOG' })
    toast.add({ severity: 'info', summary: 'Added to backlog', detail: `ID: ${id}`, life: 3000 })
    emit('submitted', id)
    resetForm()
  } catch (e) {
    toast.add({
      severity: 'error',
      summary: 'Save failed',
      detail: e instanceof Error ? e.message : String(e),
      life: 5000,
    })
  }
}

// --- Close split dropdown on outside click ---

function onDocumentClick(e: MouseEvent) {
  if (splitRef.value && !splitRef.value.contains(e.target as Node)) {
    splitOpen.value = false
  }
}

onMounted(() => document.addEventListener('click', onDocumentClick))
onBeforeUnmount(() => document.removeEventListener('click', onDocumentClick))
</script>

<style scoped>
.split-menu-enter-active,
.split-menu-leave-active {
  transition: opacity 0.1s ease, transform 0.1s ease;
}
.split-menu-enter-from,
.split-menu-leave-to {
  opacity: 0;
  transform: translateY(4px);
}
</style>
