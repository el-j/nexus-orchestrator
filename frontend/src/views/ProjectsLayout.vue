<!--
  Projects hub: one place for everything that belongs to a project (activity, plans, finished
  tasks, the backlog and the brain), as tabs over the routed views.
-->
<template>
  <div class="flex flex-col h-full overflow-hidden">
    <nav
      class="flex items-center gap-1 px-5 py-2 border-b border-white/5 bg-[#0a0a10] shrink-0"
      aria-label="Project sections"
    >
      <button
        v-for="tab in tabs"
        :key="tab.path"
        :data-testid="`project-tab-${tab.label.toLowerCase()}`"
        :aria-current="isActive(tab.path) ? 'page' : undefined"
        :class="[
          'px-3 py-1 rounded-lg text-xs font-medium transition-all border',
          isActive(tab.path)
            ? 'bg-violet-600/20 text-violet-300 border-violet-500/30'
            : 'text-slate-500 hover:text-slate-300 hover:bg-white/5 border-transparent',
        ]"
        @click="router.push(tab.path)"
      >
        {{ tab.label }}
      </button>
    </nav>
    <div class="flex-1 min-h-0 overflow-hidden">
      <router-view />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router';

const router = useRouter();
const route = useRoute();

const tabs = [
  { label: 'Activity', path: '/projects' },
  { label: 'Plans', path: '/projects/plans' },
  { label: 'History', path: '/projects/history' },
  { label: 'Backlog', path: '/projects/backlog' },
  { label: 'Brain', path: '/projects/brain' },
];

// The timeline is opened from the activity tab, so that tab stays lit there.
function isActive(path: string): boolean {
  return route.path === path || (path === '/projects' && route.path === '/projects/live-activity');
}
</script>
