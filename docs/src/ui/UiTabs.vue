<!--
  Accessible tab set: a tablist with roving focus (Left/Right/Home/End) and one panel per tab.
  Every panel stays in the DOM (hidden when inactive) so pre-rendered pages keep all their content.
  Props: tabs (tab labels), label (accessible name of the tablist).
  Slots: panel-0, panel-1, ... (content of each tab).
-->
<template>
  <div>
    <div
      role="tablist"
      :aria-label="label"
      class="flex border-b border-white/10"
      @keydown="onKeydown"
    >
      <button
        v-for="(tab, i) in tabs"
        :id="`${uid}-tab-${i}`"
        :key="tab"
        ref="tabEls"
        type="button"
        role="tab"
        :aria-selected="active === i"
        :aria-controls="`${uid}-panel-${i}`"
        :tabindex="active === i ? 0 : -1"
        class="px-4 py-2 text-sm border-b-2 transition-colors"
        :class="
          active === i
            ? 'text-violet-400 border-violet-500'
            : 'text-slate-400 border-transparent hover:text-slate-200'
        "
        @click="active = i"
      >
        {{ tab }}
      </button>
    </div>
    <div
      v-for="(tab, i) in tabs"
      :id="`${uid}-panel-${i}`"
      :key="tab"
      role="tabpanel"
      :aria-labelledby="`${uid}-tab-${i}`"
      :hidden="active !== i"
      class="py-4"
    >
      <slot :name="`panel-${i}`" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref, useId } from 'vue';

const props = defineProps<{ tabs: string[]; label?: string }>();

const uid = useId();
const active = ref(0);
const tabEls = ref<HTMLElement[]>([]);

/** Moves the selection with the arrow keys, Home and End, and focuses the new tab. */
async function onKeydown(e: KeyboardEvent) {
  const last = props.tabs.length - 1;
  const next = { ArrowRight: active.value + 1, ArrowLeft: active.value - 1, Home: 0, End: last }[
    e.key
  ];
  if (next === undefined) return;
  e.preventDefault();
  active.value = (next + last + 1) % (last + 1);
  await nextTick();
  tabEls.value[active.value]!.focus();
}
</script>
