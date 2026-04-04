<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <header class="px-5 py-3 border-b border-white/5 bg-[#0a0a10] shrink-0">
      <h1 class="text-sm font-bold text-white">Projects</h1>
      <p class="text-xs text-slate-500">AI activity grouped by project</p>
    </header>

    <div class="flex-1 overflow-auto p-5">
      <!-- Loading state -->
      <div v-if="loading" class="flex items-center justify-center py-16">
        <div
          class="w-6 h-6 border-2 border-violet-500 border-t-transparent rounded-full animate-spin"
        ></div>
      </div>

      <!-- Error state -->
      <div
        v-else-if="error"
        class="text-sm text-red-400 bg-red-500/10 border border-red-500/20 rounded-xl p-4"
      >
        {{ error }}
      </div>

      <!-- Empty state -->
      <div
        v-else-if="projects.length === 0"
        class="flex flex-col items-center justify-center py-20 text-center"
      >
        <div class="text-4xl mb-4 opacity-30">📁</div>
        <p class="text-sm font-medium text-slate-400">No project activity yet</p>
        <p class="text-xs text-slate-600 mt-1">
          Activity will appear here once AI agents start working on projects.
        </p>
      </div>

      <!-- Project cards grid -->
      <div v-else class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div
          v-for="project in projects"
          :key="project.path"
          class="rounded-xl border border-white/[0.07] bg-[#0d0d18] overflow-hidden hover:border-violet-500/20 transition-colors"
        >
          <!-- Card header -->
          <div
            class="px-4 py-3 border-b border-white/[0.05] flex items-start justify-between gap-2"
          >
            <div class="min-w-0">
              <p class="text-xs font-mono font-semibold text-slate-400 truncate">
                {{ shortPath(project.path) }}
              </p>
              <p class="text-[10px] text-slate-600 truncate mt-0.5">{{ project.path }}</p>
            </div>
            <span
              class="shrink-0 text-[10px] px-2 py-0.5 rounded-full bg-violet-500/15 text-violet-300 font-medium"
            >
              {{ project.agents.length }} agent{{ project.agents.length !== 1 ? 's' : '' }}
            </span>
          </div>

          <!-- Agents row -->
          <div class="px-4 py-2 flex flex-wrap gap-1.5 border-b border-white/[0.04]">
            <span
              v-for="agent in project.agents"
              :key="agent"
              class="text-[10px] px-2 py-0.5 rounded-full bg-white/[0.05] text-slate-300 border border-white/[0.06]"
              >{{ agent }}</span
            >
          </div>

          <!-- Mini activity feed -->
          <div class="px-4 py-2 space-y-1">
            <div v-for="a in project.recentActivities" :key="a.id" class="flex items-center gap-2">
              <span class="shrink-0 text-xs">{{ activityEmoji(a.activityType) }}</span>
              <span class="text-[10px] text-slate-500 shrink-0 truncate max-w-[4rem]">{{
                a.agentName
              }}</span>
              <p class="flex-1 text-[11px] text-slate-300 truncate">{{ a.summary }}</p>
              <span class="shrink-0 text-[10px] text-slate-600">{{ timeAgo(a.timestamp) }}</span>
            </div>
          </div>

          <!-- Card footer -->
          <div class="px-4 py-2.5 flex justify-end border-t border-white/[0.04]">
            <button
              class="text-[11px] text-violet-400 hover:text-violet-300 transition-colors"
              @click="router.push('/projects/live-activity')"
            >
              View timeline →
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { useRouter } from 'vue-router';
import type { AIActivity } from '../types/domain';
import { resolveServerUrl } from '../composables/useServerUrl';
import { timeAgo } from '../utils/time';

const router = useRouter();

interface ProjectGroup {
  path: string;
  agents: string[];
  recentActivities: AIActivity[];
  lastActivity: string;
}

const allActivities = ref<AIActivity[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
let pollInterval: ReturnType<typeof setInterval> | null = null;

async function fetchActivities() {
  try {
    const baseUrl = await resolveServerUrl();
    const since = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();
    const res = await fetch(
      `${baseUrl}/api/activities/timeline?limit=200&since=${encodeURIComponent(since)}`,
    );
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const data = (await res.json()) as AIActivity[];
    allActivities.value = data ?? [];
    error.value = null;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load activities';
  }
}

const projects = computed((): ProjectGroup[] => {
  const byPath = new Map<string, AIActivity[]>();
  for (const a of allActivities.value) {
    const key = a.projectPath ?? '(no project)';
    const existing = byPath.get(key);
    if (existing) {
      existing.push(a);
    } else {
      byPath.set(key, [a]);
    }
  }

  const groups: ProjectGroup[] = [];
  for (const [path, acts] of byPath.entries()) {
    const sorted = [...acts].sort(
      (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime(),
    );
    const agents = [...new Set(sorted.map((a) => a.agentName))];
    groups.push({
      path,
      agents,
      recentActivities: sorted.slice(0, 5),
      lastActivity: sorted[0]?.timestamp ?? '',
    });
  }

  return groups.sort(
    (a, b) => new Date(b.lastActivity).getTime() - new Date(a.lastActivity).getTime(),
  );
});

function shortPath(fullPath: string): string {
  const parts = fullPath.replace(/\\/g, '/').split('/').filter(Boolean);
  return parts.length >= 2 ? parts.slice(-2).join('/') : fullPath;
}

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

onMounted(async () => {
  loading.value = true;
  try {
    await fetchActivities();
  } finally {
    loading.value = false;
    pollInterval = setInterval(fetchActivities, 15_000);
  }
});

onUnmounted(() => {
  if (pollInterval) clearInterval(pollInterval);
});
</script>
