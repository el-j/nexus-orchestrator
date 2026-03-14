<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <header class="flex items-center justify-between px-5 py-3 border-b border-white/5 bg-[#0a0a10] flex-shrink-0">
      <div>
        <h1 class="text-sm font-bold text-white">AI Agents</h1>
        <p class="text-xs text-slate-500">
          <span class="font-semibold" :class="activeCount > 0 ? 'text-emerald-400' : 'text-slate-500'">{{ activeCount }}</span>
          active ·
          <span class="text-cyan-400 font-semibold">{{ discovered.length }}</span>
          discovered
        </p>
      </div>
      <div class="flex items-center gap-2">
        <button
          v-if="hasActiveDelegatable"
          class="text-xs text-violet-400 hover:text-violet-300 px-3 py-1.5 rounded-lg border border-violet-500/30 hover:border-violet-500/60 transition-all"
          :disabled="delegatingAll"
          @click="delegateAll"
        >
          <i :class="['pi pi-arrow-right-arrow-left text-xs mr-1', delegatingAll && 'animate-spin']"></i>
          {{ delegatingAll ? 'Delegating…' : 'Delegate All Active' }}
        </button>
        <button
          class="text-xs text-slate-400 hover:text-white px-3 py-1.5 rounded-lg border border-white/10 hover:border-violet-500/40 transition-all"
          @click="refresh"
        >⟳ Refresh</button>
      </div>
    </header>

    <!-- Content -->
    <div class="flex-1 overflow-auto p-5">

      <!-- Loading -->
      <div v-if="loading" class="flex items-center justify-center py-16">
        <div class="w-6 h-6 border-2 border-violet-500 border-t-transparent rounded-full animate-spin"></div>
      </div>

      <!-- Error -->
      <div v-else-if="error" class="text-sm text-red-400 bg-red-500/10 border border-red-500/20 rounded-xl p-4">
        {{ error }}
      </div>

      <!-- Empty state -->
      <div v-else-if="sessions.length === 0 && discovered.length === 0" class="flex flex-col items-center justify-center py-20 text-center">
        <div class="text-4xl mb-4 opacity-40">🕵️</div>
        <p class="text-sm font-medium text-slate-400">No AI agents detected</p>
        <p class="text-xs text-slate-600 mt-1 max-w-sm">Connect VS Code Copilot, an MCP client, or another AI agent to see it here.</p>
      </div>

      <template v-else>
        <!-- Active sessions -->
        <section v-if="sessions.length > 0" class="mb-6">
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Registered Sessions</h2>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <AISessionCard
              v-for="s in sessions"
              :key="s.id"
              :session="s"
              @delegate="delegate"
            />
          </div>
        </section>

        <!-- Discovered agents -->
        <section v-if="discovered.length > 0">
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Discovered Agents</h2>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div
              v-for="agent in discovered"
              :key="agent.id"
              class="rounded-xl border border-white/10 bg-white/[0.02] p-4"
            >
              <div class="flex items-center gap-2 mb-1">
                <span
                  class="w-2 h-2 rounded-full flex-shrink-0"
                  :class="agent.isRunning ? 'bg-emerald-400' : 'bg-slate-600'"
                ></span>
                <span class="font-semibold text-sm text-white truncate">{{ agent.name }}</span>
                <span class="text-[10px] px-1.5 py-0.5 rounded bg-white/5 text-slate-400 ml-auto flex-shrink-0">{{ agent.kind }}</span>
              </div>
              <div class="text-[11px] text-slate-500 mb-1">{{ agent.detectionMethod }}</div>
              <div v-if="agent.mcpEndpoint" class="text-[10px] text-slate-600 font-mono truncate">{{ agent.mcpEndpoint }}</div>
              <div class="text-[10px] text-slate-700 mt-1">Last seen {{ relativeTime(agent.lastSeen) }}</div>
            </div>
          </div>
        </section>
      </template>

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useToast } from 'primevue/usetoast'
import type { AISession, DiscoveredAgent } from '../types/domain'
import AISessionCard from '../components/AISessionCard.vue'
import { resolveServerUrl } from '../composables/useServerUrl'

const toast = useToast()

const sessions = ref<AISession[]>([])
const discovered = ref<DiscoveredAgent[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const delegatingAll = ref(false)

let sessionsInterval: ReturnType<typeof setInterval> | null = null
let discoveredInterval: ReturnType<typeof setInterval> | null = null

const activeCount = computed(() => sessions.value.filter(s => s.status === 'active').length)
const hasActiveDelegatable = computed(() =>
  sessions.value.some(s => s.status === 'active' && !s.delegatedToNexus)
)

function relativeTime(ts: string | undefined): string {
  if (!ts) return 'never'
  const diff = Date.now() - new Date(ts).getTime()
  if (diff < 60_000) return 'just now'
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)} min ago`
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)} hr ago`
  return new Date(ts).toLocaleDateString()
}

async function fetchSessions() {
  try {
    const base = await resolveServerUrl()
    const r = await fetch(`${base}/api/ai-sessions`)
    if (!r.ok) throw new Error(`HTTP ${r.status}`)
    sessions.value = ((await r.json()) as AISession[]) ?? []
    error.value = null
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load AI sessions'
  }
}

async function fetchDiscovered() {
  try {
    const base = await resolveServerUrl()
    const r = await fetch(`${base}/api/ai-sessions/discovered`)
    if (!r.ok) return
    discovered.value = ((await r.json()) as DiscoveredAgent[]) ?? []
  } catch { /* non-critical */ }
}

async function refresh() {
  await Promise.all([fetchSessions(), fetchDiscovered()])
}

async function delegate(session: AISession) {
  try {
    const base = await resolveServerUrl()
    const resp = await fetch(`${base}/api/ai-sessions/${session.id}/delegate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: '{}',
    })
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
    const data = (await resp.json()) as { instruction?: string }
    if (data.instruction) {
      await navigator.clipboard.writeText(data.instruction)
      toast.add({ severity: 'success', summary: 'Delegated', detail: 'Instruction copied to clipboard', life: 3000 })
    } else {
      toast.add({ severity: 'success', summary: 'Delegated', life: 2000 })
    }
    await refresh()
  } catch (e) {
    toast.add({ severity: 'error', summary: 'Delegate failed', detail: e instanceof Error ? e.message : String(e), life: 4000 })
  }
}

async function delegateAll() {
  delegatingAll.value = true
  const active = sessions.value.filter(s => s.status === 'active' && !s.delegatedToNexus)
  for (const s of active) {
    await delegate(s)
  }
  delegatingAll.value = false
}

onMounted(async () => {
  loading.value = true
  await refresh()
  loading.value = false

  sessionsInterval = setInterval(fetchSessions, 10_000)
  discoveredInterval = setInterval(fetchDiscovered, 30_000)
})

onUnmounted(() => {
  if (sessionsInterval) clearInterval(sessionsInterval)
  if (discoveredInterval) clearInterval(discoveredInterval)
})
</script>
