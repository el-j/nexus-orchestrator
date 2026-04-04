<template>
  <Teleport to="body">
    <Transition name="fade">
      <div
        v-if="open"
        class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm"
        @click.self="$emit('cancel')"
      >
        <div
          class="w-full max-w-sm rounded-xl border border-white/10 bg-[#0d0d18] shadow-2xl p-5 space-y-4"
          role="dialog"
          aria-modal="true"
        >
          <!-- Title -->
          <h3 class="text-sm font-semibold text-white">{{ title }}</h3>

          <!-- Message -->
          <p class="text-xs text-slate-400">{{ message }}</p>

          <!-- Actions -->
          <div class="flex items-center justify-end gap-2 pt-1">
            <button
              class="text-xs px-3 py-1.5 rounded-lg border border-white/10 text-slate-400 hover:text-white hover:border-white/20 transition-colors"
              @click="$emit('cancel')"
            >
              {{ cancelLabel ?? 'Cancel' }}
            </button>
            <button
              class="text-xs px-3 py-1.5 rounded-lg border font-medium transition-colors"
              :class="
                danger
                  ? 'border-red-500/40 bg-red-500/10 text-red-400 hover:bg-red-500/20'
                  : 'border-violet-500/40 bg-violet-500/10 text-violet-300 hover:bg-violet-500/20'
              "
              @click="$emit('confirm')"
            >
              {{ confirmLabel ?? 'Confirm' }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { onUnmounted, watch } from 'vue';

const props = defineProps<{
  open: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
}>();

const emit = defineEmits<{
  (e: 'confirm'): void;
  (e: 'cancel'): void;
}>();

let listening = false;

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    emit('cancel');
  }
}

function setEscapeListener(active: boolean) {
  if (active && !listening) {
    window.addEventListener('keydown', onKeydown);
    listening = true;
    return;
  }
  if (!active && listening) {
    window.removeEventListener('keydown', onKeydown);
    listening = false;
  }
}

watch(
  () => props.open,
  (open) => {
    setEscapeListener(open);
  },
  { immediate: true },
);

onUnmounted(() => {
  setEscapeListener(false);
});
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
