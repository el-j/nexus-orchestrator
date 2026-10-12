<!--
  Native <select> that binds the typed `value` of an option list (numbers stay numbers).
  Props: modelValue (selected value; v-model), options ({ label, value }[]), label (accessible
  name), placeholder (shown while nothing matches), disabled, invalid.
  Emits: update:modelValue(value).
-->
<template>
  <select
    class="ui-field"
    :aria-label="label"
    :aria-invalid="invalid ? 'true' : undefined"
    :disabled="disabled"
    :value="selectedIndex"
    @change="onChange"
  >
    <option v-if="selectedIndex === ''" value="" disabled hidden>{{ placeholder }}</option>
    <option v-for="(o, i) in options" :key="i" :value="String(i)">{{ o.label }}</option>
  </select>
</template>

<script setup lang="ts">
import { computed } from 'vue';

/** One entry of a {@link UiSelect}. */
export interface SelectOption<T extends string | number = string | number> {
  label: string;
  value: T;
}

const props = defineProps<{
  modelValue: string | number;
  options: SelectOption[];
  label: string;
  placeholder?: string;
  disabled?: boolean;
  invalid?: boolean;
}>();
const emit = defineEmits<{ 'update:modelValue': [value: string | number] }>();

// Options are bound by index so the emitted value keeps its type.
const selectedIndex = computed(() => {
  const i = props.options.findIndex((o) => o.value === props.modelValue);
  return i < 0 ? '' : String(i);
});

function onChange(e: Event) {
  emit('update:modelValue', props.options[Number((e.target as HTMLSelectElement).value)]!.value);
}
</script>
