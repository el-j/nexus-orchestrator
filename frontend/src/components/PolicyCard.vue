<!--
  PolicyCard: edits the routing and token settings of the current project's .nexus/policy.json: the
  plan-allowance threshold, the context and history token caps, the daily token budget per provider and
  the model ladder per role. Other policy fields (rules, verify, ...) are kept as they are. No props.
-->
<template>
  <section
    class="rounded-xl border border-white/5 bg-white/[0.02] p-4 space-y-3"
    data-testid="policy-card"
  >
    <div>
      <h2 class="text-xs font-semibold text-slate-400 uppercase tracking-wider">Project policy</h2>
      <p class="text-[11px] text-slate-500">
        Budgets, model ladder and token caps of <code class="font-mono">.nexus/policy.json</code>; a
        change applies to the next task.
      </p>
    </div>

    <p v-if="!currentProject" class="text-xs text-slate-500" data-testid="policy-no-project">
      Pick a project in the sidebar to edit its policy.
    </p>
    <template v-else>
      <p v-if="error" class="text-xs text-amber-400 font-mono" data-testid="policy-error">
        {{ error }}
      </p>
      <p v-if="!found && !error" class="text-xs text-slate-500">
        This project has no policy file yet; saving creates it.
      </p>

      <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs">
        <label class="flex flex-col gap-1 text-slate-400">
          Quota threshold (%)
          <input
            v-model.number="quotaPercent"
            data-testid="policy-quota"
            type="number"
            min="0"
            max="100"
            class="px-2 py-1 font-mono text-white bg-black/30 border border-white/10 rounded-lg"
          />
        </label>
        <label class="flex flex-col gap-1 text-slate-400">
          Context budget (tokens)
          <input
            v-model.number="contextBudget"
            data-testid="policy-context"
            type="number"
            min="0"
            class="px-2 py-1 font-mono text-white bg-black/30 border border-white/10 rounded-lg"
          />
        </label>
        <label class="flex flex-col gap-1 text-slate-400">
          History budget (tokens)
          <input
            v-model.number="historyBudget"
            data-testid="policy-history"
            type="number"
            min="0"
            class="px-2 py-1 font-mono text-white bg-black/30 border border-white/10 rounded-lg"
          />
        </label>
      </div>

      <div class="space-y-1.5">
        <div class="flex items-center justify-between">
          <h3 class="text-[11px] font-semibold text-slate-400">Daily token budgets</h3>
          <button
            data-testid="policy-add-budget"
            class="text-[10px] text-violet-400"
            @click="budgetRows.push({ name: '', tokens: 0 })"
          >
            + Add
          </button>
        </div>
        <div
          v-for="(row, i) in budgetRows"
          :key="'b' + i"
          class="flex items-center gap-2"
          data-testid="policy-budget-row"
        >
          <input
            v-model="row.name"
            placeholder="Provider name"
            class="flex-1 px-2 py-1 text-xs font-mono text-white bg-black/30 border border-white/10 rounded-lg"
          />
          <input
            v-model.number="row.tokens"
            type="number"
            min="0"
            placeholder="Tokens per day"
            class="w-32 px-2 py-1 text-xs font-mono text-white bg-black/30 border border-white/10 rounded-lg"
          />
          <button class="text-[10px] text-red-400" @click="budgetRows.splice(i, 1)">Remove</button>
        </div>
      </div>

      <div class="space-y-1.5">
        <div class="flex items-center justify-between">
          <h3 class="text-[11px] font-semibold text-slate-400">
            Model ladder (cheapest first, comma separated)
          </h3>
          <button
            data-testid="policy-add-ladder"
            class="text-[10px] text-violet-400"
            @click="ladderRows.push({ role: '', patterns: '' })"
          >
            + Add
          </button>
        </div>
        <div
          v-for="(row, i) in ladderRows"
          :key="'l' + i"
          class="flex items-center gap-2"
          data-testid="policy-ladder-row"
        >
          <input
            v-model="row.role"
            placeholder="Role (or default)"
            class="w-32 px-2 py-1 text-xs font-mono text-white bg-black/30 border border-white/10 rounded-lg"
          />
          <input
            v-model="row.patterns"
            placeholder="Ollama/*, Anthropic/claude-sonnet*"
            class="flex-1 px-2 py-1 text-xs font-mono text-white bg-black/30 border border-white/10 rounded-lg"
          />
          <button class="text-[10px] text-red-400" @click="ladderRows.splice(i, 1)">Remove</button>
        </div>
      </div>

      <div class="flex items-center gap-3">
        <button
          data-testid="policy-save"
          :disabled="saving"
          class="text-xs px-3 py-1 rounded border border-violet-500/30 text-violet-300 hover:bg-violet-500/10 disabled:opacity-50"
          @click="save"
        >
          Save policy
        </button>
        <span v-if="saved" class="text-xs text-emerald-400" data-testid="policy-saved">Saved</span>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import type { ProjectPolicy } from '../types/domain';
import { getPolicy, setPolicy } from '../types/wails';
import { currentProject } from '../composables/useProjectState';

const found = ref(false);
const error = ref<string | null>(null);
const saving = ref(false);
const saved = ref(false);
const loaded = ref<ProjectPolicy>({});
const quotaPercent = ref(0);
const contextBudget = ref(0);
const historyBudget = ref(0);
const budgetRows = ref<{ name: string; tokens: number }[]>([]);
const ladderRows = ref<{ role: string; patterns: string }[]>([]);

function show(policy: ProjectPolicy) {
  loaded.value = policy;
  quotaPercent.value = Math.round((policy.quotaThreshold ?? 0) * 100);
  contextBudget.value = policy.contextBudget ?? 0;
  historyBudget.value = policy.historyBudget ?? 0;
  budgetRows.value = Object.entries(policy.budgets ?? {}).map(([name, b]) => ({
    name,
    tokens: b.dailyTokens,
  }));
  ladderRows.value = Object.entries(policy.ladder ?? {}).map(([role, rungs]) => ({
    role,
    patterns: rungs.join(', '),
  }));
}

async function load(project: string | null) {
  saved.value = false;
  error.value = null;
  if (!project) return;
  try {
    const reply = await getPolicy(project);
    found.value = reply.found;
    show(reply.policy);
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load the policy';
  }
}

/** Builds the policy to save: the loaded one with the edited routing and token fields replaced. */
function edited(): ProjectPolicy {
  const next: ProjectPolicy = { ...loaded.value };
  const set = <K extends keyof ProjectPolicy>(key: K, value: ProjectPolicy[K] | undefined) => {
    if (value === undefined) delete next[key];
    else next[key] = value;
  };
  set('quotaThreshold', quotaPercent.value > 0 ? quotaPercent.value / 100 : undefined);
  set('contextBudget', contextBudget.value > 0 ? contextBudget.value : undefined);
  set('historyBudget', historyBudget.value > 0 ? historyBudget.value : undefined);
  const budgets: NonNullable<ProjectPolicy['budgets']> = {};
  for (const row of budgetRows.value) {
    if (row.name.trim() && row.tokens > 0) budgets[row.name.trim()] = { dailyTokens: row.tokens };
  }
  set('budgets', Object.keys(budgets).length > 0 ? budgets : undefined);
  const ladder: NonNullable<ProjectPolicy['ladder']> = {};
  for (const row of ladderRows.value) {
    const rungs = row.patterns
      .split(',')
      .map((p) => p.trim())
      .filter(Boolean);
    if (row.role.trim() && rungs.length > 0) ladder[row.role.trim()] = rungs;
  }
  set('ladder', Object.keys(ladder).length > 0 ? ladder : undefined);
  return next;
}

async function save() {
  if (!currentProject.value) return;
  saving.value = true;
  saved.value = false;
  try {
    const reply = await setPolicy(currentProject.value, edited());
    found.value = true;
    show(reply.policy);
    error.value = null;
    saved.value = true;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to save the policy';
  } finally {
    saving.value = false;
  }
}

watch(currentProject, (p) => void load(p), { immediate: true });
</script>
