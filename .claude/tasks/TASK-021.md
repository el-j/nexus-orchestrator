---
id: TASK-021
title: GUI — Task History page + TaskDetail Drawer (Vue 3 + PrimeVue 4)
role: devops
planId: PLAN-002
status: done
completedAt: 2026-03-14T18:30:00.000Z
dependencies: [TASK-017, TASK-018]
createdAt: 2026-03-09T14:00:00.000Z
---

## Context

The Task History page shows all completed and failed tasks in a filterable PrimeVue DataTable. Selecting a row opens a slide-in PrimeVue `Drawer` panel showing full prompt, session message history, writeback source info, and a "Retry" action. This is the primary debugging and auditability view.

## Files to Read

- `frontend/src/types/domain.ts` — Task, Message interfaces
- `frontend/src/components/StatusTag.vue` — reuse from TASK-019
- `frontend/src/composables/useNexus.ts` — Go bindings
- `frontend/src/utils/time.ts` — relativeTime (from TASK-020)
- `app.go` — `GetAllTasks`, `GetSession`, `ClearSession`, `SubmitTask` signatures

## Implementation Steps

1. **Create `frontend/src/pages/TaskHistory.vue`**:

   ```vue
   <script setup lang="ts">
   import { ref, computed } from 'vue';
   import { useQuery, useQueryClient } from '@tanstack/vue-query';
   import { useToast } from 'primevue/usetoast';
   import { useRouter } from 'vue-router';
   import DataTable from 'primevue/datatable';
   import Column from 'primevue/column';
   import InputText from 'primevue/inputtext';
   import Button from 'primevue/button';
   import Select from 'primevue/select'; // PrimeVue 4 — formerly Dropdown
   import { useNexus } from '../composables/useNexus';
   import StatusTag from '../components/StatusTag.vue';
   import TaskDetail from '../components/TaskDetail.vue';
   import type { Task } from '../types/domain';

   const nexus = useNexus();
   const toast = useToast();
   const router = useRouter();
   const queryClient = useQueryClient();

   const filterStatus = ref<'all' | 'completed' | 'failed'>('all');
   const filterProject = ref('');
   const selectedTask = ref<Task | null>(null);

   const statusOptions = [
     { label: 'All', value: 'all' },
     { label: 'Completed', value: 'completed' },
     { label: 'Failed', value: 'failed' },
   ];

   const { data: completed } = useQuery({
     queryKey: ['history', 'completed'],
     queryFn: () => nexus.getAllTasks('completed'),
     refetchInterval: 5000,
   });
   const { data: failed } = useQuery({
     queryKey: ['history', 'failed'],
     queryFn: () => nexus.getAllTasks('failed'),
     refetchInterval: 5000,
   });

   const allTasks = computed(() => {
     const c = filterStatus.value !== 'failed' ? (completed.value ?? []) : [];
     const f = filterStatus.value !== 'completed' ? (failed.value ?? []) : [];
     return [...c, ...f]
       .filter((t) => !filterProject.value || t.projectPath.includes(filterProject.value))
       .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
   });

   async function retryTask(task: Task) {
     try {
       const id = await nexus.submitTask(task.projectPath, task.prompt);
       toast.add({
         severity: 'success',
         summary: 'Retried',
         detail: `New task: ${id}`,
         life: 4000,
       });
       router.push('/queue');
     } catch (e) {
       toast.add({ severity: 'error', summary: 'Retry failed', detail: String(e), life: 4000 });
     }
   }
   </script>

   <template>
     <div class="p-6 h-full flex flex-col gap-4">
       <div class="flex items-center justify-between">
         <h1 class="text-xl font-semibold">Task History</h1>
         <div class="flex gap-2">
           <InputText v-model="filterProject" placeholder="Filter by project..." class="w-56" />
           <Select
             v-model="filterStatus"
             :options="statusOptions"
             optionLabel="label"
             optionValue="value"
             class="w-36"
           />
         </div>
       </div>

       <DataTable
         :value="allTasks"
         dataKey="id"
         stripedRows
         scrollable
         scrollHeight="flex"
         selectionMode="single"
         v-model:selection="selectedTask"
         class="flex-1"
       >
         <template #empty>No tasks in history yet.</template>
         <Column field="status" header="Status">
           <template #body="{ data }"><StatusTag :status="data.status" /></template>
         </Column>
         <Column field="projectPath" header="Project">
           <template #body="{ data }">
             <span :title="data.projectPath" class="text-sm font-mono">
               {{ data.projectPath.split('/').slice(-2).join('/') }}
             </span>
           </template>
         </Column>
         <Column field="prompt" header="Prompt">
           <template #body="{ data }"
             >{{ data.prompt.slice(0, 80) }}{{ data.prompt.length > 80 ? '…' : '' }}</template
           >
         </Column>
         <Column field="retryCount" header="Retries" style="width: 80px" />
         <Column field="updatedAt" header="Completed">
           <template #body="{ data }">{{ relativeTime(data.updatedAt) }}</template>
         </Column>
         <Column header="Actions" style="width: 100px">
           <template #body="{ data }">
             <Button
               icon="pi pi-replay"
               text
               rounded
               size="small"
               title="Retry"
               @click.stop="retryTask(data)"
             />
           </template>
         </Column>
       </DataTable>

       <TaskDetail
         v-model:visible="!!selectedTask"
         :task="selectedTask"
         @close="selectedTask = null"
         @retry="retryTask"
       />
     </div>
   </template>
   ```

   Import `relativeTime` from `../utils/time`.

2. **Create `frontend/src/components/TaskDetail.vue`** (PrimeVue Drawer, right side, 520px wide):

   ```vue
   <script setup lang="ts">
   import { computed } from 'vue';
   import { useQuery, useQueryClient } from '@tanstack/vue-query';
   import { useToast } from 'primevue/usetoast';
   import Drawer from 'primevue/drawer';
   import Button from 'primevue/button';
   import Tag from 'primevue/tag';
   import { useNexus } from '../composables/useNexus';
   import StatusTag from './StatusTag.vue';
   import type { Task } from '../types/domain';

   const props = defineProps<{ visible: boolean; task: Task | null }>();
   const emit = defineEmits<{ 'update:visible': [boolean]; close: []; retry: [Task] }>();

   const nexus = useNexus();
   const toast = useToast();
   const queryClient = useQueryClient();

   const sessionKey = computed(() => props.task?.projectPath ?? '');
   const { data: messages } = useQuery({
     queryKey: computed(() => ['session', sessionKey.value]),
     queryFn: () => nexus.getSession(sessionKey.value),
     enabled: computed(() => !!props.task),
   });

   async function clearSession() {
     if (!props.task) return;
     await nexus.clearSession(props.task.projectPath);
     await queryClient.invalidateQueries({ queryKey: ['session', sessionKey.value] });
     toast.add({ severity: 'info', summary: 'Session cleared', life: 3000 });
   }
   </script>

   <template>
     <Drawer
       :visible="visible"
       @update:visible="emit('update:visible', $event)"
       position="right"
       class="!w-[520px]"
       @hide="emit('close')"
     >
       <template #header>
         <div class="flex items-center gap-2">
           <StatusTag v-if="task" :status="task.status" />
           <code class="text-sm text-surface-400">{{ task?.id }}</code>
         </div>
       </template>

       <template v-if="task">
         <!-- Prompt -->
         <section class="mb-6">
           <h3 class="text-xs uppercase text-surface-500 mb-2 font-semibold tracking-wide">
             Prompt
           </h3>
           <pre class="whitespace-pre-wrap text-sm bg-surface-900 rounded-lg p-3">{{
             task.prompt
           }}</pre>
         </section>

         <!-- Session history -->
         <section class="mb-6">
           <div class="flex items-center justify-between mb-2">
             <h3 class="text-xs uppercase text-surface-500 font-semibold tracking-wide">
               Session History
             </h3>
             <Button
               label="Clear"
               icon="pi pi-trash"
               text
               size="small"
               severity="secondary"
               @click="clearSession"
             />
           </div>
           <div v-if="!messages?.length" class="text-sm text-surface-600 italic">
             No session history.
           </div>
           <div v-else class="space-y-2 max-h-64 overflow-y-auto">
             <div
               v-for="(msg, i) in messages"
               :key="i"
               :class="msg.role === 'user' ? 'text-right' : 'text-left'"
             >
               <Tag
                 :value="msg.role"
                 :severity="msg.role === 'user' ? 'info' : 'secondary'"
                 class="text-xs mb-1"
               />
               <pre
                 class="whitespace-pre-wrap text-xs bg-surface-900 rounded-lg p-2 inline-block max-w-[90%] text-left"
                 >{{ msg.content }}</pre
               >
             </div>
           </div>
         </section>

         <!-- Source writeback info (shown only if sourceProjectPath is set) -->
         <section v-if="task.sourceProjectPath" class="mb-6">
           <h3 class="text-xs uppercase text-surface-500 mb-2 font-semibold tracking-wide">
             Writeback Source
           </h3>
           <table class="text-sm w-full">
             <tr>
               <td class="text-surface-500 pr-3 py-0.5">Project</td>
               <td class="font-mono text-xs">{{ task.sourceProjectPath }}</td>
             </tr>
             <tr>
               <td class="text-surface-500 pr-3 py-0.5">Task ID</td>
               <td class="font-mono text-xs">{{ task.sourceTaskId }}</td>
             </tr>
             <tr v-if="task.sourcePlanId">
               <td class="text-surface-500 pr-3 py-0.5">Plan ID</td>
               <td class="font-mono text-xs">{{ task.sourcePlanId }}</td>
             </tr>
           </table>
         </section>

         <!-- Metadata -->
         <section>
           <h3 class="text-xs uppercase text-surface-500 mb-2 font-semibold tracking-wide">
             Metadata
           </h3>
           <table class="text-xs w-full">
             <tr
               v-for="[k, v] in [
                 ['Project', task.projectPath],
                 ['Retries', task.retryCount],
                 ['Created', task.createdAt],
                 ['Updated', task.updatedAt],
               ]"
               :key="k"
             >
               <td class="text-surface-500 pr-3 py-0.5 w-20">{{ k }}</td>
               <td class="font-mono break-all">{{ v }}</td>
             </tr>
           </table>
         </section>
       </template>

       <template #footer>
         <Button
           label="Retry Task"
           icon="pi pi-replay"
           class="w-full"
           @click="task && emit('retry', task)"
         />
       </template>
     </Drawer>
   </template>
   ```

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./...` exits 0
- [ ] `cd frontend && npm run build` exits 0
- [ ] `cd frontend && npm run type-check` exits 0
- [ ] `frontend/src/pages/TaskHistory.vue` exists, uses PrimeVue DataTable selection
- [ ] `frontend/src/components/TaskDetail.vue` exists with all 4 sections (Prompt / Session / Source / Metadata)
- [ ] Source section conditionally renders only when `task.sourceProjectPath` is truthy
- [ ] Session query is lazy-loaded — only fires when a task is selected (use `enabled: computed(...)`)
- [ ] `v-model:visible` pattern used for Drawer

## Anti-patterns to Avoid

- NEVER load all sessions on mount — only load `getSession` when a row is selected (use `enabled: computed(...)`)
- NEVER render raw HTML from LLM output — use `<pre>` with `whitespace-pre-wrap`
- NEVER duplicate StatusTag logic — import `StatusTag` from TASK-019
- NEVER use `$emit` in `<script setup>` — use the `emit` function from `defineEmits`
- NEVER use PrimeVue 3 `Dropdown` — use PrimeVue 4 `Select`
