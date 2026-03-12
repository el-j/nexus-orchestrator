<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <header class="flex items-center justify-between px-5 py-3 border-b border-white/5 bg-[#0a0a10] flex-shrink-0">
      <div>
        <h1 class="text-sm font-bold text-white">System Discovery</h1>
        <p class="text-xs text-slate-500">
          <span class="text-cyan-400 font-semibold">{{ discovered.length }}</span> AI tools detected on this system
        </p>
      </div>
      <button
        @click="scanNow"
        :disabled="scanning"
        class="text-xs text-slate-400 hover:text-white px-3 py-1.5 rounded-lg border border-white/10
               hover:border-cyan-500/40 transition-all disabled:opacity-50 flex items-center gap-1.5"
      >
        <i :class="['pi pi-sync text-xs', scanning && 'animate-spin']"></i>
        {{ scanning ? 'Scanning…' : 'Scan Now' }}
      </button>
    </header>

    <!-- Content -->
    <div class="flex-1 overflow-auto p-5">
      <DiscoveredProvidersPanel
        :providers="discovered"
        :loading="loading"
        :scanning="scanning"
        @scan="scanNow"
        @promote="handlePromote"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useDiscovery } from '../composables/useDiscovery'
import DiscoveredProvidersPanel from '../components/DiscoveredProvidersPanel.vue'
import type { DiscoveredProvider } from '../types/discovery'

const { discovered, loading, scanning, scanNow } = useDiscovery()

function handlePromote(provider: DiscoveredProvider) {
  console.log('Promote discovered provider:', provider.name, provider.baseUrl)
  // TODO: navigate to providers view and open add form pre-filled
}
</script>
