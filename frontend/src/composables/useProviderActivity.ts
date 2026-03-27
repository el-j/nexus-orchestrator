import { ref, onMounted, onUnmounted } from 'vue';
import { resolveServerUrl } from './useServerUrl';

export interface ProviderModelState {
  provider: string; // "lm-studio" | "ollama" | "antigravity"
  models: string[]; // loaded model IDs
  lastSeen: string; // ISO timestamp
  active: boolean; // true if last seen < 30s ago
}

export function useProviderActivity() {
  const providerStates = ref<ProviderModelState[]>([]);
  let interval: ReturnType<typeof setInterval> | null = null;

  async function refresh() {
    try {
      const baseUrl = await resolveServerUrl();
      const url = `${baseUrl}/api/activities?type=generation&limit=50`;
      const res = await fetch(url);
      if (!res.ok) return;
      const activities = (await res.json()) as Array<{
        agentName: string;
        model?: string;
        timestamp: string;
        metadata?: Record<string, string>;
      }>;

      // Group by agentName (which is the provider name)
      const byProvider = new Map<string, { models: Set<string>; lastSeen: string }>();
      const thirtySecsAgo = new Date(Date.now() - 30_000).toISOString();

      for (const a of activities) {
        if (!byProvider.has(a.agentName)) {
          byProvider.set(a.agentName, { models: new Set(), lastSeen: a.timestamp });
        }
        const entry = byProvider.get(a.agentName)!;
        if (a.model) entry.models.add(a.model);
        if (a.timestamp > entry.lastSeen) entry.lastSeen = a.timestamp;
      }

      providerStates.value = [...byProvider.entries()].map(([provider, data]) => ({
        provider,
        models: [...data.models],
        lastSeen: data.lastSeen,
        active: data.lastSeen >= thirtySecsAgo,
      }));
    } catch {
      // silently fail
    }
  }

  onMounted(() => {
    void refresh();
    interval = setInterval(() => void refresh(), 15_000);
  });
  onUnmounted(() => {
    if (interval) clearInterval(interval);
  });

  return { providerStates, refresh };
}
