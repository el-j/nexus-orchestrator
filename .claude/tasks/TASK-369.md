# TASK-369: Providers view — Add "AI Coding Tools" section with discovered agents

**Plan:** PLAN-054 | **Wave:** 1 | **Status:** done | **Role:** frontend

## Goal

Add a new section to ProvidersView.vue called "AI Coding Tools" that shows discovered
AI agents (GitHub Copilot, Continue, Claude CLI, etc.) from the existing endpoint.
Users currently have no way to see these from the Providers page.

## Context

- Endpoint already exists: `GET /api/ai-sessions/discovered` → `DiscoveredAgent[]`
- AIAgentsView.vue already uses this — we can reuse the same composable/fetch
- `DiscoveredAgent` fields: `id`, `kind` (copilot/continue/claude-cli/...), `name`, `detectionMethod`, `isRunning`, `lastSeen`, `cliPath`, `configPath`
- File: `frontend/src/views/ProvidersView.vue`

## Implementation

### New section below "Configured Providers":

```vue
<!-- AI CODING TOOLS -->
<section class="mt-8">
  <div class="flex items-center justify-between mb-3">
    <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider">AI Coding Tools</h2>
    <span class="text-xs text-slate-600">detected on this system</span>
  </div>
  <div v-if="agents.length === 0" class="text-xs text-slate-600 italic p-3">
    No AI coding tools detected — install GitHub Copilot, Continue, or run Claude CLI
  </div>
  <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
    <div v-for="agent in agents" :key="agent.id" class="rounded-xl border border-white/10 bg-white/[0.02] p-4">
      <div class="flex items-center justify-between mb-2">
        <span class="font-semibold text-sm text-white">{{ agentLabel(agent) }}</span>
        <span :class="agentStatusClass(agent)" class="text-[10px] px-2 py-0.5 rounded font-semibold">
          {{ agent.isRunning ? 'Active' : agent.detectionMethod === 'vscode-extension' ? 'Installed' : 'Detected' }}
        </span>
      </div>
      <p class="text-[11px] text-slate-500 font-mono truncate">{{ agent.cliPath || agent.configPath || agent.detectionMethod }}</p>
      <p class="text-[10px] text-slate-700 mt-1">via {{ agent.detectionMethod }}</p>
    </div>
  </div>
</section>
```

### Data fetching — add to existing setup():

```typescript
const agents = ref<DiscoveredAgent[]>([]);
async function loadAgents() {
  const base = await resolveServerUrl();
  const res = await fetch(`${base}/api/ai-sessions/discovered`);
  if (res.ok) agents.value = await res.json();
}
onMounted(loadAgents);

function agentLabel(a: DiscoveredAgent): string {
  const labels: Record<string, string> = {
    copilot: 'GitHub Copilot',
    continue: 'Continue',
    'claude-cli': 'Claude CLI',
    cline: 'Cline',
    cursor: 'Cursor',
  };
  return labels[a.kind] ?? a.name ?? a.kind;
}
function agentStatusClass(a: DiscoveredAgent): string {
  return a.isRunning ? 'bg-emerald-500/20 text-emerald-300' : 'bg-slate-500/20 text-slate-400';
}
```

## Status

todo
