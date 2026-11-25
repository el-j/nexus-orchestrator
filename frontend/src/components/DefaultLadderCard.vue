<!--
  DefaultLadderCard: edits the daemon-wide global default model ladder stored in RuntimeConfig.
  Applies automatically to every project that does not define its own ladder in .nexus/policy.json.
-->
<template>
  <section
    class="rounded-xl border border-white/5 bg-white/[0.02] p-4 space-y-3"
    data-testid="default-ladder-card"
  >
    <div>
      <h2 class="text-xs font-semibold text-slate-400 uppercase tracking-wider">
        Global default model ladder
      </h2>
      <p class="text-[11px] text-slate-500">
        Fallback provider priority across all projects when a project has no per-project ladder in
        <code class="font-mono">.nexus/policy.json</code>.
      </p>
    </div>

    <p v-if="error" class="text-xs text-amber-400 font-mono" data-testid="default-ladder-error">
      {{ error }}
    </p>

    <LadderEditor
      v-model="rows"
      :suggestions="suggestions"
      title="Default provider order per role"
      add-button-test-id="default-ladder-add"
      row-test-id="default-ladder-row"
    />

    <div class="flex items-center gap-3">
      <button
        data-testid="default-ladder-save"
        :disabled="saving"
        class="text-xs px-3 py-1 rounded border border-violet-500/30 text-violet-300 hover:bg-violet-500/10 disabled:opacity-50"
        @click="save"
      >
        Save global ladder
      </button>
      <span v-if="saved" class="text-xs text-emerald-400" data-testid="default-ladder-saved"
        >Saved</span
      >
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import LadderEditor, { type LadderRoleRow } from './LadderEditor.vue';
import { getRuntimeConfig, updateRuntimeConfig } from '../types/wails';

withDefaults(
  defineProps<{
    suggestions?: string[];
  }>(),
  {
    suggestions: () => [],
  },
);

const rows = ref<LadderRoleRow[]>([]);
const saving = ref(false);
const saved = ref(false);
const error = ref<string | null>(null);

function applyLadder(ladder?: Record<string, string[]>) {
  rows.value = Object.entries(ladder ?? {}).map(([role, rungs]) => ({
    role,
    rungs: [...rungs],
  }));
}

async function load() {
  error.value = null;
  try {
    const cfg = await getRuntimeConfig();
    applyLadder(cfg.defaultLadder);
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load global ladder';
  }
}

async function save() {
  saving.value = true;
  saved.value = false;
  error.value = null;
  try {
    const defaultLadder: Record<string, string[]> = {};
    for (const row of rows.value) {
      const role = row.role.trim();
      const rungs = row.rungs.map((r) => r.trim()).filter(Boolean);
      if (role && rungs.length > 0) defaultLadder[role] = rungs;
    }
    const updated = await updateRuntimeConfig({ defaultLadder });
    applyLadder(updated.defaultLadder);
    saved.value = true;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to save global ladder';
  } finally {
    saving.value = false;
  }
}

onMounted(() => void load());
</script>
