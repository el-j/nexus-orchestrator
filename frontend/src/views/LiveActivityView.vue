<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <header class="flex items-center justify-between px-5 py-3 border-b border-white/5 bg-[#0a0a10] shrink-0">
      <div>
        <h1 class="text-sm font-bold text-white">Live AI Activity</h1>
        <p class="text-xs text-slate-500">
          <span class="font-semibold text-emerald-400">{{ activeCount }}</span> active ·
          <span class="font-semibold text-violet-400">{{ providersFound }}</span> providers detected
        </p>
      </div>
      <button
        class="text-xs text-slate-400 hover:text-white px-3 py-1.5 rounded-lg border border-white/10 hover:border-violet-500/40 transition-all"
        :disabled="scanning"
        @click="scanNow"
      >{{ scanning ? '⏳ Scanning…' : '⟳ Scan Now' }}</button>
    </header>

    <div class="flex-1 overflow-auto p-5 space-y-6">

      <!-- Actively generating indicator -->
      <div v-if="generating.length > 0" class="rounded-xl border border-emerald-500/30 bg-emerald-500/5 p-4">
        <div class="flex items-center gap-2 mb-3">
          <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
          <span class="text-xs font-semibold text-emerald-400 uppercase tracking-wider">Actively Generating</span>
        </div>
        <div class="flex flex-wrap gap-2">
          <div v-for="p in generating" :key="p.id"
            class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-emerald-500/10 border border-emerald-500/20">
            <span class="text-xs font-medium text-emerald-300">{{ p.name }}</span>
            <span v-if="p.activeModels?.length" class="text-[10px] text-emerald-500 font-mono">
              {{ p.activeModels.join(', ') }}
            </span>
          </div>
        </div>
      </div>

      <!-- AI Sessions -->
      <section>
        <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">
          Connected Agents
          <span class="ml-1 text-violet-400">{{ activeSessions.length }}</span>
        </h2>

        <div v-if="activeSessions.length === 0" class="text-xs text-slate-600 italic py-2">
          No active agent sessions.
          <span class="text-slate-500">Connect VS Code Copilot or an MCP client to see them here.</span>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div
            v-for="s in activeSessions"
            :key="s.id"
            class="rounded-xl border border-emerald-500/20 bg-white/2 p-3 flex items-center gap-3"
          >
            <span class="w-2 h-2 rounded-full shrink-0"
              :class="s.status === 'active' ? 'bg-emerald-400 animate-pulse' : 'bg-yellow-400'"
            ></span>
            <div class="flex-1 min-w-0">
              <div class="text-xs font-semibold text-white truncate">{{ s.agentName }}</div>
              <div class="text-[10px] text-slate-500">
                {{ s.source }} · Last active {{ timeAgo(s.lastActivity) }}
              </div>
            </div>
            <span
              class="text-[10px] px-1.5 py-0.5 rounded font-medium shrink-0"
              :class="s.status === 'active' ? 'bg-emerald-500/20 text-emerald-300' : 'bg-yellow-500/20 text-yellow-300'"
            >{{ s.status }}</span>
          </div>
        </div>
      </section>

      <!-- Detected AI Providers -->
      <section>
        <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">
          Detected on This Machine
          <span class="ml-1 text-violet-400">{{ discovered.length }}</span>
        </h2>

        <div v-if="loading && discovered.length === 0" class="flex items-center justify-center py-8">
          <div class="w-5 h-5 border-2 border-violet-500 border-t-transparent rounded-full animate-spin"></div>
        </div>

        <div v-else-if="discovered.length === 0" class="text-xs text-slate-600 italic py-2">
          No AI tools detected on this machine. Try scanning.
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          <div
            v-for="p in discovered"
            :key="p.id"
            class="rounded-xl border bg-white/2 p-3 transition-all"
            :class="{
              'border-emerald-500/40': p.generating,
              'border-violet-500/20': p.status === 'reachable' && !p.generating,
              'border-slate-700/40': p.status !== 'reachable',
            }"
          >
            <div class="flex items-start justify-between gap-2 mb-1.5">
              <span class="text-xs font-semibold text-white truncate">{{ p.name }}</span>
              <div class="flex items-center gap-1.5 shrink-0">
                <span v-if="p.generating"
                  class="text-[10px] px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-300 font-medium">
                  generating
                </span>
                <span
                  class="text-[10px] px-1.5 py-0.5 rounded font-medium"
                  :class="{
                    'bg-emerald-500/20 text-emerald-300': p.status === 'reachable',
                    'bg-blue-500/20 text-blue-300': p.status === 'running',
                    'bg-slate-700/50 text-slate-400': p.status ===  'installed',
                  }"
                >{{ p.status }}</span>
              </div>
            </div>

            <div class="text-[10px] text-slate-500 mb-1.5">
              {{ methodLabel(p.method) }} · {{ p.kind }}
            </div>

            <!-- Active models (Ollama /api/ps) -->
            <div v-if="p.activeModels?.length" class="flex flex-wrap gap-1 mb-1.5">
              <span v-for="m in p.activeModels" :key="m"
                class="text-[10px] px-1.5 py-0.5 rounded-full bg-violet-500/15 text-violet-300 font-mono">
                {{ m }}
              </span>
            </div>

            <!-- All available models (collapsed preview) -->
            <div v-else-if="p.models?.length" class="text-[10px] text-slate-600 font-mono truncate">
              {{ p.models.slice(0, 2).join(', ') }}{{ p.models.length > 2 ? ` +${p.models.length - 2}` : '' }}
            </div>
          </div>
        </div>
      </section>

    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useDiscovery } from '../composables/useDiscovery'
import { useAISessions } from '../composables/useAISessions'

const { discovered, loading, scanning, scanNow } = useDiscovery()
const { sessions } = useAISessions()

const activeSessions = computed(() =>
  sessions.value.filter(s => s.status === 'active' || s.status === 'idle')
)

const generating = computed(() =>
  discovered.value.filter(p => p.generating)
)

const activeCount = computed(() =>
  activeSessions.value.length + generating.value.length
)

const providersFound = computed(() => discovered.value.length)

function methodLabel(method: string): string {
  return method === 'port' ? 'Port' : method === 'cli' ? 'CLI' : 'Process'
}

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const s = Math.floor(diff / 1000)
  if (s < 60) return `${s}s ago`
  if (s < 3600) return `${Math.floor(s / 60)}m ago`
  return `${Math.floor(s / 3600)}h ago`
}
</script>
