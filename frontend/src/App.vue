<template>
  <div class="flex h-screen bg-[#050508] overflow-hidden">
    <AppSidebar />
    <main class="flex-1 flex flex-col overflow-hidden">
      <DashboardView />
    </main>
    <Toast position="bottom-right" />
    <ConfirmDialog />
  </div>
</template>

<script setup lang="ts">
import Toast from 'primevue/toast'
import ConfirmDialog from 'primevue/confirmdialog'
import AppSidebar from './components/AppSidebar.vue'
import DashboardView from './views/DashboardView.vue'
</script>
