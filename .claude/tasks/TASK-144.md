---
id: TASK-144
title: Wire AppSidebar routing and create ProvidersView
role: frontend
planId: PLAN-020
status: todo
dependencies: []
createdAt: 2026-03-11T19:00:00.000Z
---

## Context

`AppSidebar.vue` stores `activeView` in a local `ref` that is never shared with the
parent. `App.vue` hardcodes `<DashboardView />` — clicking "Providers" in the sidebar
does nothing visible. Requires:
1. AppSidebar to emit the selected view.
2. App.vue to switch between `DashboardView` and `ProvidersView` based on selection.
3. A `ProvidersView.vue` that shows the full provider status prominently.

## Files to Read

- `frontend/src/components/AppSidebar.vue`
- `frontend/src/App.vue`
- `frontend/src/views/DashboardView.vue` (to understand structure)
- `frontend/src/components/ProviderStatus.vue` (to understand what to embed in ProvidersView)
- `frontend/src/composables/useProviders.ts`
- `frontend/src/types/domain.ts`

## Implementation Steps

### 1. Update `AppSidebar.vue` — emit view changes

Add `defineEmits` and emit on nav click:

```vue
<script setup lang="ts">
import { ref } from 'vue'

const emit = defineEmits<{ (e: 'view-change', id: string): void }>()

const activeView = ref('dashboard')
const navItems = [
  { id: 'dashboard', label: 'Task Queue', icon: 'pi-list' },
  { id: 'providers', label: 'Providers', icon: 'pi-server' },
]

function navigate(id: string) {
  activeView.value = id
  emit('view-change', id)
}
</script>
```

Change the template click handler from `@click="activeView = item.id"` to
`@click="navigate(item.id)"`.

### 2. Update `App.vue` — handle view switching

```vue
<template>
  <div class="flex h-screen bg-[#050508] overflow-hidden">
    <AppSidebar @view-change="currentView = $event" />
    <main class="flex-1 flex flex-col overflow-hidden">
      <DashboardView v-if="currentView === 'dashboard'" />
      <ProvidersView v-else-if="currentView === 'providers'" />
    </main>
    <Toast position="bottom-right" />
    <ConfirmDialog />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import Toast from 'primevue/toast'
import ConfirmDialog from 'primevue/confirmdialog'
import AppSidebar from './components/AppSidebar.vue'
import DashboardView from './views/DashboardView.vue'
import ProvidersView from './views/ProvidersView.vue'

const currentView = ref('dashboard')
</script>
```

### 3. Create `frontend/src/views/ProvidersView.vue`

A full-page provider management view. It should:
- Show a page header ("Providers") with a refresh button
- Render all discovered providers as larger cards with: name, status indicator,
  base URL, active model (if any), error message (if any), all available models list
- Show the "Configured Providers" CRUD section below (reuse / inline the ProviderStatus logic)
- Use the same dark theme as DashboardView

```vue
<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <header class="flex items-center justify-between px-5 py-3 border-b border-white/5 bg-[#0a0a10] flex-shrink-0">
      <div>
        <h1 class="text-sm font-bold text-white">Providers</h1>
        <p class="text-xs text-slate-500">
          <span class="font-semibold" :class="activeCount > 0 ? 'text-emerald-400' : 'text-slate-500'">
            {{ activeCount }}
          </span> active of {{ providers.length }} discovered
        </p>
      </div>
      <button
        class="text-xs text-slate-400 hover:text-white px-3 py-1.5 rounded-lg border border-white/10 hover:border-violet-500/40 transition-all"
        @click="refresh()"
      >⟳ Refresh</button>
    </header>

    <!-- Content -->
    <div class="flex-1 overflow-auto p-5 space-y-6">

      <!-- Discovered providers -->
      <section>
        <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Discovered Providers</h2>
        <div v-if="providers.length === 0" class="text-sm text-slate-600 py-8 text-center">
          No providers detected — start LM Studio or Ollama
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div
            v-for="p in providers"
            :key="p.name"
            class="rounded-xl border p-4 transition-all"
            :class="p.active
              ? 'border-emerald-500/30 bg-emerald-500/5'
              : 'border-red-500/20 bg-red-500/5'"
          >
            <div class="flex items-start justify-between gap-2 mb-2">
              <div class="flex items-center gap-2">
                <span :class="['w-2 h-2 rounded-full flex-shrink-0 mt-0.5', p.active ? 'bg-emerald-400 animate-pulse' : 'bg-red-500']"></span>
                <span class="font-semibold text-sm text-white">{{ p.name }}</span>
              </div>
              <span
                class="text-[10px] px-1.5 py-0.5 rounded font-medium"
                :class="p.active ? 'bg-emerald-500/20 text-emerald-300' : 'bg-red-500/20 text-red-400'"
              >{{ p.active ? 'Active' : 'Unreachable' }}</span>
            </div>

            <div class="text-[11px] text-slate-500 font-mono mb-2">{{ p.baseURL }}</div>

            <div v-if="p.active && p.activeModel" class="text-xs text-slate-400 mb-1">
              Active model: <span class="text-violet-300 font-mono">{{ p.activeModel }}</span>
            </div>

            <div v-if="p.active && p.models?.length" class="flex flex-wrap gap-1 mt-2">
              <span
                v-for="m in p.models"
                :key="m"
                class="text-[10px] px-1.5 py-0.5 rounded bg-white/5 text-slate-400 font-mono"
                :class="m === p.activeModel ? 'border border-violet-500/30 text-violet-300' : ''"
              >{{ m }}</span>
            </div>

            <div v-if="!p.active && p.error" class="text-[11px] text-amber-500/80 mt-2 font-mono">
              {{ p.error }}
            </div>
          </div>
        </div>
      </section>

      <!-- Configured providers section (reuse ProviderStatus inline) -->
      <section>
        <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Configured Providers</h2>
        <ProviderStatus :providers="providers" :refresh="refresh" />
      </section>

    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useProviders } from '../composables/useProviders'
import ProviderStatus from '../components/ProviderStatus.vue'

const { providers, refresh } = useProviders()

const activeCount = computed(() => providers.value.filter(p => p.active).length)
</script>
```

Note: The `ProviderStatus` component will still show the "Configured Providers" CRUD
section — hide the duplicate "Discovered Providers" mini-list from ProviderStatus when
embedded in ProvidersView by not passing it, OR just let it render both sections
(the compact discovered list + the configured CRUD). Either way is acceptable.

Actually, to avoid double-rendering the discovered providers mini-list, add an optional
`hideDiscovered` prop to `ProviderStatus.vue`:

```vue
<!-- ProviderStatus.vue -->
const props = defineProps<{ providers: ProviderInfo[], refresh?: () => void, hideDiscovered?: boolean }>()
```

And wrap the discovered providers loop with `v-if="!hideDiscovered"`. Then pass
`:hide-discovered="true"` when embedding in ProvidersView.

## Acceptance Criteria

- [ ] `AppSidebar.vue` emits `'view-change'` on nav click
- [ ] `App.vue` renders `ProvidersView` when `currentView === 'providers'`
- [ ] `ProvidersView.vue` exists and shows provider cards with name, status, baseURL, models
- [ ] TypeScript compiler reports no errors: `cd frontend && npm run build 2>&1`

## Anti-patterns to Avoid

- Do not add `vue-router` — the app uses a simple ref-based view switcher
- Do not change the DashboardView layout
- Do not add unnecessary animations or effects beyond what already exists in the project
