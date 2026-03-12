<template>
  <div class="flex flex-col gap-4">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider">Discovered on System</h2>
      <button
        @click="$emit('scan')"
        :disabled="scanning"
        class="text-[10px] text-cyan-400 hover:text-cyan-300 transition-colors px-2 py-1 rounded-md
               border border-cyan-500/20 hover:border-cyan-500/40 hover:bg-cyan-500/5
               disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5"
      >
        <i :class="['pi pi-sync text-[10px]', scanning && 'animate-spin']"></i>
        {{ scanning ? 'Scanning…' : 'Scan Now' }}
      </button>
    </div>

    <!-- Empty state -->
    <div v-if="!loading && providers.length === 0"
         class="text-center py-10 text-sm text-slate-600 border border-dashed border-white/10 rounded-xl">
      <i class="pi pi-search text-2xl text-slate-700 mb-2 block"></i>
      No AI tools detected — click Scan Now to search
    </div>

    <!-- Loading skeleton -->
    <div v-else-if="loading" class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div v-for="i in 3" :key="i"
           class="h-36 rounded-xl border border-white/5 bg-white/[0.02] animate-pulse" />
    </div>

    <!-- Cards grid -->
    <div v-else class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div
        v-for="p in providers"
        :key="p.id"
        class="rounded-xl border p-4 transition-all"
        :class="cardBorder(p.status)"
      >
        <!-- Top row: icon + name + status badge -->
        <div class="flex items-start justify-between gap-2 mb-3">
          <div class="flex items-center gap-2.5">
            <div class="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0"
                 :class="iconBg(p.status)">
              <i :class="['pi text-sm', methodIcon(p.method)]"></i>
            </div>
            <div class="min-w-0">
              <p class="font-semibold text-sm text-white truncate">{{ p.name }}</p>
              <p class="text-[10px] text-slate-500 font-mono">{{ p.kind }}</p>
            </div>
          </div>
          <span class="text-[10px] px-1.5 py-0.5 rounded font-medium flex-shrink-0"
                :class="statusBadge(p.status)">
            {{ statusLabel(p.status) }}
          </span>
        </div>

        <!-- Details -->
        <div class="space-y-1.5 mb-3">
          <div v-if="p.baseURL" class="flex items-center gap-1.5">
            <i class="pi pi-globe text-[10px] text-slate-600"></i>
            <span class="text-[10px] text-slate-500 font-mono truncate">{{ p.baseURL }}</span>
          </div>
          <div v-if="p.cliPath" class="flex items-center gap-1.5">
            <i class="pi pi-chevron-right text-[10px] text-slate-600"></i>
            <span class="text-[10px] text-slate-500 font-mono truncate">{{ p.cliPath }}</span>
          </div>
          <div v-if="p.processName" class="flex items-center gap-1.5">
            <i class="pi pi-cog text-[10px] text-slate-600"></i>
            <span class="text-[10px] text-slate-500 font-mono">{{ p.processName }}</span>
          </div>
          <div class="text-[10px] text-slate-600">
            Last seen {{ timeAgo(p.lastSeen) }}
          </div>
        </div>

        <!-- Models (if reachable) -->
        <div v-if="p.models?.length" class="flex flex-wrap gap-1 mb-3">
          <span
            v-for="m in p.models.slice(0, 5)"
            :key="m"
            class="text-[10px] px-1.5 py-0.5 rounded bg-white/5 text-slate-400 font-mono"
          >{{ m }}</span>
          <span v-if="p.models.length > 5"
                class="text-[10px] px-1.5 py-0.5 rounded bg-white/5 text-slate-500">
            +{{ p.models.length - 5 }} more
          </span>
        </div>

        <!-- Action -->
        <button
          v-if="p.status === 'reachable'"
          @click="$emit('promote', p)"
          class="w-full text-xs font-medium py-1.5 rounded-lg border transition-all
                 border-emerald-500/30 text-emerald-400 hover:bg-emerald-500/10 hover:border-emerald-500/50"
        >
          <i class="pi pi-plus-circle mr-1 text-[10px]"></i>
          Promote to Active
        </button>
        <div v-else-if="p.status === 'installed'"
             class="text-[10px] text-amber-500/70 text-center py-1">
          CLI found — start the server to activate
        </div>
        <div v-else
             class="text-[10px] text-slate-600 text-center py-1">
          Process detected — no API endpoint found
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { DiscoveredProvider, DiscoveryStatus, DiscoveryMethod } from '../types/discovery'

defineProps<{
  providers: DiscoveredProvider[]
  loading: boolean
  scanning: boolean
}>()

defineEmits<{
  promote: [provider: DiscoveredProvider]
  scan: []
}>()

function cardBorder(status: DiscoveryStatus): string {
  switch (status) {
    case 'reachable': return 'border-emerald-500/20 bg-emerald-500/[0.03]'
    case 'installed': return 'border-amber-500/20 bg-amber-500/[0.03]'
    case 'running': return 'border-amber-500/15 bg-amber-500/[0.02]'
    default: return 'border-white/5 bg-white/[0.02]'
  }
}

function iconBg(status: DiscoveryStatus): string {
  switch (status) {
    case 'reachable': return 'bg-emerald-500/15 text-emerald-400'
    case 'installed': return 'bg-amber-500/15 text-amber-400'
    case 'running': return 'bg-amber-500/10 text-amber-500'
    default: return 'bg-white/5 text-slate-500'
  }
}

function methodIcon(method: DiscoveryMethod): string {
  switch (method) {
    case 'port': return 'pi-wifi'       // network / port scan
    case 'cli': return 'pi-terminal'    // CLI / which
    case 'process': return 'pi-cog'     // process list
    default: return 'pi-question-circle'
  }
}

function statusBadge(status: DiscoveryStatus): string {
  switch (status) {
    case 'reachable': return 'bg-emerald-500/20 text-emerald-300'
    case 'installed': return 'bg-amber-500/20 text-amber-300'
    case 'running': return 'bg-amber-500/15 text-amber-400'
    default: return 'bg-slate-500/15 text-slate-400'
  }
}

function statusLabel(status: DiscoveryStatus): string {
  switch (status) {
    case 'reachable': return 'API Reachable'
    case 'installed': return 'Installed'
    case 'running': return 'Running'
    default: return status
  }
}

function timeAgo(iso: string): string {
  try {
    const diff = Date.now() - new Date(iso).getTime()
    if (diff < 60_000) return 'just now'
    if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`
    if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`
    return new Date(iso).toLocaleDateString()
  } catch {
    return ''
  }
}
</script>
