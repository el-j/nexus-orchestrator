<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <header class="flex items-center justify-between px-5 py-3 border-b border-white/5 bg-[#0a0a10] flex-shrink-0">
      <div>
        <h1 class="text-sm font-bold text-white">Backlog</h1>
        <p class="text-xs text-slate-500">
          <span class="text-white font-semibold">{{ backlogTasks.length }}</span>
          {{ backlogTasks.length === 1 ? 'item' : 'items' }}
        </p>
      </div>
    </header>

    <!-- List (scrollable) -->
    <div class="flex-1 overflow-auto p-4">
      <BacklogList :items="backlogTasks" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useTasks } from '../composables/useTasks'
import BacklogList from '../components/BacklogList.vue'

const { backlogTasks } = useTasks()
</script>
