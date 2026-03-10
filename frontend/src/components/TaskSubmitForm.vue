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

        <div class="grid grid-cols-2 sm:grid-cols-3 gap-2">
          <Select
            v-model="form.Command"
            :options="commandOptions"
            option-label="label"
            option-value="value"
            placeholder="Command"
            class="text-xs"
          />
          <InputText
            v-model="form.ProviderHint"
            placeholder="Provider hint"
            class="text-xs"
          />
          <InputText
            v-model="form.ModelID"
            placeholder="Model ID"
            class="text-xs"
          />
        </div>

        <div class="flex justify-end gap-2">
          <Button
            type="button"
            label="Clear"
            severity="secondary"
            text
            size="small"
            @click="resetForm"
          />
          <Button
            type="submit"
            :label="submitting ? 'Submitting…' : 'Submit Task'"
            :loading="submitting"
            size="small"
            icon="pi pi-send"
          />
        </div>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import Select from 'primevue/select'
import { useToast } from 'primevue/usetoast'
import { submitTask } from '../types/wails'
import type { TaskInput } from '../types/domain'

const emit = defineEmits<{ submitted: [id: string] }>()

const toast = useToast()
const expanded = ref(true)
const submitting = ref(false)
const submitted = ref(false)

const commandOptions = [
  { label: 'Auto', value: 'auto' },
  { label: 'Plan', value: 'plan' },
  { label: 'Execute', value: 'execute' },
]

function defaultForm(): TaskInput {
  return {
    ProjectPath: '',
    TargetFile: '',
    Instruction: '',
    Command: 'auto',
    ProviderHint: '',
    ModelID: '',
  }
}

const form = reactive<TaskInput>(defaultForm())

function resetForm() {
  submitted.value = false
  Object.assign(form, defaultForm())
}

async function handleSubmit() {
  submitted.value = true
  if (!form.Instruction?.trim()) return

  submitting.value = true
  try {
    const id = await submitTask({ ...form, ContextFiles: [] })
    toast.add({
      severity: 'success',
      summary: 'Task submitted',
      detail: `ID: ${id}`,
      life: 3000,
    })
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
</script>
