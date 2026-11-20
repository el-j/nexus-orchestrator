<!--
  Projects: AI activity and active sessions grouped by project, polled every 15 seconds.
  Allows selecting timeframes (24h, 7d, All) and bridges active AI sessions.
  Clicking a project opens Live Activity filtered to it.
-->
<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <header
      class="px-5 py-3 border-b border-white/5 bg-[#0a0a10] shrink-0 flex items-center justify-between flex-wrap gap-2"
    >
      <div>
        <h1 class="text-sm font-bold text-white">Projects</h1>
        <p class="text-xs text-slate-500">AI activity grouped by project</p>
      </div>

      <div
        class="flex items-center rounded-lg bg-black/40 p-0.5 border border-white/5 text-[10px]"
        data-testid="activity-timeframe-selector"
      >
        <button
          v-for="tf in timeframes"
          :key="tf.value"
          type="button"
          :class="[
            'px-2 py-0.5 rounded transition font-medium',
            timeframe === tf.value
              ? 'bg-purple-600/70 text-white shadow-sm'
              : 'text-slate-400 hover:text-slate-200',
          ]"
          :data-testid="`activity-timeframe-${tf.value}`"
          @click="selectTimeframe(tf.value)"
        >
          {{ tf.label }}
        </button>
      </div>
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
        data-testid="activity-error"
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
            <div class="flex items-center gap-1.5 shrink-0">
              <span
                v-if="project.hasActiveSession"
                class="inline-flex items-center gap-1 text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-300 font-medium"
                data-testid="active-session-badge"
              >
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
                Active
              </span>
              <span
                class="text-[10px] px-2 py-0.5 rounded-full bg-violet-500/15 text-violet-300 font-medium"
              >
                {{ project.agents.length }} agent{{ project.agents.length !== 1 ? 's' : '' }}
              </span>
            </div>
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
            <template v-if="project.recentActivities.length > 0">
              <div
                v-for="a in project.recentActivities"
                :key="a.id"
                class="flex items-center gap-2"
              >
                <span class="shrink-0 text-xs">{{ activityEmoji(a.activityType) }}</span>
                <span class="text-[10px] text-slate-500 shrink-0 truncate max-w-[4rem]">{{
                  a.agentName
                }}</span>
                <p class="flex-1 text-[11px] text-slate-300 truncate">{{ a.summary }}</p>
                <span class="shrink-0 text-[10px] text-slate-600">{{ timeAgo(a.timestamp) }}</span>
              </div>
            </template>
            <div
              v-else
              class="text-[11px] text-slate-500 py-1.5 italic"
              data-testid="no-recent-events"
            >
              No logged events in {{ timeframeLabel }}.
            </div>
          </div>

          <!-- Card footer -->
          <div class="px-4 py-2.5 flex justify-end border-t border-white/[0.04]">
            <button
              class="text-[11px] text-violet-400 hover:text-violet-300 transition-colors"
              @click="openTimeline(project.path)"
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
import type { AIActivity, AISession } from '../types/domain';
import { resolveServerUrl } from '../composables/useServerUrl';
import { timeAgo } from '../utils/time';

const router = useRouter();

/** Activities without a project are grouped under this key; there is nothing to filter by. */
const NO_PROJECT = '(no project)';

const timeframes = [
  { value: '24h', label: '24h' },
  { value: '7d', label: '7d' },
  { value: 'all', label: 'All' },
];

const timeframe = ref('24h');

const timeframeLabel = computed(() => {
  if (timeframe.value === '7d') return '7d';
  if (timeframe.value === 'all') return 'All';
  return '24h';
});

function openTimeline(projectPath: string) {
  router.push({
    name: 'live-activity',
    query: projectPath === NO_PROJECT ? {} : { project: projectPath },
  });
}

interface ProjectGroup {
  path: string;
  agents: string[];
  recentActivities: AIActivity[];
  lastActivity: string;
  hasActiveSession: boolean;
}

const allActivities = ref<AIActivity[]>([]);
const activeSessions = ref<AISession[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
let pollInterval: ReturnType<typeof setInterval> | undefined;
let disposed = false;

function getSinceIso(): string {
  if (timeframe.value === '7d') {
    return new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString();
  }
  if (timeframe.value === 'all') {
    return new Date(0).toISOString();
  }
  return new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();
}

async function fetchActivities() {
  try {
    const baseUrl = await resolveServerUrl();
    const since = getSinceIso();
    const limit = timeframe.value === 'all' ? 500 : 200;

    const actPromise = fetch(
      `${baseUrl}/api/activities/timeline?limit=${limit}&since=${encodeURIComponent(since)}`,
    );
    const sessPromise = fetch(`${baseUrl}/api/ai-sessions`).catch(() => null);

    const [actRes, sessRes] = await Promise.all([actPromise, sessPromise]);

    if (!actRes.ok) throw new Error(`HTTP ${actRes.status}`);
    const data = (await actRes.json()) as AIActivity[];
    allActivities.value = data ?? [];

    if (sessRes && sessRes.ok) {
      const sessData = (await sessRes.json()) as AISession[];
      activeSessions.value = Array.isArray(sessData) ? sessData : [];
    } else {
      activeSessions.value = [];
    }

    error.value = null;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load activities';
  }
}

async function selectTimeframe(val: string) {
  timeframe.value = val;
  loading.value = true;
  await fetchActivities();
  loading.value = false;
}

const projects = computed((): ProjectGroup[] => {
  const byPath = new Map<
    string,
    { activities: AIActivity[]; sessions: AISession[]; lastTime: number }
  >();

  // 1. Group activities
  for (const a of allActivities.value) {
    const key = a.projectPath ?? NO_PROJECT;
    let group = byPath.get(key);
    if (!group) {
      group = { activities: [], sessions: [], lastTime: 0 };
      byPath.set(key, group);
    }
    group.activities.push(a);
    const t = new Date(a.timestamp).getTime();
    if (t > group.lastTime) group.lastTime = t;
  }

  // 2. Group active sessions
  for (const s of activeSessions.value) {
    const key = s.projectPath || NO_PROJECT;
    let group = byPath.get(key);
    if (!group) {
      group = { activities: [], sessions: [], lastTime: 0 };
      byPath.set(key, group);
    }
    group.sessions.push(s);
    const t = new Date(s.lastActivity || s.updatedAt || s.createdAt || 0).getTime();
    if (t > group.lastTime) group.lastTime = t;
  }

  const groups: ProjectGroup[] = [];
  for (const [path, data] of byPath.entries()) {
    const sorted = [...data.activities].sort(
      (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime(),
    );
    const actAgents = sorted.map((a) => a.agentName);
    const sessAgents = data.sessions.map((s) => s.agentName);
    const agents = [...new Set([...actAgents, ...sessAgents])];

    const hasActiveSession = data.sessions.some((s) => s.status === 'active');
    const lastActivity = new Date(data.lastTime).toISOString();

    groups.push({
      path,
      agents,
      recentActivities: sorted.slice(0, 5),
      lastActivity,
      hasActiveSession,
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
    if (!disposed) pollInterval = setInterval(fetchActivities, 15_000);
  }
});

onUnmounted(() => {
  disposed = true;
  clearInterval(pollInterval);
});
</script>
