<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <header class="flex items-center justify-between px-5 py-3 border-b border-white/5 bg-[#0a0a10] flex-shrink-0">
      <div>
        <h1 class="text-sm font-bold text-white">Providers</h1>
        <p class="text-xs text-slate-500">
          <span class="font-semibold" :class="activeCount > 0 ? 'text-emerald-400' : 'text-slate-500'">{{ activeCount }}</span>
          active of {{ providers.length }} registered
          <span class="text-slate-700 mx-1">·</span>
          <span class="text-cyan-400 font-semibold">{{ discovered.length }}</span> discovered
        </p>
      </div>
      <button
        class="text-xs text-slate-400 hover:text-white px-3 py-1.5 rounded-lg border border-white/10 hover:border-violet-500/40 transition-all"
        @click="refresh(); discoveryRefresh()"
      >⟳ Refresh</button>
    </header>

    <!-- Content -->
    <div class="flex-1 overflow-auto p-5 space-y-6">

      <!-- Active providers grid -->
      <section>
        <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Active Providers</h2>
        <div v-if="providers.length === 0" class="text-sm text-slate-600 py-8 text-center">
          No providers detected — start LM Studio or Ollama
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div
            v-for="p in providers"
            :key="p.name"
            class="rounded-xl border p-4 transition-all"
            :class="p.active ? 'border-emerald-500/30 bg-emerald-500/5' : 'border-red-500/20 bg-red-500/5'"
          >
            <div class="flex items-start justify-between gap-2 mb-2">
              <div class="flex items-center gap-2">
                <span :class="['w-2 h-2 rounded-full flex-shrink-0 mt-0.5', p.active ? 'bg-emerald-400 animate-pulse' : 'bg-red-500']"></span>
                <span class="font-semibold text-sm text-white">{{ p.name }}</span>
              </div>
              <span class="text-[10px] px-1.5 py-0.5 rounded font-medium" :class="p.active ? 'bg-emerald-500/20 text-emerald-300' : 'bg-red-500/20 text-red-400'">
                {{ p.active ? 'Active' : 'Unreachable' }}
              </span>
            </div>
            <div class="text-[11px] text-slate-500 font-mono mb-2">{{ p.baseURL }}</div>
            <div v-if="p.active && p.activeModel" class="text-xs text-slate-400 mb-1">
              Active model: <span class="text-violet-300 font-mono">{{ p.activeModel }}</span>
            </div>
            <div v-if="p.active && p.models?.length" class="flex flex-wrap gap-1 mt-2">
              <span
                v-for="m in p.models"
                :key="m"
                class="text-[10px] px-1.5 py-0.5 rounded font-mono"
                :class="m === p.activeModel ? 'border border-violet-500/30 bg-violet-500/10 text-violet-300' : 'bg-white/5 text-slate-400'"
              >{{ m }}</span>
            </div>
            <div v-if="!p.active && p.error" class="text-[11px] text-amber-500/80 mt-2 font-mono">{{ p.error }}</div>
          </div>
        </div>
      </section>

      <!-- Discovered providers (auto-detected from system) -->
      <section>
        <DiscoveredProvidersPanel
          :providers="discovered"
          :loading="discoveryLoading"
          :scanning="scanning"
          @scan="scanNow"
          @promote="handlePromote"
        />
      </section>

      <!-- Configured providers (CRUD) -->
      <section>
        <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Configured Providers</h2>
        <ProviderStatus :providers="providers" :refresh="refresh" :hide-discovered="true" />
      </section>

    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useProviders } from '../composables/useProviders'
import { useDiscovery } from '../composables/useDiscovery'
import ProviderStatus from '../components/ProviderStatus.vue'
import DiscoveredProvidersPanel from '../components/DiscoveredProvidersPanel.vue'
import type { DiscoveredProvider } from '../types/discovery'

const { providers, refresh } = useProviders()
const {
  discovered,
  loading: discoveryLoading,
  scanning,
  refresh: discoveryRefresh,
  scanNow,
} = useDiscovery()

const activeCount = computed(() => providers.value.filter(p => p.active).length)

function handlePromote(provider: DiscoveredProvider) {
  // Pre-fill the configured provider form with discovered data
  // For now, we open the add form — a future iteration can pre-fill fields
  console.log('Promote discovered provider:', provider.name, provider.baseURL)
  // TODO: open ProviderConfigForm pre-filled with provider.kind, provider.baseURL, provider.name
}
</script>
