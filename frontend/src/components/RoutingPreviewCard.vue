<!--
  RoutingPreviewCard: displays the effective routing order (GET /api/routing) for a project and role,
  including the configured ladder rungs, quota threshold, cooldown reasons, and quota utilization.
-->
<template>
  <section class="rounded-xl border border-white/5 bg-white/[0.02] p-4 space-y-3">
    <div class="flex items-center justify-between gap-3 flex-wrap">
      <div>
        <h2 class="text-xs font-semibold text-slate-400 uppercase tracking-wider">
          Routing Preview
        </h2>
        <p class="text-[11px] text-slate-500">
          Effective provider order with role ladder, cooldowns, and quota threshold ({{
            Math.round((preview?.quotaThreshold ?? 0.9) * 100)
          }}%)
        </p>
      </div>
      <div class="flex items-center gap-2">
        <input
          v-model="projectInput"
          data-testid="routing-project-input"
          type="text"
          placeholder="Project path…"
          class="w-44 px-2 py-1 text-xs font-mono text-white bg-black/30 border border-white/10 rounded-lg focus:outline-none focus:border-violet-500/50"
        />
        <input
          v-model="roleInput"
          data-testid="routing-role-input"
          type="text"
          placeholder="Role (e.g. coder)"
          class="w-32 px-2 py-1 text-xs font-mono text-white bg-black/30 border border-white/10 rounded-lg focus:outline-none focus:border-violet-500/50"
        />
        <button
          data-testid="routing-preview-btn"
          class="text-[10px] px-2.5 py-1 rounded border border-violet-500/30 text-violet-400 hover:bg-violet-500/10 transition-colors"
          @click="loadPreview"
        >
          Preview
        </button>
      </div>
    </div>

    <div v-if="error" class="text-xs text-amber-400 font-mono">
      {{ error }}
    </div>

    <template v-else-if="preview">
      <div v-if="preview.ladder?.length" class="flex items-center gap-1.5 flex-wrap text-xs">
        <span class="text-slate-500">Ladder:</span>
        <span
          v-for="(rung, idx) in preview.ladder"
          :key="rung"
          class="px-2 py-0.5 rounded bg-violet-500/10 border border-violet-500/20 text-violet-300 font-mono text-[11px]"
        >
          {{ idx + 1 }}. {{ rung }}
        </span>
      </div>

      <div v-if="preview.order.length > 0" class="space-y-1.5">
        <div
          v-for="entry in preview.order"
          :key="entry.position"
          class="flex items-center justify-between gap-2 px-3 py-1.5 rounded-lg border border-white/5 bg-black/20 text-xs"
        >
          <div class="flex items-center gap-2 min-w-0">
            <span class="font-mono text-slate-500 w-4">{{ entry.position }}.</span>
            <span class="font-medium text-white truncate">{{ entry.provider }}</span>
            <span v-if="entry.model" class="font-mono text-slate-400 text-[11px] truncate">
              {{ entry.model }}
            </span>
            <span class="text-[10px] px-1.5 py-0.5 rounded bg-white/5 text-slate-400 font-mono">
              {{ entry.rung >= 0 ? `rung ${entry.rung + 1}` : 'unlisted' }}
            </span>
          </div>
          <div class="flex items-center gap-2 shrink-0">
            <span
              v-if="entry.cooldownReason"
              class="text-[10px] px-1.5 py-0.5 rounded bg-amber-500/20 text-amber-300"
            >
              {{ entry.cooldownReason }}
            </span>
            <span
              v-if="entry.utilization !== undefined"
              class="text-[10px] px-1.5 py-0.5 rounded font-mono"
              :class="
                entry.utilization >= preview.quotaThreshold
                  ? 'bg-red-500/20 text-red-300'
                  : 'bg-emerald-500/15 text-emerald-300'
              "
            >
              {{ Math.round(entry.utilization * 100) }}% quota
            </span>
          </div>
        </div>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue';
import type { RoutingPreview } from '../types/domain';
import { getRouting } from '../types/wails';
import { currentProject } from '../composables/useProjectState';

const projectInput = ref(currentProject.value ?? '.');
const roleInput = ref('');
const preview = ref<RoutingPreview | null>(null);
const error = ref<string | null>(null);

async function loadPreview() {
  try {
    preview.value = await getRouting(projectInput.value, roleInput.value);
    error.value = null;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load routing preview';
  }
}

watch(currentProject, (val) => {
  projectInput.value = val ?? '.';
  void loadPreview();
});

onMounted(loadPreview);
</script>
