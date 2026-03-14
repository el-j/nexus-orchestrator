# TASK-286 — Frontend: "AI Agents" dashboard section

**Plan:** PLAN-044  
**Status:** TODO  
**Layer:** Vue · frontend (`frontend/src/`)  
**Depends on:** TASK-281  

## Objective

Add an "AI Agents" view to the Wails/browser dashboard showing registered sessions (colour-coded) and unregistered discovered agents, with a "Delegate All" action.

## Changes

### New view: `frontend/src/views/AIAgentsView.vue`

Route: `/agents` (add to `frontend/src/router/index.ts`).

**Template structure:**
```
<section class="ai-agents">
  <header>
    <h2>AI Agents</h2>
    <button @click="delegateAll" :disabled="delegatingAll">Delegate All Active</button>
    <button @click="refresh" class="icon-btn">↺</button>
  </header>

  <div class="sessions-grid">
    <AISessionCard v-for="s in sessions" :key="s.id" :session="s" @delegate="delegate(s)" />
  </div>

  <details>
    <summary>Discovered (unregistered) agents ({{ discoveredAgents.length }})</summary>
    <DiscoveredAgentList :agents="discoveredAgents" />
  </details>
</section>
```

**Data fetching:**
- `GET /api/ai-sessions` every 10 s (use existing `useInterval` composable or inline `setInterval`)
- `GET /api/ai-sessions/discovered` every 30 s

### New component: `frontend/src/components/AISessionCard.vue`

Props: `session: AISession`.

Status dot colour (CSS variable or inline style):
```typescript
function statusColour(session: AISession): string {
  if (session.status === 'active' && session.delegatedToNexus) return '#4ade80'; // green
  if (session.status === 'active') return '#facc15';                              // yellow
  if (session.status === 'idle')   return '#fb923c';                              // orange
  return '#6b7280';                                                               // gray (disconnected)
}
```

Card content:
- Agent name (bold)
- Source badge (`vscode`, `mcp`, `http`, `vscode-discovered`)
- Last two segments of `projectPath`
- Last activity relative time
- Capability chips: `file-write`, `code-execute`, `mcp-client`, `chat`, `terminal`
- "Delegate →" button (calls `POST /api/ai-sessions/{id}/delegate` directly)
- If `delegatedToNexus`: green "✓ Delegated" badge

**`delegate(session)` handler in `AIAgentsView.vue`:**
```typescript
async function delegate(session: AISession) {
  const resp = await fetch(`${daemonUrl}/api/ai-sessions/${session.id}/delegate`, { method: 'POST' });
  const data = await resp.json();
  // Show instruction in a modal or copy to clipboard
  await navigator.clipboard.writeText(data.instruction);
  showNotification('Delegation instruction copied to clipboard');
  await refresh();
}
```

**`delegateAll()` handler:**
Iterates `sessions.filter(s => s.status === 'active' && !s.delegatedToNexus)` and calls `delegate(s)` for each sequentially (to avoid request flood).

### Sidebar nav addition (`frontend/src/App.vue` or router link component)

Add nav link: `<RouterLink to="/agents">AI Agents</RouterLink>` with icon `🤖`.

## Acceptance Criteria

- View renders at `/ui#/agents` (or however the SPA routes are structured)
- Delegated sessions display green dot
- "Delegate All" iterates non-delegated active sessions
- Handles 0 sessions gracefully (empty state message)
- No TypeScript build errors
