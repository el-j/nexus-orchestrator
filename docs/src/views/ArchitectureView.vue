<!--
  Architecture page: the hexagonal layout, ports and adapters, rendered from docs/architecture.md.
-->
<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
    <div class="grid grid-cols-1 lg:grid-cols-4 gap-12">
      <!-- Sidebar TOC -->
      <aside class="lg:col-span-1">
        <div class="sticky top-24 space-y-1">
          <h4 class="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-4">
            On this page
          </h4>
          <nav class="space-y-1">
            <a
              v-for="item in toc"
              :key="item.id"
              :href="`#${item.id}`"
              @click.prevent="scrollToId(item.id)"
              class="block text-sm text-slate-500 hover:text-violet-400 transition-colors py-0.5 cursor-pointer"
              >{{ item.label }}</a
            >
          </nav>
        </div>
      </aside>

      <!-- Main content -->
      <main class="lg:col-span-3 space-y-16">
        <!-- Hero -->
        <div class="reveal">
          <div
            class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-violet-500/30 bg-violet-500/5 text-sm text-violet-300 mb-4"
          >
            <UiIcon name="sitemap" class="text-xs" />
            Ports &amp; Adapters
          </div>
          <h1 class="text-4xl font-black mb-4"><span class="gradient-text">Architecture</span></h1>
          <p class="text-lg text-slate-400">
            nexus-orchestrator follows hexagonal architecture with strict inward dependency rules.
          </p>
        </div>

        <!-- Hexagonal Architecture -->
        <section id="hexagonal" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> Hexagonal Architecture
          </h2>
          <p class="text-slate-400 text-sm mb-6">
            The core business logic never imports adapter code — all external concerns are
            abstracted behind interfaces.
          </p>
          <div class="rounded-xl border border-white/8 bg-[#0a0a10] p-6 overflow-x-auto mb-6">
            <pre class="!border-0 !bg-transparent !p-0 text-xs text-slate-400">{{
              fullDiagram
            }}</pre>
          </div>
          <p class="text-slate-400 text-sm">
            The dependency flow is strictly <strong class="text-white">inward</strong>:
          </p>
          <div class="rounded-xl border border-white/8 bg-[#0a0a10] p-4 mt-3">
            <pre class="!border-0 !bg-transparent !p-0 text-xs text-violet-300">
inbound adapters → core services → ports ← outbound adapters</pre>
          </div>
        </section>

        <!-- Domain Layer -->
        <section id="domain" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> Domain Layer
          </h2>
          <p class="text-slate-400 text-sm mb-6">Pure Go types with no framework imports.</p>

          <h3 class="text-lg font-bold mb-3 text-slate-200">Task Entity</h3>
          <p class="text-slate-500 text-sm mb-3">
            The central entity representing a single unit of AI work:
          </p>
          <div class="overflow-x-auto rounded-xl border border-white/8">
            <table class="w-full text-sm">
              <thead>
                <tr class="bg-[#0d0d14] text-left">
                  <th class="px-4 py-3 text-slate-400 font-semibold">Field</th>
                  <th class="px-4 py-3 text-slate-400 font-semibold">Type</th>
                  <th class="px-4 py-3 text-slate-400 font-semibold">Description</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="(row, i) in taskFields"
                  :key="row.field"
                  :class="i % 2 === 0 ? 'bg-[#0a0a10]' : 'bg-[#0d0d14]'"
                >
                  <td class="px-4 py-2.5 font-mono text-violet-300 text-xs">{{ row.field }}</td>
                  <td class="px-4 py-2.5 font-mono text-cyan-400 text-xs">{{ row.type }}</td>
                  <td class="px-4 py-2.5 text-slate-400 text-xs">{{ row.desc }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- TaskStatus Lifecycle -->
        <section id="status-lifecycle" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> TaskStatus Lifecycle
          </h2>
          <div class="rounded-xl border border-white/8 bg-[#0a0a10] p-6 mb-6 overflow-x-auto">
            <pre class="!border-0 !bg-transparent !p-0 text-xs text-slate-400">{{
              statusDiagram
            }}</pre>
          </div>

          <div class="overflow-x-auto rounded-xl border border-white/8">
            <table class="w-full text-sm">
              <thead>
                <tr class="bg-[#0d0d14] text-left">
                  <th class="px-4 py-3 text-slate-400 font-semibold">Status</th>
                  <th class="px-4 py-3 text-slate-400 font-semibold">Description</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="(row, i) in statusRows"
                  :key="row.status"
                  :class="i % 2 === 0 ? 'bg-[#0a0a10]' : 'bg-[#0d0d14]'"
                >
                  <td class="px-4 py-2.5">
                    <span
                      :class="`inline-block px-2 py-0.5 rounded text-xs font-mono font-bold ${row.color}`"
                      >{{ row.status }}</span
                    >
                  </td>
                  <td class="px-4 py-2.5 text-slate-400 text-xs">{{ row.desc }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- CommandType -->
        <section id="command-type" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> CommandType
          </h2>
          <div class="overflow-x-auto rounded-xl border border-white/8">
            <table class="w-full text-sm">
              <thead>
                <tr class="bg-[#0d0d14] text-left">
                  <th class="px-4 py-3 text-slate-400 font-semibold">Command</th>
                  <th class="px-4 py-3 text-slate-400 font-semibold">Description</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="(row, i) in commandRows"
                  :key="row.cmd"
                  :class="i % 2 === 0 ? 'bg-[#0a0a10]' : 'bg-[#0d0d14]'"
                >
                  <td class="px-4 py-2.5 font-mono text-violet-300 text-xs">{{ row.cmd }}</td>
                  <td class="px-4 py-2.5 text-slate-400 text-xs">{{ row.desc }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- Port Contracts -->
        <section id="ports" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> Port Contracts
          </h2>
          <p class="text-slate-500 text-sm mb-6">
            All external dependencies are abstracted behind Go interfaces in
            <code class="text-slate-300">internal/core/ports/ports.go</code>.
          </p>

          <div class="space-y-6">
            <div
              v-for="port in ports"
              :key="port.name"
              class="rounded-xl border border-white/8 bg-[#0d0d14] overflow-hidden"
            >
              <div
                class="px-4 py-3 bg-[#14141f] border-b border-white/5 flex items-center justify-between"
              >
                <span class="font-mono text-sm font-bold text-violet-300">{{ port.name }}</span>
                <span class="text-xs text-slate-500">{{ port.direction }} Port</span>
              </div>
              <div class="p-4">
                <p class="text-xs text-slate-500 mb-3">{{ port.desc }}</p>
                <CodeBlock language="go" :code="port.code" />
              </div>
            </div>
          </div>
        </section>

        <!-- Adapters -->
        <section id="adapters" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> Adapters
          </h2>

          <h3 class="text-lg font-bold mb-3 text-slate-200">Inbound Adapters</h3>
          <div class="overflow-x-auto rounded-xl border border-white/8 mb-8">
            <table class="w-full text-sm">
              <thead>
                <tr class="bg-[#0d0d14] text-left">
                  <th class="px-4 py-3 text-slate-400 font-semibold">Adapter</th>
                  <th class="px-4 py-3 text-slate-400 font-semibold">Package</th>
                  <th class="px-4 py-3 text-slate-400 font-semibold">Protocol</th>
                  <th class="px-4 py-3 text-slate-400 font-semibold">Default Port</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="(row, i) in inboundAdapters"
                  :key="row.adapter"
                  :class="i % 2 === 0 ? 'bg-[#0a0a10]' : 'bg-[#0d0d14]'"
                >
                  <td class="px-4 py-2.5 font-semibold text-white text-xs">{{ row.adapter }}</td>
                  <td class="px-4 py-2.5 font-mono text-cyan-400 text-xs">{{ row.pkg }}</td>
                  <td class="px-4 py-2.5 text-slate-400 text-xs">{{ row.proto }}</td>
                  <td class="px-4 py-2.5 font-mono text-violet-300 text-xs">{{ row.port }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <h3 class="text-lg font-bold mb-3 text-slate-200">Outbound Adapters</h3>
          <div class="overflow-x-auto rounded-xl border border-white/8">
            <table class="w-full text-sm">
              <thead>
                <tr class="bg-[#0d0d14] text-left">
                  <th class="px-4 py-3 text-slate-400 font-semibold">Adapter</th>
                  <th class="px-4 py-3 text-slate-400 font-semibold">Package</th>
                  <th class="px-4 py-3 text-slate-400 font-semibold">Backend</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="(row, i) in outboundAdapters"
                  :key="row.adapter"
                  :class="i % 2 === 0 ? 'bg-[#0a0a10]' : 'bg-[#0d0d14]'"
                >
                  <td class="px-4 py-2.5 font-semibold text-white text-xs">{{ row.adapter }}</td>
                  <td class="px-4 py-2.5 font-mono text-cyan-400 text-xs">{{ row.pkg }}</td>
                  <td class="px-4 py-2.5 text-slate-400 text-xs">{{ row.backend }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- Concurrency -->
        <section id="concurrency" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> Concurrency Model
          </h2>
          <div class="space-y-3">
            <div
              v-for="point in concurrencyPoints"
              :key="point"
              class="flex items-start gap-3 rounded-xl border border-white/5 bg-[#0d0d14] p-4"
            >
              <UiIcon name="check-circle" class="text-emerald-500 text-sm mt-0.5 flex-shrink-0" />
              <p class="text-sm text-slate-400" v-html="point"></p>
            </div>
          </div>
        </section>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import UiIcon from '../ui/UiIcon.vue';
import CodeBlock from '../components/CodeBlock.vue';
import { scrollToId } from '../utils/scroll';
import {
  commandRows,
  concurrencyPoints,
  inboundAdapters,
  outboundAdapters,
  ports,
  statusRows,
  taskFields,
} from '../data/architecture';

const toc = [
  { id: 'hexagonal', label: 'Hexagonal Architecture' },
  { id: 'domain', label: 'Domain Layer' },
  { id: 'status-lifecycle', label: 'Status Lifecycle' },
  { id: 'command-type', label: 'CommandType' },
  { id: 'ports', label: 'Port Contracts' },
  { id: 'adapters', label: 'Adapters' },
  { id: 'concurrency', label: 'Concurrency Model' },
];

const fullDiagram = `┌───────────────────────────────────────────────────────────┐
│                     Inbound Adapters                       │
│  HTTP API (chi)  │  MCP (JSON-RPC)  │  CLI  │  Wails GUI │
├───────────────────────────────────────────────────────────┤
│                      Core Services                         │
│         OrchestratorService   │   DiscoveryService         │
├───────────────────────────────────────────────────────────┤
│                     Port Interfaces                        │
│  Orchestrator │ LLMClient │ TaskRepo │ FileWriter │ Sess  │
├───────────────────────────────────────────────────────────┤
│                    Outbound Adapters                       │
│  LM Studio │ Ollama │ OpenAI │ Anthropic │ SQLite  │  FS  │
└───────────────────────────────────────────────────────────┘`;

const statusDiagram = `          ┌──────────┐
          │  QUEUED   │
          └────┬─────┘
               │
          ┌────▼─────┐
          │PROCESSING │
          └────┬─────┘
               │
    ┌──────────┼──────────┬────────────┐
    ▼          ▼          ▼            ▼
COMPLETED   FAILED   TOO_LARGE   NO_PROVIDER

              CANCELLED (from QUEUED only)`;
</script>
