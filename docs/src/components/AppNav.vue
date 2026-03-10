<template>
  <nav class="fixed top-0 left-0 right-0 z-50 backdrop-blur-md bg-[#050508]/80 border-b border-white/5">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="flex items-center justify-between h-16">
        <!-- Logo -->
        <RouterLink to="/" class="flex items-center gap-2 text-lg font-bold">
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
    <Drawer v-model:visible="mobileOpen" position="right" style="width: 18rem; background: #0d0d14;">
      <template #header>
        <span class="text-white font-bold">nexus-<span class="gradient-text">orchestrator</span></span>
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
  <!-- Spacer for fixed nav -->
  <div class="h-16"></div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import Drawer from 'primevue/drawer'

const mobileOpen = ref(false)

const navLinks = [
  { to: '/', label: 'Home' },
  { to: '/architecture', label: 'Architecture' },
  { to: '/downloads', label: 'Downloads' },
  { to: '/api-reference', label: 'API Reference' },
  { to: '/getting-started', label: 'Getting Started' },
  { to: '/mcp-integration', label: 'MCP Integration' },
]
</script>
