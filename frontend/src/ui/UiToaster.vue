<!--
  Renders the toasts raised through useToast in the bottom-right corner. Announced politely to
  screen readers (errors assertively); each toast can be dismissed with its close button.
-->
<template>
  <div class="fixed bottom-4 right-4 z-[60] flex w-80 flex-col gap-2" aria-live="polite">
    <div
      v-for="t in activeToasts"
      :key="t.id"
      :role="t.severity === 'error' ? 'alert' : 'status'"
      class="flex items-start gap-2 rounded-lg border px-3 py-2.5 shadow-xl backdrop-blur"
      :class="TONES[t.severity]"
      data-testid="toast"
    >
      <div class="min-w-0 flex-1">
        <p class="text-xs font-semibold">{{ t.summary }}</p>
        <p v-if="t.detail" class="mt-0.5 break-words text-xs opacity-80">{{ t.detail }}</p>
      </div>
      <button
        type="button"
        class="opacity-60 transition-opacity hover:opacity-100"
        aria-label="Dismiss notification"
        @click="remove(t.id)"
      >
        <UiIcon name="times" class="text-xs" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { activeToasts, useToast, type ToastSeverity } from '../composables/useToast';
import UiIcon from './UiIcon.vue';

const { remove } = useToast();

const TONES: Record<ToastSeverity, string> = {
  success: 'border-emerald-500/40 bg-emerald-950/80 text-emerald-200',
  info: 'border-sky-500/40 bg-sky-950/80 text-sky-200',
  warn: 'border-amber-500/40 bg-amber-950/80 text-amber-200',
  error: 'border-red-500/40 bg-red-950/80 text-red-200',
};
</script>
