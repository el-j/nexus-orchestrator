<!--
  Button with an optional icon, a loading state and primary/secondary/danger tones.
  Props: label, icon (icon name), severity (primary | secondary | danger), size (small | normal),
  outlined, text (borderless), rounded, loading (shows a spinner and disables), disabled,
  type (button | submit, default button). A slot replaces the label.
-->
<template>
  <button
    :type="type ?? 'button'"
    :disabled="disabled || loading"
    :aria-busy="loading ? 'true' : undefined"
    class="inline-flex items-center justify-center gap-1.5 border font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-400 disabled:opacity-50 disabled:cursor-not-allowed"
    :class="[sizeClass, shapeClass, toneClass]"
  >
    <UiIcon v-if="loading" name="spinner" class="animate-spin" />
    <UiIcon v-else-if="icon" :name="icon" />
    <slot>{{ label }}</slot>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import UiIcon from './UiIcon.vue';
import type { IconName } from './icons';

const props = defineProps<{
  label?: string;
  icon?: IconName;
  severity?: 'primary' | 'secondary' | 'danger';
  size?: 'small' | 'normal';
  outlined?: boolean;
  text?: boolean;
  rounded?: boolean;
  loading?: boolean;
  disabled?: boolean;
  type?: 'button' | 'submit';
}>();

const TONES = {
  primary: {
    solid: 'bg-indigo-600 border-indigo-500 text-white hover:bg-indigo-500 active:bg-indigo-700',
    outlined: 'border-indigo-500/50 text-indigo-300 hover:bg-indigo-500/10',
    text: 'text-indigo-300 hover:bg-indigo-500/10',
  },
  secondary: {
    solid: 'bg-white/10 border-white/10 text-slate-200 hover:bg-white/15',
    outlined: 'border-white/15 text-slate-300 hover:bg-white/5',
    text: 'text-slate-400 hover:text-white hover:bg-white/5',
  },
  danger: {
    solid: 'bg-red-600 border-red-500 text-white hover:bg-red-500',
    outlined: 'border-red-500/40 text-red-400 hover:bg-red-500/10',
    text: 'text-red-400 hover:bg-red-500/10',
  },
} as const;

const sizeClass = computed(() =>
  props.size === 'small' ? 'text-xs px-3 py-1.5' : 'text-sm px-4 py-2',
);
const shapeClass = computed(() => (props.rounded ? 'rounded-full' : 'rounded-md'));
const toneClass = computed(() => {
  const tone = TONES[props.severity ?? 'primary'];
  if (props.text) return `border-transparent ${tone.text}`;
  return props.outlined ? tone.outlined : tone.solid;
});
</script>
