<!--
  Right-hand drawer with every field of a task, its logs and a Cancel/Interrupt action.
  Props: task (null renders only the header), modelValue (open state).
  Emits: update:modelValue(open), cancelled(id) after a successful cancel, close().
-->
<template>
  <UiDrawer v-model="visible" label="Task details">
    <template #header>
      <div class="flex items-center gap-3">
        <TaskStatusBadge v-if="task" :status="task.status" />
        <span class="text-sm font-semibold text-white">Task Details</span>
      </div>
    </template>

    <div v-if="task" class="space-y-5 text-sm px-1">
      <!-- ID + timing -->
      <div class="rounded-lg bg-[#0a0a10] border border-white/5 p-3 space-y-2">
        <div class="flex justify-between items-center">
          <span class="text-slate-500 text-xs">ID</span>
          <span class="font-mono text-xs text-slate-400 select-all">{{ task.id }}</span>
        </div>
        <div class="flex justify-between items-center">
          <span class="text-slate-500 text-xs">Command</span>
          <span class="font-mono text-xs text-violet-400 uppercase">{{
            task.command || 'auto'
          }}</span>
        </div>
        <div class="flex justify-between items-center">
          <span class="text-slate-500 text-xs">Created</span>
          <span class="text-xs text-slate-400">{{ formatDate(task.createdAt) }}</span>
        </div>
        <div class="flex justify-between items-center">
          <span class="text-slate-500 text-xs">Updated</span>
          <span class="text-xs text-slate-400">{{ formatDate(task.updatedAt) }}</span>
        </div>
      </div>

      <!-- Instruction -->
      <div>
        <p class="text-xs text-slate-500 mb-2 uppercase tracking-wider font-semibold">
          Instruction
        </p>
        <p class="text-white leading-relaxed bg-[#0a0a10] border border-white/5 rounded-lg p-3">
          {{ task.instruction }}
        </p>
      </div>

      <!-- Project / Target -->
      <div class="grid grid-cols-2 gap-3">
        <div>
          <p class="text-xs text-slate-500 mb-1 uppercase tracking-wider font-semibold">Project</p>
          <p class="font-mono text-xs text-slate-400 break-all">{{ task.projectPath || '—' }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-500 mb-1 uppercase tracking-wider font-semibold">
            Target File
          </p>
          <p class="font-mono text-xs text-slate-400">{{ task.targetFile || '—' }}</p>
        </div>
      </div>

      <!-- Provider / Model -->
      <div class="grid grid-cols-2 gap-3">
        <div>
          <p class="text-xs text-slate-500 mb-1 uppercase tracking-wider font-semibold">Provider</p>
          <p class="text-slate-400 text-xs">{{ task.providerHint || 'Auto' }}</p>
        </div>
        <div>
          <p class="text-xs text-slate-500 mb-1 uppercase tracking-wider font-semibold">Model</p>
          <p class="text-slate-400 font-mono text-xs">{{ task.modelId || 'Auto' }}</p>
        </div>
      </div>

      <div v-if="task.logs">
        <p class="text-xs text-slate-500 mb-2 uppercase tracking-wider font-semibold">Output</p>
        <pre
          class="text-xs text-emerald-300/80 bg-[#0a0a10] border border-white/5 rounded-lg p-3 overflow-auto max-h-64 whitespace-pre-wrap font-mono leading-relaxed"
          >{{ task.logs }}</pre>
      </div>

      <div v-if="task.status === 'QUEUED'" class="pt-2">
        <UiButton
          label="Cancel Task"
          icon="times"
          severity="danger"
          outlined
          size="small"
          :loading="cancelling"
          @click="handleCancel"
        />
      </div>

      <div v-if="task.status === 'PROCESSING'" class="pt-2 flex flex-col gap-2">
        <UiButton
          label="Interrupt"
          icon="stop"
          size="small"
          title="Force cancel a running task"
          :loading="cancelling"
          class="!bg-amber-500/20 !text-amber-400 !border-amber-500/40 hover:!bg-amber-500/30"
          outlined
          @click="handleCancel"
        />
        <p v-if="cancelError" class="text-xs text-red-400">{{ cancelError }}</p>
      </div>
    </div>
  </UiDrawer>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import UiDrawer from '../ui/UiDrawer.vue';
import UiButton from '../ui/UiButton.vue';
import { useToast } from '../composables/useToast';
import { cancelTask } from '../types/wails';
import type { Task } from '../types/domain';
import TaskStatusBadge from './TaskStatusBadge.vue';
import { formatDate } from '../utils/time';

const props = defineProps<{ task: Task | null; modelValue: boolean }>();
const emit = defineEmits<{
  'update:modelValue': [v: boolean];
  cancelled: [id: string];
  close: [];
}>();

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
});

const toast = useToast();
const cancelling = ref(false);
const cancelError = ref<string | null>(null);

async function handleCancel() {
  // Both buttons only render while a task is shown, so the task is always present.
  const id = props.task!.id;
  cancelling.value = true;
  cancelError.value = null;
  try {
    await cancelTask(id);
    toast.add({ severity: 'success', summary: 'Task cancelled', life: 2000 });
    emit('cancelled', id);
    emit('close');
    visible.value = false;
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    cancelError.value = msg;
    toast.add({
      severity: 'error',
      summary: 'Cancel failed',
      detail: msg,
      life: 4000,
    });
  } finally {
    cancelling.value = false;
  }
}
</script>
