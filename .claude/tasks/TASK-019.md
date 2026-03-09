---
id: TASK-019
title: GUI — Dashboard view + Provider status cards (Vue 3 + PrimeVue 4)
role: devops
planId: PLAN-002
status: todo
dependencies: [TASK-017, TASK-018]
createdAt: 2026-03-09T14:00:00.000Z
---

## Context

The Dashboard is the default landing page of the Wails 2 GUI. It shows: live queue statistics, a mini activity feed of recent task completions, and PrimeVue Card-based provider health cards showing which LLMs are online. This replaces the current stub HTML.

## Files to Read

- `frontend/src/App.vue` — shell nav + RouterView setup (created TASK-017)
- `frontend/src/types/domain.ts` — Task, Provider, Stats interfaces
- `frontend/src/composables/useNexus.ts` — Go binding wrappers
- `app.go` — `GetQueue()`, `GetProviders()`, `GetStats()` signatures
- `internal/core/ports/ports.go` — ProviderInfo struct (to match JSON shape)

## Implementation Steps

1. **Create `frontend/src/pages/Dashboard.vue`**:
   - Uses `@tanstack/vue-query` `useQuery` with `refetchInterval: 3000`.
   - Three sections: stat cards row, recent-tasks panel (left 60%), provider cards (right 40%).

2. **Stat cards** using PrimeVue `Card` or plain Tailwind divs:
   ```vue
   <script setup lang="ts">
   import { computed } from 'vue'
   import { useQuery } from '@tanstack/vue-query'
   import { useNexus } from '../composables/useNexus'
   import Card from 'primevue/card'
   import Tag from 'primevue/tag'
   import ProgressSpinner from 'primevue/progressspinner'
   import { Cpu, ListTodo, Zap } from 'lucide-vue-next'
   import type { Task, Provider, Stats } from '../types/domain'

   const nexus = useNexus()

   const { data: stats } = useQuery({ queryKey: ['stats'], queryFn: nexus.getStats, refetchInterval: 3000 })
   const { data: providers, isLoading: loadingProviders } = useQuery({ queryKey: ['providers'], queryFn: nexus.getProviders, refetchInterval: 5000 })
   const { data: completed } = useQuery({ queryKey: ['history', 'completed'], queryFn: () => nexus.getAllTasks('completed'), refetchInterval: 5000 })
   const { data: failed } = useQuery({ queryKey: ['history', 'failed'], queryFn: () => nexus.getAllTasks('failed'), refetchInterval: 5000 })

   const recentTasks = computed(() => {
     const all = [...(completed.value ?? []), ...(failed.value ?? [])]
     return all.sort((a, b) => b.updatedAt.localeCompare(a.updatedAt)).slice(0, 10)
   })

   const onlineProviders = computed(() => (providers.value ?? []).filter(p => p.available))
   </script>
   ```

3. **Stats row** — 3 `Card` components in a `grid grid-cols-3 gap-4`:
   - "Queue Depth": `stats?.queueDepth ?? 0` with `ListTodo` icon, severity `info` Tag
   - "Active Task": `stats?.activeTask || 'Idle'` with `Zap` icon  
   - "Providers Online": `onlineProviders.length / (providers?.length ?? 1)` with `Cpu` icon

4. **Recent tasks table** using PrimeVue `DataTable` + `Column`:
   - `:value="recentTasks"`, `:rows="10"`, `scrollable scrollHeight="300px"`
   - Columns: Status (custom `<Tag>` slot), Project (truncated with `title` attr), Prompt (60 chars), Time
   - Status Tag severity mapping: `completed → success`, `failed → danger`, `processing → warn`, `queued → info`
   - Empty state slot: "No completed tasks yet"

5. **Provider cards** using `v-for` over `providers` — each PrimeVue `Card`:
   - Header: provider name + online/offline `Tag`
   - Content: model count, base URL
   - Footer: shows models as comma list (truncated to 3)
   - Pulsing green dot for online, grey for offline (Tailwind `animate-pulse`)

6. **Create `frontend/src/components/StatusTag.vue`** — reusable PrimeVue `Tag` wrapper:
   ```vue
   <script setup lang="ts">
   import Tag from 'primevue/tag'
   const props = defineProps<{ status: 'queued' | 'processing' | 'completed' | 'failed' }>()
   const severityMap = { queued: 'info', processing: 'warn', completed: 'success', failed: 'danger' } as const
   const labelMap = { queued: 'Queued', processing: 'Processing', completed: 'Done', failed: 'Failed' }
   </script>
   <template>
     <Tag :severity="severityMap[status]" :value="labelMap[status]" />
   </template>
   ```

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./...` exits 0
- [ ] `cd frontend && npm run build` exits 0
- [ ] `cd frontend && npm run type-check` exits 0
- [ ] `frontend/src/pages/Dashboard.vue` exists, uses PrimeVue Card + DataTable
- [ ] `frontend/src/components/StatusTag.vue` exists with 4 severity mappings
- [ ] `frontend/src/types/domain.ts` is imported (not inlined)
- [ ] Dashboard is reachable via Vue Router at `/dashboard`

## Anti-patterns to Avoid

- NEVER use React hooks — use Vue 3 `useQuery`, `computed`, `ref`
- NEVER poll faster than 2s from a Vue component — it locks the Wails bridge
- NEVER use `any` in TypeScript — use `domain.ts` interfaces
- NEVER hardcode provider names — derive from `getProviders()` response
- NEVER use `<style scoped>` for layout that should be Tailwind utilities
