<template>
  <!-- Empty state -->
  <div
    v-if="items.length === 0"
    class="flex flex-col items-center justify-center py-20 text-center px-6"
  >
    <div
      class="w-16 h-16 rounded-2xl bg-violet-600/10 border border-violet-500/20 flex items-center justify-center mb-4"
    >
      <i class="pi pi-bookmark text-2xl text-violet-400"></i>
    </div>
    <h3 class="font-semibold text-white mb-2">Backlog is empty</h3>
    <p class="text-sm text-slate-500 max-w-xs">
      Draft ideas here before queuing them for execution.
    </p>
  </div>

  <!-- Task cards -->
  <div v-else class="space-y-2">
    <div
      v-for="task in items"
      :key="task.id"
      class="group flex items-start gap-4 p-4 rounded-xl border border-white/5 bg-nexus-800 hover:border-violet-500/20 hover:bg-nexus-700 transition-all"
    >
      <!-- Priority badge -->
      <span :class="['text-xs font-semibold mt-0.5 w-14 shrink-0', priorityColor(task.priority)]">
        {{ priorityLabel(task.priority) }}
      </span>

      <!-- Content -->
      <div class="flex-1 min-w-0">
        <p class="text-sm font-medium text-white leading-snug">{{ truncate(task.instruction) }}</p>
        <p class="text-xs text-slate-600 font-mono mt-0.5 truncate">{{ task.projectPath }}</p>
        <div class="flex flex-wrap gap-1 mt-1.5">
          <span
            v-if="task.providerName"
            class="text-xs bg-violet-500/15 text-violet-400 border border-violet-500/20 px-1.5 py-0.5 rounded-full"
          >
            {{ task.providerName }}
          </span>
          <span
            v-else
            class="text-xs bg-white/5 text-slate-500 border border-white/10 px-1.5 py-0.5 rounded-full"
          >
            Auto
          </span>
          <span
            v-for="tag in task.tags ?? []"
            :key="tag"
            class="text-xs bg-white/5 text-slate-400 border border-white/10 px-1.5 py-0.5 rounded-full"
            >{{ tag }}</span
          >
        </div>
      </div>

      <!-- Status + Promote -->
      <div class="flex flex-col items-end gap-1.5 shrink-0">
        <span class="text-xs text-slate-600 uppercase tracking-wider">{{ task.status }}</span>
        <div class="opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-1">
          <Button
            @click="onPromote(task.id)"
            label="Promote ↑"
            size="small"
            severity="secondary"
            class="text-xs"
            :loading="promoting === task.id"
            :disabled="dismissing === task.id"
          />
          <Button
            @click="onDismiss(task.id)"
            label="Dismiss"
            size="small"
            severity="danger"
            outlined
            class="text-xs"
            :loading="dismissing === task.id"
            :disabled="promoting === task.id"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import Button from 'primevue/button';
import { useToast } from 'primevue/usetoast';
import { useTasks } from '../composables/useTasks';
import type { Task } from '../types/domain';

const emit = defineEmits<{
  promoted: [id: string];
  dismissed: [id: string];
}>();

defineProps<{ items: Task[] }>();

const { promoteTask, cancelTask } = useTasks();
const toast = useToast();
const promoting = ref<string | null>(null);
const dismissing = ref<string | null>(null);

function priorityLabel(p?: number): string {
  if (p === 1) return 'High';
  if (p === 3) return 'Low';
  return 'Medium';
}

function priorityColor(p?: number): string {
  if (p === 1) return 'text-red-400';
  if (p === 3) return 'text-slate-500';
  return 'text-amber-400';
}

function truncate(s: string, n = 120): string {
  return s.length > n ? s.slice(0, n) + '…' : s;
}

async function onPromote(id: string) {
  if (promoting.value) return;
  promoting.value = id;
  try {
    await promoteTask(id);
    emit('promoted', id);
  } catch (e) {
    toast.add({ severity: 'error', summary: 'Promote Failed', detail: String(e), life: 5000 });
  } finally {
    promoting.value = null;
  }
}

async function onDismiss(id: string) {
  if (dismissing.value) return;
  dismissing.value = id;
  try {
    await cancelTask(id);
    emit('dismissed', id);
  } catch (e) {
    toast.add({ severity: 'error', summary: 'Dismiss Failed', detail: String(e), life: 5000 });
  } finally {
    dismissing.value = null;
  }
}
</script>
