<template>
  <span :class="['inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold', colorClass]">
    <span v-if="status === 'PROCESSING'" class="w-1.5 h-1.5 rounded-full bg-current animate-pulse"></span>
    {{ status }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { TaskStatus } from '../types/domain'

const props = defineProps<{ status: TaskStatus }>()

const colorClass = computed(() => {
  switch (props.status) {
    case 'COMPLETED': return 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/20'
    case 'PROCESSING': return 'bg-blue-500/15 text-blue-400 border border-blue-500/20'
    case 'QUEUED': return 'bg-violet-500/15 text-violet-400 border border-violet-500/20'
    case 'FAILED': return 'bg-red-500/15 text-red-400 border border-red-500/20'
    case 'CANCELLED': return 'bg-slate-500/15 text-slate-400 border border-slate-500/20'
    case 'TOO_LARGE': return 'bg-orange-500/15 text-orange-400 border border-orange-500/20'
    case 'NO_PROVIDER': return 'bg-yellow-500/15 text-yellow-400 border border-yellow-500/20'
    default: return 'bg-slate-500/15 text-slate-400 border border-slate-500/20'
  }
})
</script>
