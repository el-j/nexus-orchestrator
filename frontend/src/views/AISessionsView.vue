<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <header class="flex items-center justify-between px-5 py-3 border-b border-white/5 bg-[#0a0a10] flex-shrink-0">
      <div>
        <h1 class="text-sm font-bold text-white">AI Sessions</h1>
        <p class="text-xs text-slate-500">
          <span class="font-semibold" :class="activeCount > 0 ? 'text-emerald-400' : 'text-slate-500'">{{ activeCount }}</span>
          active of {{ sessions.length }} total
        </p>
      </div>
      <button
        class="text-xs text-slate-400 hover:text-white px-3 py-1.5 rounded-lg border border-white/10 hover:border-violet-500/40 transition-all"
        @click="refresh"
      >⟳ Refresh</button>
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
      <div v-else-if="sessions.length === 0" class="flex flex-col items-center justify-center py-20 text-center">
        <div class="text-4xl mb-4 opacity-40">🤖</div>
        <p class="text-sm font-medium text-slate-400">No AI sessions detected</p>
        <p class="text-xs text-slate-600 mt-1 max-w-sm">Connect VS Code Copilot or an MCP client to see sessions here.</p>
      </div>

      <!-- Session cards -->
      <div v-else class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div
          v-for="s in sessions"
          :key="s.id"
          class="rounded-xl border bg-white/[0.03] p-4 transition-all"
          :class="{
            'border-l-4 border-emerald-500/60': s.status === 'active',
            'border-l-4 border-yellow-500/60': s.status === 'idle',
            'border-l-4 border-slate-600/60': s.status === 'disconnected',
          }"
        >
          <!-- Card header -->
          <div class="flex items-start justify-between gap-2 mb-2">
            <span class="font-semibold text-sm text-white truncate">{{ s.agentName }}</span>
            <span
              class="text-[10px] px-1.5 py-0.5 rounded font-medium flex-shrink-0"
              :class="{
                'bg-emerald-500/20 text-emerald-300': s.status === 'active',
                'bg-yellow-500/20 text-yellow-300': s.status === 'idle',
                'bg-slate-700/50 text-slate-400': s.status === 'disconnected',
              }"
            >{{ s.status }}</span>
          </div>

          <!-- Source badge -->
          <div class="flex items-center gap-2 mb-2">
            <span class="text-[11px] px-2 py-0.5 rounded bg-white/5 text-slate-400">
              <template v-if="s.source === 'mcp'">🤖 MCP</template>
              <template v-else-if="s.source === 'vscode'">🔵 VS Code</template>
              <template v-else>🌐 HTTP</template>
            </span>
            <span v-if="s.externalId" class="text-[10px] text-slate-600 font-mono truncate">{{ s.externalId }}</span>
          </div>

          <!-- Project path -->
          <div v-if="s.projectPath" class="text-[11px] text-slate-500 font-mono mb-2 truncate" :title="s.projectPath">
            {{ s.projectPath }}
          </div>

          <!-- Last activity -->
          <div class="text-[11px] text-slate-600 mb-3">
            Last active: {{ new Date(s.lastActivity).toLocaleString() }}
          </div>

          <!-- Disconnect button -->
          <button
            v-if="s.status !== 'disconnected'"
            class="text-xs text-red-400 hover:text-red-300 px-2.5 py-1 rounded-lg border border-red-500/20 hover:border-red-500/40 transition-all"
            @click="deregister(s.id)"
          >Disconnect</button>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAISessions } from '../composables/useAISessions'

const { sessions, loading, error, refresh, deregister } = useAISessions()

const activeCount = computed(() => sessions.value.filter(s => s.status === 'active').length)
</script>
