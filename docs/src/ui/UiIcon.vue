<!--
  Inline SVG icon drawn from the internal icon set; sized in em so it follows the text size and
  coloured with currentColor. Decorative: hidden from assistive technology.
  Props: name (an icon from icons.ts).
-->
<template>
  <svg
    class="inline-block shrink-0 align-[-0.125em]"
    width="1em"
    height="1em"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="2"
    stroke-linecap="round"
    stroke-linejoin="round"
    :data-icon="name"
    aria-hidden="true"
    focusable="false"
  >
    <path v-for="(d, i) in ICONS[name]" :key="i" :d="d" />
  </svg>
</template>

<script setup lang="ts">
import { ICONS, type IconName } from './icons';

defineProps<{ name: IconName }>();
</script>
