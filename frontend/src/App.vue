<template>
  <div class="flex h-screen bg-[#050508] overflow-hidden">
    <AppSidebar @view-change="currentView = $event" />
    <div class="flex-1 flex flex-col overflow-hidden">
      <main class="flex-1 flex flex-col overflow-hidden" style="padding-bottom: 208px">
        <DashboardView v-if="currentView === 'dashboard'" />
        <BacklogView v-else-if="currentView === 'backlog'" />
        <HistoryView v-else-if="currentView === 'history'" />
        <ProvidersView v-else-if="currentView === 'providers'" />
        <DiscoveryView v-else-if="currentView === 'discovery'" />
        <AISessionsView v-else-if="currentView === 'ai-sessions'" />
        <SettingsView v-else-if="currentView === 'settings'" />
      </main>
      <LogPanel />
    </div>
    <Toast position="bottom-right" />
    <ConfirmDialog />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import Toast from 'primevue/toast'
import ConfirmDialog from 'primevue/confirmdialog'
import AppSidebar from './components/AppSidebar.vue'
import DashboardView from './views/DashboardView.vue'
import BacklogView from './views/BacklogView.vue'
import HistoryView from './views/HistoryView.vue'
import ProvidersView from './views/ProvidersView.vue'
import DiscoveryView from './views/DiscoveryView.vue'
import AISessionsView from './views/AISessionsView.vue'
import SettingsView from './views/SettingsView.vue'
import LogPanel from './components/LogPanel.vue'

const currentView = ref('dashboard')
</script>
