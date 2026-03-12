<template>
  <aside class="w-14 lg:w-56 flex flex-col bg-[#0d0d14] border-r border-white/5 flex-shrink-0">
    <!-- Logo -->
    <div class="p-4 border-b border-white/5">
      <div class="hidden lg:flex items-center gap-2">
        <div class="w-7 h-7 rounded-lg bg-violet-600 flex items-center justify-center text-white text-xs font-black">N</div>
        <span class="font-bold text-sm">nexus<span class="text-violet-400">Orchestrator</span></span>
      </div>
      <div class="lg:hidden flex justify-center">
        <div class="w-7 h-7 rounded-lg bg-violet-600 flex items-center justify-center text-white text-xs font-black">N</div>
      </div>
    </div>

    <!-- Project selector -->
    <ProjectSelector />

    <!-- Nav -->
    <nav class="flex-1 p-2">
      <button v-for="item in navItems" :key="item.label"
        @click="navigate(item.id)"
        :class="[
          'w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm transition-all mb-1',
          activeView === item.id
            ? 'bg-violet-600/15 text-violet-300 border border-violet-500/20'
            : 'text-slate-500 hover:text-slate-300 hover:bg-white/5'
        ]">
        <i :class="`pi ${item.icon} text-base`"></i>
        <span class="hidden lg:block font-medium">{{ item.label }}</span>
      </button>
    </nav>

    <!-- Footer -->
    <div class="p-3 border-t border-white/5">
      <div class="hidden lg:flex items-center gap-2 px-2 py-1.5 rounded-lg bg-white/[0.03]">
        <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse flex-shrink-0"></span>
        <span class="text-xs text-slate-500 truncate">nexus-daemon</span>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import ProjectSelector from './ProjectSelector.vue'

const emit = defineEmits<{ (e: 'view-change', id: string): void }>()

const activeView = ref('dashboard')
const navItems = [
  { id: 'dashboard', label: 'Task Queue', icon: 'pi-list' },
  { id: 'backlog', label: 'Backlog', icon: 'pi-bookmark' },
  { id: 'history', label: 'History', icon: 'pi-clock' },
  { id: 'providers', label: 'Providers', icon: 'pi-server' },
  { id: 'discovery', label: 'Discovery', icon: 'pi-search' },
  { id: 'ai-sessions', label: 'AI Sessions', icon: 'pi-share-alt' },
  { id: 'settings', label: 'Settings', icon: 'pi-cog' },
]

function navigate(id: string) {
  activeView.value = id
  emit('view-change', id)
}
</script>
