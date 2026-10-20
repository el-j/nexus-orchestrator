<!--
  SessionCard: one registered AI session of the Agents view, with its status, source, project and last
  activity, and the Delegate, Disconnect and Terminate actions. Props: session. Emits open (card
  clicked), delegate, disconnect and terminate (the buttons; they do not open the card).
-->
<template>
  <div
    class="rounded-xl border bg-white/[0.03] p-3 transition-all border-l-4 cursor-pointer hover:bg-white/[0.05]"
    :class="{
      'border-l-emerald-500/60': session.status === 'active',
      'border-l-yellow-500/60': session.status === 'idle',
      'border-l-slate-600/60': session.status === 'disconnected',
    }"
    @click="emit('open')"
  >
    <!-- Header: status dot + agent name + status badge -->
    <div class="flex items-center gap-2 mb-2">
      <span
        class="w-2 h-2 rounded-full flex-shrink-0"
        :class="{
          'bg-emerald-400': session.status === 'active',
          'bg-yellow-400': session.status === 'idle',
          'bg-slate-600': session.status === 'disconnected',
        }"
      ></span>
      <span class="font-semibold text-sm text-white truncate flex-1">{{ session.agentName }}</span>
      <span
        class="text-[10px] px-1.5 py-0.5 rounded font-medium flex-shrink-0"
        :class="{
          'bg-emerald-500/20 text-emerald-300': session.status === 'active',
          'bg-yellow-500/20 text-yellow-300': session.status === 'idle',
          'bg-slate-700/50 text-slate-400': session.status === 'disconnected',
        }"
        >{{ session.status }}</span
      >
    </div>

    <!-- Source badge -->
    <div class="flex items-center gap-2 mb-2">
      <span class="text-[11px] px-2 py-0.5 rounded bg-white/5 text-slate-400">
        <template v-if="session.source === 'mcp'">🤖 MCP</template>
        <template v-else-if="session.source === 'vscode'">🔵 VS Code</template>
        <template v-else>🌐 HTTP</template>
      </span>
      <span v-if="session.externalId" class="text-[10px] text-slate-600 font-mono truncate">{{
        session.externalId
      }}</span>
    </div>

    <!-- Project path -->
    <div
      v-if="session.projectPath"
      class="text-[11px] text-slate-500 font-mono mb-2 truncate"
      :title="session.projectPath"
    >
      {{ shortPath(session.projectPath) }}
    </div>

    <!-- Last activity -->
    <div class="text-[11px] text-slate-600 mb-2">
      Last active: {{ relativeTime(session.lastActivity) }}
    </div>

    <!-- Actions -->
    <div class="flex items-center gap-2 flex-wrap">
      <span
        v-if="session.delegatedToNexus"
        class="text-[10px] px-1.5 py-0.5 rounded bg-violet-500/20 text-violet-300"
        >Delegated</span
      >
      <button
        v-else-if="session.status !== 'disconnected'"
        class="text-xs text-violet-300 hover:text-violet-200 px-2.5 py-1 rounded-lg border border-violet-500/20 hover:border-violet-500/40 transition-all"
        title="Hand this agent's work to Nexus"
        @click.stop="emit('delegate')"
      >
        Delegate
      </button>
      <button
        v-if="session.status !== 'disconnected'"
        class="text-xs text-red-400 hover:text-red-300 px-2.5 py-1 rounded-lg border border-red-500/20 hover:border-red-500/40 transition-all"
        @click.stop="emit('disconnect')"
      >
        Disconnect
      </button>
      <button
        v-if="session.status !== 'disconnected'"
        class="text-xs text-red-400 hover:text-red-300 px-2.5 py-1 rounded-lg border border-red-500/20 hover:border-red-500/40 transition-all"
        title="Stop the agent's process (only a process Nexus found itself is signalled)"
        @click.stop="emit('terminate')"
      >
        Terminate
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { AISession } from '../types/domain';
import { relativeTime } from '../utils/time';
import { shortPath } from '../utils/path';

defineProps<{ session: AISession }>();
const emit = defineEmits<{
  (e: 'open'): void;
  (e: 'delegate'): void;
  (e: 'disconnect'): void;
  (e: 'terminate'): void;
}>();
</script>
