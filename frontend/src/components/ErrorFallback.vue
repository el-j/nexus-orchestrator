<template>
  <div class="error-fallback p-4 text-center">
    <p class="text-red-500 font-semibold mb-2">Something went wrong</p>
    <button class="p-button p-button-sm" @click="$emit('retry')">Retry</button>
  </div>
</template>

<script setup lang="ts">
defineEmits<{ retry: [] }>()
</script>
