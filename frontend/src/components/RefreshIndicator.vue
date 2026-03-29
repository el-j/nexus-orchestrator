<template>
  <span class="text-[10px] text-slate-600 tabular-nums">
    {{ label }}
  </span>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue';

const props = defineProps<{ lastRefreshed: Date | null }>();

const now = ref(Date.now());
let timer: ReturnType<typeof setInterval> | null = null;

onMounted(() => {
  timer = setInterval(() => {
    now.value = Date.now();
  }, 5000);
});
onUnmounted(() => {
  if (timer) clearInterval(timer);
});

const label = computed(() => {
  if (!props.lastRefreshed) return '';
  const diff = now.value - props.lastRefreshed.getTime();
  if (diff < 5000) return 'just now';
  if (diff < 60000) return `${Math.floor(diff / 1000)}s ago`;
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  return `${Math.floor(diff / 3600000)}h ago`;
});
</script>
