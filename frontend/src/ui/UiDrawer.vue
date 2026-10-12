<!--
  Modal right-hand drawer with a header slot, a scrollable body, a backdrop, Escape-to-close and a
  focus trap that returns focus to the opener when the drawer closes.
  Props: modelValue (open state; v-model), label (accessible name of the dialog).
  Emits: update:modelValue(open).
  Slots: header (title row), default (body).
-->
<template>
  <Teleport to="body">
    <Transition name="ui-drawer">
      <div v-if="modelValue" class="fixed inset-0 z-50 flex justify-end" @keydown="onKeydown">
        <div class="absolute inset-0 bg-black/50 backdrop-blur-sm" @click="close"></div>
        <aside
          ref="panel"
          role="dialog"
          aria-modal="true"
          :aria-label="label"
          tabindex="-1"
          class="relative flex flex-col w-full sm:w-[480px] h-full bg-[#0d0d14] border-l border-white/10 shadow-2xl overflow-hidden focus:outline-none"
        >
          <header
            class="flex items-center justify-between gap-3 px-4 py-3 border-b border-white/10 shrink-0"
          >
            <slot name="header" />
            <button
              type="button"
              class="text-slate-500 hover:text-white transition-colors"
              aria-label="Close"
              @click="close"
            >
              <UiIcon name="times" class="text-sm" />
            </button>
          </header>
          <div class="flex-1 overflow-auto p-4">
            <slot />
          </div>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue';
import UiIcon from './UiIcon.vue';

const props = defineProps<{ modelValue: boolean; label?: string }>();
const emit = defineEmits<{ 'update:modelValue': [open: boolean] }>();

const panel = ref<HTMLElement | null>(null);
let opener: HTMLElement | null = null;

const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

function close() {
  emit('update:modelValue', false);
}

/** Closes on Escape and keeps Tab / Shift+Tab inside the panel. */
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.stopPropagation();
    close();
    return;
  }
  if (e.key !== 'Tab') return;
  const items = [...panel.value!.querySelectorAll<HTMLElement>(FOCUSABLE)];
  const first = items[0] ?? panel.value!;
  const last = items[items.length - 1] ?? panel.value!;
  const active = document.activeElement;
  if (e.shiftKey && (active === first || active === panel.value)) {
    e.preventDefault();
    last.focus();
  } else if (!e.shiftKey && active === last) {
    e.preventDefault();
    first.focus();
  }
}

watch(
  () => props.modelValue,
  async (open) => {
    if (open) {
      opener = document.activeElement as HTMLElement | null;
      await nextTick();
      panel.value?.focus();
    } else {
      opener?.focus();
      opener = null;
    }
  },
);
</script>

<style scoped>
.ui-drawer-enter-active,
.ui-drawer-leave-active {
  transition: opacity 0.2s ease;
}
.ui-drawer-enter-active aside,
.ui-drawer-leave-active aside {
  transition: transform 0.25s ease;
}
.ui-drawer-enter-from,
.ui-drawer-leave-to {
  opacity: 0;
}
.ui-drawer-enter-from aside,
.ui-drawer-leave-to aside {
  transform: translateX(100%);
}
@media (prefers-reduced-motion: reduce) {
  .ui-drawer-enter-active,
  .ui-drawer-leave-active,
  .ui-drawer-enter-active aside,
  .ui-drawer-leave-active aside {
    transition: none;
  }
}
</style>
