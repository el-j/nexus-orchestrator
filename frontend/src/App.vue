<template>
  <ErrorFallback v-if="appError" :error="appError" @retry="appError = null" />
  <template v-else>
    <div class="flex h-screen bg-[#050508] overflow-hidden">
      <AppSidebar @view-change="currentView = $event" />
      <div class="flex-1 flex flex-col overflow-hidden">
        <main class="flex-1 flex flex-col overflow-hidden" style="padding-bottom: 208px">
          <MissionControlView v-if="currentView === 'mission-control'" />
          <AgentsView v-else-if="currentView === 'agents'" />
          <ProvidersView v-else-if="currentView === 'providers'" />
          <ProjectActivityView
            v-else-if="currentView === 'projects'"
            @navigate="currentView = $event"
          />
          <DiscoveredPlansView v-else-if="currentView === 'plans'" />
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
import { ref, onErrorCaptured } from 'vue';
import { useGlobalSSE } from './composables/useGlobalSSE';
import Toast from 'primevue/toast';
import ConfirmDialog from 'primevue/confirmdialog';
import AppSidebar from './components/AppSidebar.vue';
import MissionControlView from './views/MissionControlView.vue';
import ProvidersView from './views/ProvidersView.vue';
import AgentsView from './views/AgentsView.vue';
import DiscoveredPlansView from './views/DiscoveredPlansView.vue';
import ProjectActivityView from './views/ProjectActivityView.vue';
import SettingsView from './views/SettingsView.vue';
import LogPanel from './components/LogPanel.vue';
import ErrorFallback from './components/ErrorFallback.vue';

const { connect: connectSSE } = useGlobalSSE();
connectSSE();

const currentView = ref('mission-control');
const appError = ref<Error | null>(null);

onErrorCaptured((err) => {
  console.error('[App] Component error captured:', err);
  appError.value = err instanceof Error ? err : new Error(String(err));
  return false;
});
</script>
