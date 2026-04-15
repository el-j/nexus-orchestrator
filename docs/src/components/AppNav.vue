<template>
  <!-- Beta channel banner -->
  <div
    v-if="isBeta"
    class="fixed top-0 left-0 right-0 z-[60] flex items-center justify-center gap-3 px-4 py-1.5 text-xs font-medium bg-amber-500/10 border-b border-amber-500/30 text-amber-300"
  >
    <i class="pi pi-exclamation-triangle text-amber-400"></i>
    You're viewing <strong class="text-amber-200">beta / develop</strong> docs — features may
    change.
    <a :href="stableUrl" class="underline underline-offset-2 hover:text-white transition-colors"
      >View stable docs →</a
    >
  </div>
  <nav
    class="fixed left-0 right-0 z-50 backdrop-blur-md bg-[#050508]/80 border-b border-white/5"
    :class="isBeta ? 'top-8' : 'top-0'"
  >
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="flex items-center justify-between h-16">
        <!-- Logo -->
        <RouterLink to="/" class="flex items-center gap-2 text-lg font-bold">
          <img src="/favicon.svg" alt="nexusOrchestrator" class="w-8 h-8" />
          <span class="text-white">nexus-</span><span class="gradient-text">orchestrator</span>
        </RouterLink>

        <!-- Desktop Nav -->
        <ul class="hidden lg:flex items-center gap-6 text-sm font-medium list-none">
          <li v-for="link in navLinks" :key="link.to">
            <RouterLink
              :to="link.to"
              class="text-slate-400 hover:text-white transition-colors"
              active-class="text-violet-400"
            >
              {{ link.label }}
            </RouterLink>
          </li>
        </ul>

        <!-- Actions -->
        <div class="flex items-center gap-3">
          <!-- Version switcher -->
          <a
            v-if="!isBeta"
            :href="betaUrl"
            class="hidden sm:flex items-center gap-1.5 px-2.5 py-1 text-xs border border-amber-500/30 rounded-md text-amber-400 hover:border-amber-400 hover:text-amber-300 transition-all"
            title="Switch to beta docs"
          >
            <i class="pi pi-code text-xs"></i>
            beta
          </a>
          <a
            v-else
            :href="stableUrl"
            class="hidden sm:flex items-center gap-1.5 px-2.5 py-1 text-xs border border-violet-500/30 rounded-md text-violet-400 hover:border-violet-400 hover:text-violet-300 transition-all"
            title="Switch to stable docs"
          >
            <i class="pi pi-check-circle text-xs"></i>
            stable
          </a>
          <a
            href="https://github.com/el-j/nexus-orchestrator"
            target="_blank"
            rel="noopener"
            class="hidden sm:flex items-center gap-1.5 px-3 py-1.5 text-sm border border-white/10 rounded-lg text-slate-300 hover:border-violet-500/50 hover:text-white transition-all"
          >
            <i class="pi pi-github text-xs"></i>
            GitHub
          </a>
          <RouterLink
            to="/downloads"
            class="px-3 py-1.5 text-sm font-semibold rounded-lg bg-violet-600 hover:bg-violet-500 text-white transition-colors"
          >
            Download
          </RouterLink>
          <!-- Mobile menu button -->
          <button
            @click="mobileOpen = true"
            class="lg:hidden p-2 text-slate-400 hover:text-white"
            aria-label="Open menu"
          >
            <i class="pi pi-bars"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- Mobile drawer -->
    <Drawer v-model:visible="mobileOpen" position="right" style="width: 18rem; background: #0d0d14">
      <template #header>
        <span class="text-white font-bold"
          >nexus-<span class="gradient-text">orchestrator</span></span
        >
      </template>
      <nav class="flex flex-col gap-1 mt-4">
        <RouterLink
          v-for="link in navLinks"
          :key="link.to"
          :to="link.to"
          @click="mobileOpen = false"
          class="px-4 py-3 rounded-lg text-slate-300 hover:bg-white/5 hover:text-white transition-all"
          active-class="bg-violet-600/10 text-violet-400"
        >
          {{ link.label }}
        </RouterLink>
      </nav>
    </Drawer>
  </nav>
  <!-- Spacer: taller when beta banner is visible -->
  <div :class="isBeta ? 'h-24' : 'h-16'"></div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { RouterLink } from 'vue-router';
import Drawer from 'primevue/drawer';

const mobileOpen = ref(false);

// Baked in at SSG build time via VITE_DOCS_CHANNEL / VITE_BASE_PATH env vars.
const isBeta = import.meta.env.VITE_DOCS_CHANNEL === 'beta';
const stableUrl = import.meta.env.VITE_STABLE_URL ?? '/nexus-orchestrator/';
const betaUrl = import.meta.env.VITE_BETA_URL ?? '/nexus-orchestrator/beta/';

const navLinks = [
  { to: '/', label: 'Home' },
  { to: '/architecture', label: 'Architecture' },
  { to: '/downloads', label: 'Downloads' },
  { to: '/api-reference', label: 'API Reference' },
  { to: '/getting-started', label: 'Getting Started' },
  { to: '/mcp-integration', label: 'MCP Integration' },
];
</script>
