<template>
  <ErrorFallback v-if="appError" :error="appError" @retry="appError = null" />
  <template v-else>
    <div class="flex h-screen bg-[#050508] overflow-hidden">
      <AppSidebar />
      <div class="flex-1 flex flex-col overflow-hidden">
        <main class="flex-1 flex flex-col overflow-hidden min-h-0">
          <RouterView />
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
import { RouterView } from 'vue-router';
import { useGlobalSSE } from './composables/useGlobalSSE';
import Toast from 'primevue/toast';
import ConfirmDialog from 'primevue/confirmdialog';
import AppSidebar from './components/AppSidebar.vue';
import LogPanel from './components/LogPanel.vue';
import ErrorFallback from './components/ErrorFallback.vue';

const { connect: connectSSE } = useGlobalSSE();
connectSSE();

const appError = ref<Error | null>(null);

onErrorCaptured((err) => {
  console.error('[App] Component error captured:', err);
  appError.value = err instanceof Error ? err : new Error(String(err));
  return false;
});
</script>
