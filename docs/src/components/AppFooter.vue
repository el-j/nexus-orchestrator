<template>
  <footer class="border-t border-white/5 bg-[#050508] mt-24 py-12">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="grid grid-cols-1 md:grid-cols-4 gap-8">
        <div class="md:col-span-1">
          <div class="text-lg font-bold mb-2">
            <span class="text-white">nexus</span><span class="gradient-text">Orchestrator</span>
          </div>
          <p class="text-sm text-slate-500">
            Local AI task orchestration with multi-provider routing and MCP support.
          </p>
        </div>
        <div v-for="col in footerCols" :key="col.heading" class="space-y-3">
          <h4 class="text-xs font-semibold uppercase tracking-wider text-slate-500">{{ col.heading }}</h4>
          <ul class="space-y-2 list-none p-0">
            <li v-for="link in col.links" :key="link.label">
              <a
                v-if="link.href"
                :href="link.href"
                target="_blank"
                rel="noopener"
                class="text-sm text-slate-400 hover:text-violet-400 transition-colors"
              >{{ link.label }}</a>
              <RouterLink
                v-else
                :to="link.to ?? '/'"
                class="text-sm text-slate-400 hover:text-violet-400 transition-colors"
              >{{ link.label }}</RouterLink>
            </li>
          </ul>
        </div>
      </div>
      <div class="mt-10 pt-8 border-t border-white/5 flex flex-col sm:flex-row items-center justify-between gap-4">
        <p class="text-xs text-slate-600">
          Built with Go 🔷 + Wails · MIT License · © {{ year }} nexusOrchestrator contributors
        </p>
        <a
          href="https://github.com/el-j/nexusOrchestrator"
          target="_blank"
          rel="noopener"
          class="text-xs text-slate-600 hover:text-violet-400 transition-colors flex items-center gap-1"
        >
          <i class="pi pi-github"></i> View on GitHub
        </a>
      </div>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'

const year = new Date().getFullYear()

interface FooterLink {
  label: string
  to?: string
  href?: string
}

interface FooterCol {
  heading: string
  links: FooterLink[]
}

const footerCols: FooterCol[] = [
  {
    heading: 'Docs',
    links: [
      { label: 'Getting Started', to: '/getting-started' },
      { label: 'Architecture', to: '/architecture' },
      { label: 'API Reference', to: '/api-reference' },
      { label: 'MCP Integration', to: '/mcp-integration' },
    ],
  },
  {
    heading: 'Download',
    links: [
      { label: 'Desktop App', to: '/downloads' },
      { label: 'CLI + Daemon', to: '/downloads' },
      { label: 'GitHub Releases', href: 'https://github.com/el-j/nexusOrchestrator/releases' },
    ],
  },
  {
    heading: 'Community',
    links: [
      { label: 'GitHub', href: 'https://github.com/el-j/nexusOrchestrator' },
      { label: 'Issues', href: 'https://github.com/el-j/nexusOrchestrator/issues' },
      { label: 'Releases', href: 'https://github.com/el-j/nexusOrchestrator/releases' },
    ],
  },
]
</script>
