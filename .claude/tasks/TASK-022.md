---
id: TASK-022
title: GUI — Settings page (read-only config overview, Vue 3 + PrimeVue 4)
role: devops
planId: PLAN-002
status: todo
dependencies: [TASK-017, TASK-018]
createdAt: 2026-03-09T14:00:00.000Z
---

## Context

The Settings page shows the current nexusOrchestrator configuration in a read-only PrimeVue-styled layout: provider status, network addresses, queue settings, all environment variables, and an About section. Phase 1 is entirely read-only; write-back to config files is out of scope. PrimeVue `DataTable`, `Panel`, and `Fieldset` components are used for structure.

## Files to Read

- `frontend/src/types/domain.ts` — Provider, Stats interfaces
- `frontend/src/composables/useNexus.ts`
- `app.go` — `GetProviders()`, `GetStats()` signatures
- `.github/copilot-instructions.md` — env var names section

## Implementation Steps

1. **Create `frontend/src/pages/Settings.vue`**:
   - Layout: `max-w-3xl mx-auto p-6 space-y-6`
   - Uses PrimeVue `Panel` for each section (collapsible, starts expanded)
   - Page title: `<h1>Settings</h1>` with sub-text "Read-only. Restart daemon to apply env var changes."

2. **Section "LLM Providers"** using PrimeVue `DataTable`:
   ```vue
   <Panel header="LLM Providers">
     <DataTable :value="providers ?? []" size="small">
       <Column field="name" header="Provider" />
       <Column field="baseURL" header="Base URL" />
       <Column header="Status">
         <template #body="{ data }">
           <Tag :severity="data.available ? 'success' : 'danger'"
                :value="data.available ? 'Online' : 'Offline'" />
         </template>
       </Column>
       <Column header="Models">
         <template #body="{ data }">{{ data.models.slice(0, 3).join(', ') }}{{ data.models.length > 3 ? ` +${data.models.length - 3}` : '' }}</template>
       </Column>
     </DataTable>
   </Panel>
   ```

3. **Section "Network"** using PrimeVue `Fieldset`:
   ```vue
   <Fieldset legend="Network Addresses">
     <div class="grid grid-cols-2 gap-4 text-sm">
       <div>
         <div class="text-surface-500 text-xs mb-1">HTTP API</div>
         <code class="bg-surface-900 px-2 py-1 rounded">NEXUS_LISTEN_ADDR=:9999</code>
       </div>
       <div>
         <div class="text-surface-500 text-xs mb-1">MCP Server</div>
         <code class="bg-surface-900 px-2 py-1 rounded">NEXUS_MCP_ADDR=:9998</code>
       </div>
     </div>
   </Fieldset>
   ```

4. **Section "Queue & Storage"**: shows `stats.queueDepth` (live, `refetchInterval: 5000`) + env var defaults:
   ```
   NEXUS_MAX_QUEUE=500
   NEXUS_DB_PATH=nexus.db
   ```

5. **Section "Environment Variables"** with copy-to-clipboard:
   ```vue
   <Panel header="Environment Variables">
     <pre class="bg-surface-900 rounded-lg p-4 text-xs font-mono text-surface-300 select-all">
   NEXUS_DB_PATH=nexus.db
   NEXUS_LISTEN_ADDR=:9999
   NEXUS_MCP_ADDR=:9998
   NEXUS_MAX_QUEUE=500</pre>
     <Button label="Copy to clipboard" icon="pi pi-copy" text size="small" class="mt-2"
             @click="copyEnvVars" />
   </Panel>
   ```
   `copyEnvVars` uses `navigator.clipboard.writeText(envVarsText)` with toast feedback.

6. **Section "About"**:
   - Module: `nexus-ai` (Go 1.24)
   - Protocol: HTTP API `:9999` + MCP JSON-RPC 2.0 `:9998`
   - Wails: v2.11.0
   - Vue: 3.x + PrimeVue 4

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./...` exits 0
- [ ] `cd frontend && npm run build` exits 0
- [ ] `cd frontend && npm run type-check` exits 0
- [ ] `frontend/src/pages/Settings.vue` exists with all 5 sections
- [ ] "Copy to clipboard" button works (uses `navigator.clipboard`)
- [ ] Provider table uses live `getProviders()` data with status Tag  
- [ ] No editable inputs — read-only page only
- [ ] Settings routed at `/settings` via Vue Router

## Anti-patterns to Avoid

- NEVER use `<Dropdown>` (PrimeVue v3 name) — use `<Select>` (PrimeVue v4)
- NEVER attempt to write env vars from JavaScript
- NEVER call `window.open()` in Wails WebView
- NEVER add form inputs that could call save/write functions (phase 1 is read-only)
