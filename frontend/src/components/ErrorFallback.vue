<template>
  <div class="error-fallback flex flex-col items-center justify-center h-screen p-8 text-center bg-[#050508]">
    <div class="w-16 h-16 rounded-2xl bg-red-500/10 border border-red-500/20 flex items-center justify-center mb-4">
      <i class="pi pi-exclamation-triangle text-2xl text-red-400"></i>
    </div>
    <p class="text-red-400 font-semibold mb-2">Something went wrong</p>
    <p v-if="error" class="text-xs text-slate-500 font-mono mb-4 max-w-md break-words">
      {{ error.message }}
    </p>
    <button
      class="px-4 py-2 text-sm rounded-lg bg-violet-600 text-white hover:bg-violet-500 transition-colors"
      @click="$emit('retry')"
    >Retry</button>
  </div>
</template>

<script setup lang="ts">
defineProps<{ error?: Error | null }>()
defineEmits<{ retry: [] }>()
</script>
