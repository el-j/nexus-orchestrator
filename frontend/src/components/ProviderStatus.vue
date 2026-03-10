<template>
  <div class="flex items-center gap-2 flex-wrap">
    <span class="text-xs text-slate-500 font-medium">Providers:</span>
    <div v-for="p in providers" :key="p.Name"
         class="flex items-center gap-1.5 px-2.5 py-1 rounded-full border text-xs transition-all"
         :class="p.Active
           ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-400'
           : 'border-white/5 bg-white/[0.03] text-slate-600'">
      <span :class="['w-1.5 h-1.5 rounded-full', p.Active ? 'bg-emerald-400 animate-pulse' : 'bg-slate-600']"></span>
      {{ p.Name }}
      <span v-if="p.Active && p.ActiveModel" class="text-slate-500 font-mono text-[10px] max-w-[80px] truncate">
        {{ p.ActiveModel }}
      </span>
    </div>
    <div v-if="providers.length === 0" class="text-xs text-slate-600">No providers detected</div>
  </div>
</template>

<script setup lang="ts">
import type { ProviderInfo } from '../types/domain'
defineProps<{ providers: ProviderInfo[] }>()
</script>
