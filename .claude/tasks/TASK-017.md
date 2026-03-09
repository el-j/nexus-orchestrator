---
id: TASK-017
title: Frontend scaffold — Vue 3 + PrimeVue 4 + Tailwind CSS 4 + Vite
role: devops
planId: PLAN-002
status: todo
dependencies: [TASK-015]
createdAt: 2026-03-09T14:00:00.000Z
---

## Context

`frontend/dist/index.html` is a placeholder stub with no source. This task creates the complete frontend scaffold for the Wails 2 GUI using Vue 3 (Composition API + `<script setup>`), PrimeVue 4 with the Aura dark theme, Tailwind CSS 4 (CSS-first configuration via `@tailwindcss/vite`), Pinia v2 for state, Vue Router 4 for navigation, and `@tanstack/vue-query` v5 for server-state. All GUI view tasks (TASK-019 through TASK-022) build on this scaffold.

## Files to Read

- `wails.json` — current Wails frontend configuration
- `go.mod` — confirm Wails v2 version
- `app.go` — binding methods available to JS

## Implementation Steps

1. **Update `wails.json`**:
   ```json
   {
     "name": "nexusOrchestrator",
     "outputfilename": "nexusOrchestrator",
     "frontend:install": "npm install",
     "frontend:build": "npm run build",
     "frontend:dev:watcher": "npm run dev",
     "frontend:dev:serverUrl": "http://localhost:5173",
     "wailsjsdir": "./frontend/src/wailsjs"
   }
   ```

2. **Create `frontend/package.json`**:
   ```json
   {
     "name": "nexus-orchestrator-frontend",
     "private": true,
     "version": "0.1.0",
     "type": "module",
     "scripts": {
       "dev": "vite",
       "build": "vue-tsc -b && vite build",
       "preview": "vite preview",
       "type-check": "vue-tsc --noEmit"
     },
     "dependencies": {
       "vue": "^3.5.13",
       "vue-router": "^4.4.5",
       "pinia": "^2.2.6",
       "primevue": "^4.2.5",
       "@primevue/themes": "^4.2.5",
       "primeicons": "^7.0.0",
       "@tanstack/vue-query": "^5.56.2",
       "lucide-vue-next": "^0.446.0",
       "chart.js": "^4.4.4",
       "vue-chartjs": "^5.3.1"
     },
     "devDependencies": {
       "@vitejs/plugin-vue": "^5.1.4",
       "@vue/tsconfig": "^0.5.1",
       "typescript": "^5.5.3",
       "vite": "^5.4.10",
       "vue-tsc": "^2.1.6",
       "@tailwindcss/vite": "^4.0.0"
     }
   }
   ```

3. **Create `frontend/vite.config.ts`**:
   ```typescript
   import { defineConfig } from 'vite'
   import vue from '@vitejs/plugin-vue'
   import tailwindcss from '@tailwindcss/vite'

   export default defineConfig({
     plugins: [vue(), tailwindcss()],
     build: { outDir: 'dist' },
   })
   ```

4. **Create `frontend/tsconfig.json`**:
   ```json
   {
     "files": [],
     "references": [
       { "path": "./tsconfig.app.json" }
     ]
   }
   ```

5. **Create `frontend/tsconfig.app.json`**:
   ```json
   {
     "extends": "@vue/tsconfig/tsconfig.dom.json",
     "compilerOptions": {
       "target": "ES2022",
       "useDefineForClassFields": true,
       "module": "ESNext",
       "moduleResolution": "bundler",
       "strict": true,
       "noEmit": true,
       "paths": {
         "@/*": ["./src/*"]
       }
     },
     "include": ["src/**/*", "src/**/*.vue"]
   }
   ```

6. **Create `frontend/index.html`** (Vite entry point, note: `frontend/dist/index.html` is the BUILD OUTPUT — this file is the source):
   ```html
   <!doctype html>
   <html lang="en" class="dark">
     <head>
       <meta charset="UTF-8" />
       <meta name="viewport" content="width=device-width, initial-scale=1.0" />
       <link rel="icon" href="/favicon.ico" />
       <title>nexusOrchestrator</title>
     </head>
     <body>
       <div id="app"></div>
       <script type="module" src="/src/main.ts"></script>
     </body>
   </html>
   ```
   Note: This `index.html` lives at `frontend/index.html` (Vite project root), NOT in `frontend/dist/`.

7. **Create `frontend/src/main.ts`**:
   ```typescript
   import { createApp } from 'vue'
   import { createPinia } from 'pinia'
   import { createRouter, createWebHashHistory } from 'vue-router'
   import PrimeVue from 'primevue/config'
   import Aura from '@primevue/themes/aura'
   import ToastService from 'primevue/toastservice'
   import { VueQueryPlugin } from '@tanstack/vue-query'
   import App from './App.vue'
   import './index.css'

   const router = createRouter({
     history: createWebHashHistory(),
     routes: [
       { path: '/', redirect: '/dashboard' },
       { path: '/dashboard', component: () => import('./pages/Dashboard.vue') },
       { path: '/queue',     component: () => import('./pages/TaskQueue.vue') },
       { path: '/history',   component: () => import('./pages/TaskHistory.vue') },
       { path: '/settings',  component: () => import('./pages/Settings.vue') },
     ],
   })

   const app = createApp(App)
   app.use(createPinia())
   app.use(router)
   app.use(PrimeVue, {
     theme: {
       preset: Aura,
       options: {
         darkModeSelector: ':root',
         cssLayer: { name: 'primevue', order: 'tailwind-base, primevue' },
       },
     },
   })
   app.use(ToastService)
   app.use(VueQueryPlugin, {
     queryClientConfig: {
       defaultOptions: { queries: { refetchInterval: 3000, staleTime: 1000 } },
     },
   })
   app.mount('#app')
   ```

8. **Create `frontend/src/index.css`** (Tailwind CSS 4 — CSS-first, NO tailwind.config.js):
   ```css
   @import "tailwindcss";

   @layer tailwind-base, primevue;

   @theme {
     --color-nexus: #6366f1;
     --color-nexus-dark: #4f46e5;
     --color-nexus-light: #818cf8;
     --font-sans: 'Inter', system-ui, sans-serif;
   }

   :root {
     color-scheme: dark;
   }

   body {
     background-color: var(--p-surface-950, #09090b);
     color: var(--p-surface-0, #ffffff);
   }
   ```
   Key: `@layer tailwind-base, primevue` tells Tailwind to put PrimeVue styles after Tailwind base — this is the PrimeVue 4 + Tailwind 4 integration pattern.

9. **Create `frontend/src/App.vue`** (shell with sidebar navigation using PrimeVue + Vue Router):
   ```vue
   <script setup lang="ts">
   import { useRouter, useRoute } from 'vue-router'
   import Toast from 'primevue/toast'
   import { LayoutDashboard, ListTodo, History, Settings } from 'lucide-vue-next'

   const router = useRouter()
   const route = useRoute()

   const navItems = [
     { to: '/dashboard', icon: LayoutDashboard, label: 'Dashboard' },
     { to: '/queue',     icon: ListTodo,        label: 'Queue' },
     { to: '/history',   icon: History,         label: 'History' },
     { to: '/settings',  icon: Settings,        label: 'Settings' },
   ]
   </script>

   <template>
     <div class="flex h-screen bg-surface-950">
       <!-- Sidebar -->
       <aside class="w-52 bg-surface-900 border-r border-surface-800 flex flex-col shrink-0">
         <div class="p-4 border-b border-surface-800">
           <span class="text-sm font-bold text-nexus">nexusOrchestrator</span>
         </div>
         <nav class="flex-1 p-2 space-y-1">
           <button
             v-for="item in navItems"
             :key="item.to"
             class="w-full flex items-center gap-2 px-3 py-2 rounded text-sm transition-colors"
             :class="route.path === item.to
               ? 'bg-nexus text-white'
               : 'text-surface-400 hover:text-surface-100 hover:bg-surface-800'"
             @click="router.push(item.to)"
           >
             <component :is="item.icon" :size="16" />
             {{ item.label }}
           </button>
         </nav>
       </aside>

       <!-- Page content -->
       <main class="flex-1 overflow-auto">
         <RouterView />
       </main>

       <!-- PrimeVue global toast outlet -->
       <Toast position="bottom-right" />
     </div>
   </template>
   ```

10. **Create `frontend/src/types/domain.ts`** — shared TypeScript interfaces:
    ```typescript
    export interface Task {
      id: string
      projectPath: string
      prompt: string
      status: 'queued' | 'processing' | 'completed' | 'failed'
      createdAt: string
      updatedAt: string
      retryCount: number
      sourceProjectPath?: string
      sourceTaskId?: string
      sourcePlanId?: string
    }

    export interface Message {
      role: 'user' | 'assistant'
      content: string
    }

    export interface Provider {
      name: string
      baseURL: string
      available: boolean
      models: string[]
    }

    export interface Stats {
      queueDepth: number
      activeTask: string
      providerCount: number
    }
    ```

11. **Create `frontend/src/composables/useNexus.ts`** — thin wrapper around Wails Go bindings:
    ```typescript
    import {
      SubmitTask, GetTask, GetQueue, GetAllTasks,
      GetProviders, CancelTask, GetSession, ClearSession, GetStats
    } from '../wailsjs/go/main/App'
    import type { Task, Message, Provider, Stats } from '../types/domain'

    export function useNexus() {
      return {
        submitTask: (projectPath: string, prompt: string) =>
          SubmitTask(projectPath, prompt) as Promise<string>,
        getTask: (id: string) =>
          GetTask(id) as Promise<Task>,
        getQueue: () =>
          GetQueue() as Promise<Task[]>,
        getAllTasks: (status: string) =>
          GetAllTasks(status) as Promise<Task[]>,
        getProviders: () =>
          GetProviders() as Promise<Provider[]>,
        cancelTask: (id: string) =>
          CancelTask(id) as Promise<void>,
        getSession: (projectPath: string) =>
          GetSession(projectPath) as Promise<Message[]>,
        clearSession: (projectPath: string) =>
          ClearSession(projectPath) as Promise<void>,
        getStats: () =>
          GetStats() as Promise<Stats>,
      }
    }
    ```

12. **Create `frontend/src/wailsjs/go/main/App.d.ts`** (TypeScript declarations for Go bindings — must match `app.go` methods after TASK-018):
    ```typescript
    export function SubmitTask(projectPath: string, prompt: string): Promise<string>
    export function GetTask(id: string): Promise<import('../../../types/domain').Task>
    export function GetQueue(): Promise<import('../../../types/domain').Task[]>
    export function GetAllTasks(status: string): Promise<import('../../../types/domain').Task[]>
    export function GetProviders(): Promise<import('../../../types/domain').Provider[]>
    export function CancelTask(id: string): Promise<void>
    export function GetSession(projectPath: string): Promise<import('../../../types/domain').Message[]>
    export function ClearSession(projectPath: string): Promise<void>
    export function GetStats(): Promise<import('../../../types/domain').Stats>
    ```

13. **Create `frontend/src/wailsjs/runtime.d.ts`**:
    ```typescript
    export declare function EventsOn(eventName: string, callback: (...data: unknown[]) => void): () => void
    export declare function EventsOff(...eventNames: string[]): void
    export declare function EventsEmit(eventName: string, ...data: unknown[]): void
    export declare function WindowMinimise(): void
    export declare function WindowMaximise(): void
    export declare function WindowClose(): void
    export declare function WindowSetTitle(title: string): void
    ```

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/... .` exits 0
- [ ] `cd frontend && npm install && npm run build` exits 0, producing `frontend/dist/` with Vue-bundled assets
- [ ] `cd frontend && npm run type-check` exits 0 (no TypeScript errors)
- [ ] `wails dev` starts without errors (Vite dev server on `:5173`)
- [ ] `frontend/dist/assets/*.js` contains bundled Vue app (not React)
- [ ] `frontend/src/index.css` uses `@import "tailwindcss"` (Tailwind 4 style) — NOT `@tailwind base`
- [ ] PrimeVue `Toast` component renders in `App.vue` (confirms PrimeVue setup works)
- [ ] NO `tailwind.config.js` or `postcss.config.js` — Tailwind 4 is CSS-first

## Anti-patterns to Avoid

- NEVER use `@tailwind base/components/utilities` directives (Tailwind v3 syntax) — use `@import "tailwindcss"` (v4)
- NEVER create `tailwind.config.js` — Tailwind 4 config belongs in `index.css` under `@theme {}`
- NEVER use React components or hooks — this is a Vue 3 project
- NEVER use Vue Options API — use `<script setup>` Composition API throughout
- NEVER put `frontend/index.html` inside `frontend/dist/` — that is build output
- NEVER import Wails bindings with path aliases that Vite cannot resolve at build time
