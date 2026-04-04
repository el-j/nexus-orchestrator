<template>
  <Teleport to="body">
    <Transition name="drawer">
      <div v-if="modelVisible" class="fixed inset-0 z-50 flex justify-end">
        <!-- Backdrop -->
        <div class="absolute inset-0 bg-black/50 backdrop-blur-sm" @click="close"></div>

        <!-- Drawer panel -->
        <aside
          class="relative flex flex-col w-80 md:w-96 h-full bg-[#0d0d18] border-l border-white/10 shadow-2xl overflow-hidden"
        >
          <!-- Header -->
          <header
            class="flex items-start justify-between px-4 py-3 border-b border-white/10 shrink-0"
          >
            <div>
              <h2 class="text-sm font-bold text-white">{{ agentName }}</h2>
              <span
                class="inline-block mt-1 text-[10px] px-2 py-0.5 rounded-full font-medium"
                :class="statusClass"
                >{{ statusLabel }}</span
              >
            </div>
            <button
              class="text-slate-500 hover:text-white ml-2 mt-0.5 transition-colors"
              @click="close"
              aria-label="Close"
            >
              <i class="pi pi-times text-sm"></i>
            </button>
          </header>

          <!-- Scrollable content -->
          <div class="flex-1 overflow-auto p-4 space-y-5">
            <!-- Loading state -->
            <div v-if="loading" class="flex justify-center py-8">
              <div
                class="w-5 h-5 border-2 border-violet-500 border-t-transparent rounded-full animate-spin"
              ></div>
            </div>

            <template v-else>
              <!-- Current activity -->
              <section>
                <h3 class="text-[10px] font-semibold text-slate-500 uppercase tracking-wider mb-2">
                  Current Activity
                </h3>
                <div class="rounded-lg bg-white/[0.03] border border-white/[0.06] px-3 py-2.5">
                  <p class="text-xs text-slate-300">
                    {{ activities[0]?.summary ?? 'No recent activity' }}
                  </p>
                  <p v-if="activities[0]" class="text-[10px] text-slate-600 mt-1">
                    {{ timeAgo(activities[0].timestamp) }}
                  </p>
                </div>
              </section>

              <!-- Stats row -->
              <section>
                <h3 class="text-[10px] font-semibold text-slate-500 uppercase tracking-wider mb-2">
                  Stats
                </h3>
                <div class="grid grid-cols-3 gap-2">
                  <div
                    class="rounded-lg bg-white/[0.03] border border-white/[0.06] px-2 py-2 text-center"
                  >
                    <p class="text-base font-bold text-violet-400">{{ activities.length }}</p>
                    <p class="text-[10px] text-slate-500 mt-0.5">Events</p>
                  </div>
                  <div
                    class="rounded-lg bg-white/[0.03] border border-white/[0.06] px-2 py-2 text-center"
                  >
                    <p class="text-base font-bold text-violet-400">{{ totalTokens }}</p>
                    <p class="text-[10px] text-slate-500 mt-0.5">Tokens</p>
                  </div>
                  <div
                    class="rounded-lg bg-white/[0.03] border border-white/[0.06] px-2 py-2 text-center"
                  >
                    <p class="text-[11px] font-medium text-slate-300 leading-tight">
                      {{ activities[0] ? timeAgo(activities[0].timestamp) : '—' }}
                    </p>
                    <p class="text-[10px] text-slate-500 mt-0.5">Last active</p>
                  </div>
                </div>
              </section>

              <!-- Mini timeline -->
              <section>
                <h3 class="text-[10px] font-semibold text-slate-500 uppercase tracking-wider mb-2">
                  Recent Activity
                </h3>
                <div v-if="activities.length === 0" class="text-xs text-slate-600 italic py-2">
                  No activity recorded.
                </div>
                <div v-else class="space-y-1">
                  <div
                    v-for="a in activities.slice(0, 10)"
                    :key="a.id"
                    class="flex items-start gap-2 px-2 py-1.5 rounded-lg hover:bg-white/[0.04] transition-colors"
                  >
                    <span class="shrink-0 text-sm leading-none mt-0.5">{{
                      activityEmoji(a.activityType)
                    }}</span>
                    <p class="flex-1 text-[11px] text-slate-300 truncate">{{ a.summary }}</p>
                    <span class="shrink-0 text-[10px] text-slate-600">{{
                      timeAgo(a.timestamp)
                    }}</span>
                  </div>
                </div>
              </section>
            </template>
          </div>

          <!-- Footer -->
          <footer class="px-4 py-3 border-t border-white/10 shrink-0">
            <button
              class="text-xs text-violet-400 hover:text-violet-300 transition-colors"
              @click="viewInTimeline"
            >
              View in timeline →
            </button>
          </footer>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import type { AIActivity } from '../types/domain';
import { useActivities } from '../composables/useActivities';
import { timeAgo } from '../utils/time';

const props = defineProps<{
  agentName: string;
  modelVisible: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:modelVisible', val: boolean): void;
}>();

const router = useRouter();

const { activities, loading } = useActivities({
  agentFilter: computed(() => props.agentName).value,
  limit: 10,
});

const totalTokens = computed(() =>
  activities.value.reduce((sum, a) => sum + (a.tokensIn ?? 0) + (a.tokensOut ?? 0), 0),
);

const statusLabel = computed(() => {
  if (activities.value.length === 0) return 'disconnected';
  const last = activities.value[0];
  const msSince = Date.now() - new Date(last.timestamp).getTime();
  if (msSince < 2 * 60 * 1000) return 'active';
  if (msSince < 10 * 60 * 1000) return 'idle';
  return 'disconnected';
});

const statusClass = computed((): string => {
  switch (statusLabel.value) {
    case 'active':
      return 'bg-emerald-500/20 text-emerald-300';
    case 'idle':
      return 'bg-yellow-500/20 text-yellow-300';
    default:
      return 'bg-slate-700/50 text-slate-400';
  }
});

function activityEmoji(type: AIActivity['activityType']): string {
  const map: Record<string, string> = {
    message: '💬',
    tool_use: '🔧',
    thinking: '🧠',
    file_edit: '📝',
    generation: '✨',
  };
  return map[type] ?? '•';
}

function close() {
  emit('update:modelVisible', false);
}

function viewInTimeline() {
  close();
  router.push({ name: 'live-activity', query: { agent: props.agentName } });
}
</script>

<style scoped>
.drawer-enter-active,
.drawer-leave-active {
  transition: opacity 0.2s ease;
}
.drawer-enter-active aside,
.drawer-leave-active aside {
  transition: transform 0.25s ease;
}
.drawer-enter-from,
.drawer-leave-to {
  opacity: 0;
}
.drawer-enter-from aside {
  transform: translateX(100%);
}
.drawer-leave-to aside {
  transform: translateX(100%);
}
</style>
