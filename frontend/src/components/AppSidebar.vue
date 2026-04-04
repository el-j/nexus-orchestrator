<template>
  <aside class="w-14 lg:w-56 flex flex-col bg-[#0d0d14] border-r border-white/5 flex-shrink-0">
    <!-- Logo -->
    <div class="p-4 border-b border-white/5">
      <div class="hidden lg:flex items-center gap-2">
        <div
          class="w-7 h-7 rounded-lg bg-violet-600 flex items-center justify-center text-white text-xs font-black"
        >
          N
        </div>
        <span class="font-bold text-sm"
          >nexus<span class="text-violet-400">Orchestrator</span></span
        >
      </div>
      <div class="lg:hidden flex justify-center">
        <div
          class="w-7 h-7 rounded-lg bg-violet-600 flex items-center justify-center text-white text-xs font-black"
        >
          N
        </div>
      </div>
    </div>

    <!-- Project selector -->
    <ProjectSelector />

    <!-- Nav -->
    <nav class="flex-1 p-2">
      <button
        v-for="item in navRoutes"
        :key="item.name"
        @click="navigate(item.path as string)"
        :class="[
          'w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm transition-all mb-1',
          isActive(item.path)
            ? 'bg-violet-600/15 text-violet-300 border border-violet-500/20'
            : 'text-slate-500 hover:text-slate-300 hover:bg-white/5',
        ]"
      >
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
import { useRouter, useRoute } from 'vue-router';
import ProjectSelector from './ProjectSelector.vue';
import routes from '../router/routes';
const router = useRouter();
const route = useRoute();

const navRoutes = routes.filter((r) => r.nav);

function navigate(path: string) {
  void router.push(path);
}

function isActive(path: string): boolean {
  if (path === '/') return route.path === '/';
  return route.path.startsWith(path);
}
</script>
