<template>
  <div
    class="rounded-lg border-l-4 bg-white/2 px-3 py-2 flex flex-col gap-0.5"
    :class="borderClass"
    :title="metaTitle"
  >
    <!-- Row 1: emoji + agent · model | timestamp -->
    <div class="flex items-center justify-between gap-2">
      <span class="text-xs font-semibold text-white truncate">
        {{ emoji }}
        <span :class="accentClass">{{ activity.agentName }}</span>
        <span v-if="activity.model" class="text-slate-500 font-normal font-mono text-[10px]">
          · {{ activity.model }}</span
        >
      </span>
      <span class="text-[10px] text-slate-500 shrink-0">{{ timeAgo(activity.timestamp) }}</span>
    </div>

    <!-- Row 2: summary | token count -->
    <div class="flex items-center justify-between gap-2">
      <span
        class="text-xs text-slate-300 truncate"
        :class="{
          'font-mono': activity.activityType === 'tool_use',
          'italic text-slate-400': activity.activityType === 'thinking',
        }"
        >{{ activity.summary }}</span
      >
      <span v-if="tokenTotal" class="text-[10px] text-slate-500 shrink-0">🪙 {{ tokenTotal }}</span>
    </div>

    <!-- Row 3: project path (last 2 parts) -->
    <div v-if="activity.projectPath" class="text-[10px] text-slate-600 truncate">
      {{ shortPath }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { AIActivity } from '../types/domain';
import { timeAgo } from '../utils/time';

const props = defineProps<{ activity: AIActivity }>();

const emoji = computed(() => {
  switch (props.activity.activityType) {
    case 'message':
      return '💬';
    case 'tool_use':
      return '🔧';
    case 'thinking':
      return '🧠';
    case 'file_edit':
      return '📄';
    case 'generation':
      return '⚡';
  }
});

const borderClass = computed(() => {
  switch (props.activity.activityType) {
    case 'message':
      return 'border-blue-400';
    case 'tool_use':
      return 'border-green-400';
    case 'thinking':
      return 'border-purple-400';
    case 'file_edit':
      return 'border-orange-400';
    case 'generation':
      return 'border-yellow-400';
  }
});

const accentClass = computed(() => {
  switch (props.activity.activityType) {
    case 'message':
      return 'text-blue-400';
    case 'tool_use':
      return 'text-green-400';
    case 'thinking':
      return 'text-purple-400';
    case 'file_edit':
      return 'text-orange-400';
    case 'generation':
      return 'text-yellow-400';
  }
});

const tokenTotal = computed(() => {
  const total = (props.activity.tokensIn ?? 0) + (props.activity.tokensOut ?? 0);
  return total > 0 ? total.toLocaleString() : null;
});

const shortPath = computed(() => {
  if (!props.activity.projectPath) return '';
  const parts = props.activity.projectPath.replace(/\\/g, '/').split('/').filter(Boolean);
  return parts.slice(-2).join('/');
});

const metaTitle = computed(() => {
  const lines: string[] = [
    `Agent: ${props.activity.agentName}`,
    `Type: ${props.activity.activityType}`,
    `Time: ${props.activity.timestamp}`,
  ];
  if (props.activity.projectPath) lines.push(`Project: ${props.activity.projectPath}`);
  if (props.activity.model) lines.push(`Model: ${props.activity.model}`);
  if (props.activity.tokensIn) lines.push(`Tokens in: ${props.activity.tokensIn}`);
  if (props.activity.tokensOut) lines.push(`Tokens out: ${props.activity.tokensOut}`);
  if (props.activity.metadata) {
    for (const [k, v] of Object.entries(props.activity.metadata)) {
      lines.push(`${k}: ${v}`);
    }
  }
  return lines.join('\n');
});
</script>
