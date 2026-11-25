<!--
  LadderEditor: visual editor for role-based model ladders. Each role contains an ordered list of
  provider/model patterns with one-click up/down priority reordering, removal, quick-add from
  detected providers, and custom wildcard entry.
-->
<template>
  <div class="space-y-2" data-testid="ladder-editor">
    <div class="flex items-center justify-between">
      <h3 class="text-[11px] font-semibold text-slate-400">
        {{ title }}
      </h3>
      <button
        :data-testid="addButtonTestId"
        class="text-[10px] text-violet-400 hover:text-violet-300"
        @click="addRole"
      >
        + Add role
      </button>
    </div>

    <div
      v-for="(row, rowIdx) in modelValue"
      :key="'role-' + rowIdx"
      :data-testid="rowTestId"
      class="rounded-lg border border-white/10 bg-black/20 p-2.5 space-y-2"
    >
      <div class="flex items-center gap-2">
        <input
          :value="row.role"
          data-testid="ladder-role-input"
          placeholder="Role (coder, architect, default…)"
          class="w-44 px-2 py-1 text-xs font-mono text-white bg-black/30 border border-white/10 rounded-lg"
          @input="updateRole(rowIdx, ($event.target as HTMLInputElement).value)"
        />
        <span class="text-[10px] text-slate-500 flex-1">
          {{ row.rungs.length }} rung{{ row.rungs.length === 1 ? '' : 's' }} (tried top → bottom)
        </span>
        <button
          data-testid="ladder-remove-role"
          class="text-[10px] text-red-400 hover:text-red-300"
          @click="removeRole(rowIdx)"
        >
          Remove
        </button>
      </div>

      <ol v-if="row.rungs.length > 0" class="space-y-1">
        <li
          v-for="(rung, rungIdx) in row.rungs"
          :key="'rung-' + rungIdx"
          data-testid="ladder-rung-item"
          class="flex items-center gap-1.5 px-2 py-1 rounded bg-white/[0.03] border border-white/5 text-xs font-mono"
        >
          <span class="text-[10px] text-slate-500 w-5">#{{ rungIdx + 1 }}</span>
          <span class="flex-1 text-slate-200 truncate">{{ rung }}</span>
          <button
            data-testid="ladder-move-up"
            :disabled="rungIdx === 0"
            title="Move up (higher priority)"
            class="px-1.5 py-0.5 text-[10px] rounded border border-white/10 text-slate-300 hover:bg-white/10 disabled:opacity-30"
            @click="moveRung(rowIdx, rungIdx, -1)"
          >
            ↑
          </button>
          <button
            data-testid="ladder-move-down"
            :disabled="rungIdx === row.rungs.length - 1"
            title="Move down (lower priority)"
            class="px-1.5 py-0.5 text-[10px] rounded border border-white/10 text-slate-300 hover:bg-white/10 disabled:opacity-30"
            @click="moveRung(rowIdx, rungIdx, 1)"
          >
            ↓
          </button>
          <button
            data-testid="ladder-remove-rung"
            title="Remove rung"
            class="px-1.5 py-0.5 text-[10px] text-red-400 hover:text-red-300"
            @click="removeRung(rowIdx, rungIdx)"
          >
            ×
          </button>
        </li>
      </ol>

      <div class="flex items-center gap-1.5 flex-wrap">
        <select
          v-if="suggestions.length > 0"
          data-testid="ladder-suggestion-select"
          class="px-2 py-1 text-[11px] font-mono text-slate-300 bg-black/30 border border-white/10 rounded-lg max-w-[220px]"
          @change="pickSuggestion(rowIdx, $event)"
        >
          <option value="">+ Pick provider…</option>
          <option v-for="s in suggestions" :key="s" :value="s">{{ s }}</option>
        </select>
        <input
          v-model="drafts[rowIdx]"
          data-testid="ladder-rung-input"
          placeholder="Custom pattern (e.g. Ollama/*, claude-sonnet*)"
          class="flex-1 min-w-[160px] px-2 py-1 text-[11px] font-mono text-white bg-black/30 border border-white/10 rounded-lg"
          @keydown.enter.prevent="commitDraft(rowIdx)"
        />
        <button
          data-testid="ladder-add-rung"
          class="text-[10px] px-2 py-1 rounded border border-violet-500/30 text-violet-300 hover:bg-violet-500/10"
          @click="commitDraft(rowIdx)"
        >
          + Rung
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive } from 'vue';

/** One role entry in a visual model ladder editor. */
export interface LadderRoleRow {
  role: string;
  rungs: string[];
}

const props = withDefaults(
  defineProps<{
    modelValue: LadderRoleRow[];
    suggestions?: string[];
    title?: string;
    addButtonTestId?: string;
    rowTestId?: string;
  }>(),
  {
    suggestions: () => [],
    title: 'Model ladder (priority order, top tried first)',
    addButtonTestId: 'ladder-add-role',
    rowTestId: 'ladder-role-row',
  },
);

const emit = defineEmits<{
  'update:modelValue': [rows: LadderRoleRow[]];
}>();

const drafts = reactive<Record<number, string>>({});

function clone(): LadderRoleRow[] {
  return props.modelValue.map((r) => ({ role: r.role, rungs: [...r.rungs] }));
}

function addRole() {
  emit('update:modelValue', [...clone(), { role: '', rungs: [] }]);
}

function removeRole(rowIdx: number) {
  const next = clone();
  next.splice(rowIdx, 1);
  delete drafts[rowIdx];
  emit('update:modelValue', next);
}

function updateRole(rowIdx: number, role: string) {
  const next = clone();
  next[rowIdx].role = role;
  emit('update:modelValue', next);
}

function appendRung(rowIdx: number, pattern: string) {
  const trimmed = pattern.trim();
  if (!trimmed) return;
  const next = clone();
  if (!next[rowIdx].rungs.includes(trimmed)) {
    next[rowIdx].rungs.push(trimmed);
    emit('update:modelValue', next);
  }
}

function commitDraft(rowIdx: number) {
  const raw = drafts[rowIdx] ?? '';
  for (const part of raw.split(',')) {
    appendRung(rowIdx, part);
  }
  drafts[rowIdx] = '';
}

function pickSuggestion(rowIdx: number, ev: Event) {
  const select = ev.target as HTMLSelectElement;
  if (select.value) {
    appendRung(rowIdx, select.value);
    select.value = '';
  }
}

function moveRung(rowIdx: number, rungIdx: number, delta: number) {
  const next = clone();
  const list = next[rowIdx].rungs;
  const target = rungIdx + delta;
  const [item] = list.splice(rungIdx, 1);
  list.splice(target, 0, item);
  emit('update:modelValue', next);
}

function removeRung(rowIdx: number, rungIdx: number) {
  const next = clone();
  next[rowIdx].rungs.splice(rungIdx, 1);
  emit('update:modelValue', next);
}
</script>
