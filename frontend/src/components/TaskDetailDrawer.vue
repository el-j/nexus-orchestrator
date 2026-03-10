<template>
  <Drawer v-model:visible="visible" position="right" class="!bg-[#0d0d14] !w-full sm:!w-[480px]">
    <template #header>
      <div class="flex items-center gap-3">
        <TaskStatusBadge v-if="task" :status="task.Status" />
        <span class="text-sm font-semibold text-white">Task Details</span>
      </div>
    </template>

    <div v-if="task" class="space-y-5 text-sm px-1">
      <!-- ID + timing -->
      <div class="rounded-lg bg-[#0a0a10] border border-white/5 p-3 space-y-2">
        <div class="flex justify-between items-center">
          <span class="text-slate-500 text-xs">ID</span>
          <span class="font-mono text-xs text-slate-400 select-all">{{ task.ID }}</span>
        </div>
        <div class="flex justify-between items-center">
          <span class="text-slate-500 text-xs">Command</span>
          <span class="font-mono text-xs text-violet-400 uppercase">{{ task.Command || 'auto' }}</span>
        </div>
        <div class="flex justify-between items-center">
          <span class="text-slate-500 text-xs">Created</span>
          <span class="text-xs text-slate-400">{{ formatDate(task.CreatedAt) }}</span>
        </div>
        <div class="flex justify-between items-center">
          <span class="text-slate-500 text-xs">Updated</span>
          <span class="text-xs text-slate-400">{{ formatDate(task.UpdatedAt) }}</span>
        </div>
      </div>

      <!-- Instruction -->
      <div>
        <p class="text-xs text-slate-500 mb-2 uppercase tracking-wider font-semibold">Instruction</p>
        <p class="text-white leading-relaxed bg-[#0a0a10] border border-white/5 rounded-lg p-3">
          {{ task.Instruction }}
        </p>
      </div>

      <!-- Project / Target -->
      <div class="grid grid-cols-2 gap-3">
        <div>
          <p class="text-xs text-slate-500 mb-1 uppercase tracking-wider font-semibold">Project</p>
          <p class="font-mono text-xs text-slate-400 break-all">{{ task.ProjectPath || '—' }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-500 mb-1 uppercase tracking-wider font-semibold">Target File</p>
          <p class="font-mono text-xs text-slate-400">{{ task.TargetFile || '—' }}</p>
        </div>
      </div>

      <!-- Provider / Model -->
      <div class="grid grid-cols-2 gap-3">
        <div>
          <p class="text-xs text-slate-500 mb-1 uppercase tracking-wider font-semibold">Provider</p>
          <p class="text-slate-400 text-xs">{{ task.ProviderHint || 'Auto' }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-500 mb-1 uppercase tracking-wider font-semibold">Model</p>
          <p class="text-slate-400 font-mono text-xs">{{ task.ModelID || 'Auto' }}</p>
        </div>
      </div>

      <!-- Output / Logs -->
      <div v-if="task.Logs">
        <p class="text-xs text-slate-500 mb-2 uppercase tracking-wider font-semibold">Output</p>
        <pre class="text-xs text-emerald-300/80 bg-[#0a0a10] border border-white/5 rounded-lg p-3
                    overflow-auto max-h-64 whitespace-pre-wrap font-mono leading-relaxed">{{ task.Logs }}</pre>
      </div>

      <!-- Cancel action -->
      <div v-if="task.Status === 'QUEUED'" class="pt-2">
        <Button
          label="Cancel Task"
          icon="pi pi-times"
          severity="danger"
          outlined
          size="small"
          :loading="cancelling"
          @click="handleCancel"
        />
      </div>
    </div>
  </Drawer>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import Drawer from 'primevue/drawer'
import Button from 'primevue/button'
import { useToast } from 'primevue/usetoast'
import { cancelTask } from '../types/wails'
import type { Task } from '../types/domain'
import TaskStatusBadge from './TaskStatusBadge.vue'

const props = defineProps<{ task: Task | null; modelValue: boolean }>()
const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  cancelled: [id: string]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const toast = useToast()
const cancelling = ref(false)

async function handleCancel() {
  if (!props.task) return
  cancelling.value = true
  try {
    await cancelTask(props.task.ID)
    toast.add({ severity: 'success', summary: 'Task cancelled', life: 2000 })
    emit('cancelled', props.task.ID)
    visible.value = false
  } catch (e) {
    toast.add({
      severity: 'error',
      summary: 'Cancel failed',
      detail: e instanceof Error ? e.message : String(e),
      life: 4000,
    })
  } finally {
    cancelling.value = false
  }
}

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleString()
  } catch {
    return iso ?? ''
  }
}
</script>
