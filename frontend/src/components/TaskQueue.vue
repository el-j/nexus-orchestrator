<template>
  <div class="flex-1 overflow-auto">
    <!-- Loading skeleton -->
    <div v-if="loading" class="p-6 space-y-3">
      <Skeleton v-for="i in 3" :key="i" height="52px" class="rounded-lg" />
    </div>

    <!-- Empty state -->
    <div v-else-if="tasks.length === 0"
         class="flex flex-col items-center justify-center h-full py-20 text-center px-6">
      <div class="w-16 h-16 rounded-2xl bg-violet-600/10 border border-violet-500/20
                  flex items-center justify-center mb-4">
        <i class="pi pi-inbox text-2xl text-violet-400"></i>
      </div>
      <h3 class="font-semibold text-white mb-2">Queue is empty</h3>
      <p class="text-sm text-slate-500 max-w-xs">Submit your first task using the form below, or via CLI / MCP.</p>
    </div>

    <!-- Task list -->
    <div v-else class="p-4 space-y-2">
      <div v-for="task in tasks" :key="task.id"
           @click="$emit('select', task)"
           class="group flex items-start gap-4 p-4 rounded-xl border border-white/5 bg-[#0d0d14]
                  hover:border-violet-500/20 hover:bg-[#14141f] cursor-pointer transition-all">
        <TaskStatusBadge :status="task.status" class="mt-0.5 flex-shrink-0" />
        <div class="flex-1 min-w-0">
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0">
              <p class="text-sm font-medium text-white truncate">{{ task.instruction }}</p>
              <p class="text-xs text-slate-500 mt-0.5 truncate font-mono">
                {{ task.projectPath }}<span class="text-slate-600 mx-1">/</span>{{ task.targetFile }}
              </p>
            </div>
            <div class="flex items-center gap-2 flex-shrink-0">
              <span class="text-xs text-slate-600">{{ formatTime(task.createdAt) }}</span>
              <Button
                v-if="task.status === 'QUEUED'"
                @click.stop="$emit('cancel', task.id)"
                icon="pi pi-times"
                severity="danger"
                text
                rounded
                size="small"
                v-tooltip.left="'Cancel task'"
                class="opacity-0 group-hover:opacity-100 transition-opacity"
              />
            </div>
          </div>
          <!-- Logs preview for completed/failed -->
          <div v-if="task.logs && (task.status === 'COMPLETED' || task.status === 'FAILED')"
               class="mt-2 text-xs text-slate-600 font-mono truncate">
            {{ task.logs.slice(0, 80) }}{{ task.logs.length > 80 ? '…' : '' }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import Skeleton from 'primevue/skeleton'
import Button from 'primevue/button'
import type { Task } from '../types/domain'
import TaskStatusBadge from './TaskStatusBadge.vue'

defineProps<{ tasks: Task[]; loading: boolean }>()
defineEmits<{ select: [task: Task]; cancel: [id: string] }>()

function formatTime(iso: string): string {
  try {
    const d = new Date(iso)
    const diff = Date.now() - d.getTime()
    if (diff < 60000) return 'just now'
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`
    return d.toLocaleDateString()
  } catch {
    return ''
  }
}
</script>
