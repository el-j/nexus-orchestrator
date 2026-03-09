---
id: TASK-020
title: GUI — Task Queue page with live DataTable and submit Drawer (Vue 3 + PrimeVue 4)
role: devops
planId: PLAN-002
status: todo
dependencies: [TASK-017, TASK-018]
createdAt: 2026-03-09T14:00:00.000Z
---

## Context

The Task Queue page is the primary operational view. It shows live queued/processing tasks in a PrimeVue `DataTable` with a slide-in `Drawer` to submit new tasks. Cancellation and toast feedback use PrimeVue's built-in `useToast()` and `ConfirmDialog`. This replaces the React-based design.

## Files to Read

- `frontend/src/App.vue` — global Toast and RouterView
- `frontend/src/types/domain.ts` — Task interface
- `frontend/src/components/StatusTag.vue` — reuse from TASK-019
- `frontend/src/composables/useNexus.ts` — Go bindings
- `app.go` — `SubmitTask`, `GetQueue`, `CancelTask` signatures

## Implementation Steps

1. **Create `frontend/src/pages/TaskQueue.vue`**:
   ```vue
   <script setup lang="ts">
   import { ref } from 'vue'
   import { useQuery, useQueryClient } from '@tanstack/vue-query'
   import { useToast } from 'primevue/usetoast'
   import DataTable from 'primevue/datatable'
   import Column from 'primevue/column'
   import Button from 'primevue/button'
   import { useNexus } from '../composables/useNexus'
   import StatusTag from '../components/StatusTag.vue'
   import SubmitDrawer from '../components/SubmitDrawer.vue'
   import type { Task } from '../types/domain'

   const nexus = useNexus()
   const toast = useToast()
   const queryClient = useQueryClient()
   const showSubmit = ref(false)

   const { data: queue, isLoading } = useQuery({
     queryKey: ['queue'],
     queryFn: nexus.getQueue,
     refetchInterval: 2000
   })

   async function cancelTask(task: Task) {
     try {
       await nexus.cancelTask(task.id)
       await queryClient.invalidateQueries({ queryKey: ['queue'] })
       toast.add({ severity: 'info', summary: 'Cancelled', detail: task.id, life: 3000 })
     } catch (e) {
       toast.add({ severity: 'error', summary: 'Cancel failed', detail: String(e), life: 4000 })
     }
   }
   </script>
   ```

2. **Queue DataTable**:
   - `:value="queue ?? []"`, `dataKey="id"`, `stripedRows`, `:loading="isLoading"`, `scrollable scrollHeight="flex"`
   - Column "Status": `<template #body="{ data }"><StatusTag :status="data.status" /></template>`
   - Column "ID": monospace, first 8 chars + `...` with full ID as `title`
   - Column "Project": truncated last 2 path segments
   - Column "Prompt": first 80 chars
   - Column "Submitted": relative time (use `Intl.RelativeTimeFormat`)
   - Column "Actions": `<Button icon="pi pi-times" severity="danger" text rounded` disabled when `data.status !== 'queued'`
   - Empty state: PrimeVue `template #empty` — "Queue is empty — click Submit to add a task"

3. **Create `frontend/src/components/SubmitDrawer.vue`** using PrimeVue `Drawer`:
   ```vue
   <script setup lang="ts">
   import { ref } from 'vue'
   import { useQueryClient } from '@tanstack/vue-query'
   import { useToast } from 'primevue/usetoast'
   import Drawer from 'primevue/drawer'
   import Button from 'primevue/button'
   import InputText from 'primevue/inputtext'
   import Textarea from 'primevue/textarea'
   import { useNexus } from '../composables/useNexus'

   const props = defineProps<{ visible: boolean }>()
   const emit = defineEmits<{ 'update:visible': [value: boolean] }>()

   const nexus = useNexus()
   const toast = useToast()
   const queryClient = useQueryClient()

   const projectPath = ref('')
   const prompt = ref('')
   const submitting = ref(false)

   async function submit() {
     if (!projectPath.value.trim() || !prompt.value.trim()) return
     submitting.value = true
     try {
       const id = await nexus.submitTask(projectPath.value.trim(), prompt.value.trim())
       toast.add({ severity: 'success', summary: 'Submitted', detail: `Task ${id}`, life: 4000 })
       projectPath.value = ''
       prompt.value = ''
       emit('update:visible', false)
       await queryClient.invalidateQueries({ queryKey: ['queue'] })
     } catch (e) {
       toast.add({ severity: 'error', summary: 'Submit failed', detail: String(e), life: 5000 })
     } finally {
       submitting.value = false
     }
   }
   </script>

   <template>
     <Drawer :visible="visible" @update:visible="$emit('update:visible', $event)"
             header="Submit Task" position="right" class="!w-[420px]">
       <div class="flex flex-col gap-4">
         <div>
           <label class="block text-sm mb-1 text-surface-300">Project Path</label>
           <InputText v-model="projectPath" class="w-full" placeholder="$PWD or /path/to/project" />
         </div>
         <div>
           <label class="block text-sm mb-1 text-surface-300">Prompt</label>
           <Textarea v-model="prompt" class="w-full" rows="6" placeholder="Describe the task..." :maxlength="4000" autoResize />
           <div class="text-xs text-surface-500 text-right mt-1">{{ prompt.length }}/4000</div>
         </div>
         <Button label="Submit Task" icon="pi pi-send" :loading="submitting"
                 :disabled="!projectPath.trim() || !prompt.trim()"
                 class="w-full" @click="submit" />
       </div>
     </Drawer>
   </template>
   ```

4. **Page header** (top of TaskQueue.vue template):
   ```html
   <div class="flex items-center justify-between p-6 pb-0">
     <h1 class="text-xl font-semibold">Task Queue</h1>
     <Button label="Submit Task" icon="pi pi-plus" @click="showSubmit = true" />
   </div>
   ```

5. **Relative time helper** `frontend/src/utils/time.ts`:
   ```typescript
   const rtf = new Intl.RelativeTimeFormat('en', { numeric: 'auto' })
   export function relativeTime(isoString: string): string {
     const diff = (new Date(isoString).getTime() - Date.now()) / 1000
     if (Math.abs(diff) < 60) return rtf.format(Math.round(diff), 'seconds')
     if (Math.abs(diff) < 3600) return rtf.format(Math.round(diff / 60), 'minutes')
     return rtf.format(Math.round(diff / 3600), 'hours')
   }
   ```

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./...` exits 0
- [ ] `cd frontend && npm run build` exits 0
- [ ] `cd frontend && npm run type-check` exits 0
- [ ] `frontend/src/pages/TaskQueue.vue` exists using PrimeVue DataTable + Drawer
- [ ] `frontend/src/components/SubmitDrawer.vue` exists with project + prompt fields
- [ ] `frontend/src/utils/time.ts` exists with `relativeTime` helper
- [ ] Cancel button disabled for non-queued tasks
- [ ] v-model pattern used for Drawer visibility (not manual show/hide)

## Anti-patterns to Avoid

- NEVER use `window.go.*` directly — use `useNexus()` composable
- NEVER mutate `queue.value` directly — use `queryClient.invalidateQueries`
- NEVER use Vue 2 Options API (`data()`, `methods:`) — use `<script setup>`
- NEVER use `<Dialog>` for the submit form — use PrimeVue 4 `<Drawer>`
- NEVER skip error handling on `cancelTask` / `submitTask` calls
