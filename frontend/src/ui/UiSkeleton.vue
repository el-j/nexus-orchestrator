<!--
  Pulsing placeholder block shown while content loads.
  Props: height (any CSS length, default 1rem).
-->
<template>
  <div
    class="ui-skeleton rounded bg-white/5 animate-pulse"
    :style="{ height: height ?? '1rem' }"
    aria-hidden="true"
  ></div>
</template>

<script setup lang="ts">
defineProps<{ height?: string }>();
</script>
