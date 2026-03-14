<template>
  <ErrorFallback v-if="appError" :error="appError" @retry="appError = null" />
  <template v-else>
    <div class="flex h-screen bg-[#050508] overflow-hidden">
      <AppSidebar @view-change="currentView = $event" />
      <div class="flex-1 flex flex-col overflow-hidden">
        <main class="flex-1 flex flex-col overflow-hidden" style="padding-bottom: 208px">
          <DashboardView v-if="currentView === 'dashboard'" />
          <BacklogView v-else-if="currentView === 'backlog'" />
          <HistoryView v-else-if="currentView === 'history'" />
          <LiveActivityView v-else-if="currentView === 'live-activity'" />
          <ProvidersView v-else-if="currentView === 'providers'" />
          <DiscoveryView v-else-if="currentView === 'discovery'" />
          <AISessionsView v-else-if="currentView === 'ai-sessions'" />
          <AIAgentsView v-else-if="currentView === 'ai-agents'" />
          <SettingsView v-else-if="currentView === 'settings'" />
        </main>
        <LogPanel />
      </div>
      <Toast position="bottom-right" />
      <ConfirmDialog />
    </div>
  </template>
</template>

<script setup lang="ts">
import { ref, onErrorCaptured } from 'vue'
import Toast from 'primevue/toast'
import ConfirmDialog from 'primevue/confirmdialog'
import AppSidebar from './components/AppSidebar.vue'
import DashboardView from './views/DashboardView.vue'
import BacklogView from './views/BacklogView.vue'
import HistoryView from './views/HistoryView.vue'
import LiveActivityView from './views/LiveActivityView.vue'
import ProvidersView from './views/ProvidersView.vue'
import DiscoveryView from './views/DiscoveryView.vue'
import AISessionsView from './views/AISessionsView.vue'
import AIAgentsView from './views/AIAgentsView.vue'
import SettingsView from './views/SettingsView.vue'
import LogPanel from './components/LogPanel.vue'
import ErrorFallback from './components/ErrorFallback.vue'

const currentView = ref('dashboard')
const appError = ref<Error | null>(null)

onErrorCaptured((err) => {
  console.error('[App] Component error captured:', err)
  appError.value = err instanceof Error ? err : new Error(String(err))
  return false
})
</script>
