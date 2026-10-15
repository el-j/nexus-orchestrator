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

const taskFields = [
  { field: 'ID', type: 'string', desc: 'UUID, generated on submission' },
  { field: 'ProjectPath', type: 'string', desc: 'Absolute path to the project' },
  { field: 'TargetFile', type: 'string', desc: 'Relative path for generated code output' },
  { field: 'Instruction', type: 'string', desc: 'Natural language prompt' },
  { field: 'ContextFiles', type: '[]string', desc: 'Files to include as context' },
  { field: 'ModelID', type: 'string', desc: 'Constrain to a specific model (optional)' },
  { field: 'ProviderHint', type: 'string', desc: 'Prefer a specific provider (optional)' },
  { field: 'Command', type: 'CommandType', desc: 'Task classification: plan, execute, or auto' },
  { field: 'Status', type: 'TaskStatus', desc: 'Lifecycle state' },
  { field: 'CreatedAt', type: 'time.Time', desc: 'Creation timestamp' },
  { field: 'UpdatedAt', type: 'time.Time', desc: 'Last update timestamp' },
  {
    field: 'Logs',
    type: 'string',
    desc: 'LLM output or error details; the engine writes "completed via <provider>"',
  },
  {
    field: 'RanBy',
    type: 'string',
    desc: 'Derived from Logs: the provider Nexus ran the task on, empty when an agent reported the status',
  },
  {
    field: 'Role / Priority / Tags',
    type: 'string / int / []string',
    desc: 'Routing tier hint, backlog ordering and labels',
  },
  {
    field: 'Agentic',
    type: 'bool',
    desc: 'Run as a bounded multi-turn tool loop over the project files',
  },
];

const statusRows = [
  { status: 'QUEUED', desc: 'Task is waiting in the queue', color: 'bg-blue-500/20 text-blue-300' },
  {
    status: 'PROCESSING',
    desc: 'Task is being processed by an LLM',
    color: 'bg-amber-500/20 text-amber-300',
  },
  {
    status: 'COMPLETED',
    desc: 'LLM generated output successfully',
    color: 'bg-emerald-500/20 text-emerald-300',
  },
  { status: 'FAILED', desc: 'LLM call failed', color: 'bg-red-500/20 text-red-300' },
  {
    status: 'DRAFT / BACKLOG',
    desc: 'Staged ideas that do not run until promoted',
    color: 'bg-sky-500/20 text-sky-300',
  },
  {
    status: 'CANCELLED',
    desc: 'Cancelled before processing',
    color: 'bg-slate-500/20 text-slate-300',
  },
  {
    status: 'TOO_LARGE',
    desc: 'Prompt exceeds model context window',
    color: 'bg-orange-500/20 text-orange-300',
  },
  {
    status: 'NO_PROVIDER',
    desc: 'No provider available for requested model',
    color: 'bg-purple-500/20 text-purple-300',
  },
];

const commandRows = [
  { cmd: 'plan', desc: 'Planning/orchestration work (creating plans, task documents)' },
  { cmd: 'execute', desc: 'Code implementation (requires a prior completed plan)' },
  { cmd: 'auto', desc: 'Let the orchestrator decide (default)' },
];

const ports = [
  {
    name: 'Orchestrator',
    direction: 'Inbound',
    desc: 'The primary interface that UI, CLI, and HTTP API call',
    code: `type Orchestrator interface {
    SubmitTask(task domain.Task) (string, error)
    GetTask(id string) (domain.Task, error)
    GetQueue() ([]domain.Task, error)
    GetProviders() ([]ProviderInfo, error)
    CancelTask(id string) error
    RegisterCloudProvider(cfg domain.ProviderConfig) error
    RemoveProvider(providerName string) error
    GetProviderModels(providerName string) ([]string, error)
}`,
  },
  {
    name: 'LLMClient',
    direction: 'Outbound',
    desc: 'Interface for any language model backend; every call takes a context so a cancel aborts it',
    code: `type LLMClient interface {
    Ping(ctx context.Context) bool
    ProviderName() string
    ActiveModel() string
    BaseURL() string
    GetAvailableModels(ctx context.Context) ([]string, error)
    ContextLimit(ctx context.Context) int
    GenerateCode(ctx context.Context, prompt string) (string, error)
    Chat(ctx context.Context, messages []domain.Message) (string, error)
}`,
  },
  {
    name: 'ChatStreamer',
    direction: 'Outbound (optional)',
    desc: 'A provider that can deliver its reply piece by piece; the engine publishes the pieces as task.chunk events, others send the whole reply as one chunk',
    code: `type ChatStreamer interface {
    ChatStream(ctx context.Context, messages []domain.Message,
        onChunk func(piece string)) (string, error)
}`,
  },
  {
    name: 'TaskRepository',
    direction: 'Outbound',
    desc: 'Persistence for tasks',
    code: `type TaskRepository interface {
    Save(ctx context.Context, t domain.Task) error
    GetByID(ctx context.Context, id string) (domain.Task, error)
    GetPending(ctx context.Context) ([]domain.Task, error)
    ClaimNextQueued(ctx context.Context) (domain.Task, error)
    GetAll(ctx context.Context) ([]domain.Task, error)
    UpdateStatus(ctx context.Context, id string, status domain.TaskStatus) error
    UpdateLogs(ctx context.Context, id, logs string) error
    // ... by project, by status, stale PROCESSING tasks for the watchdog
}`,
  },
  {
    name: 'SessionRepository',
    direction: 'Outbound',
    desc: 'Per-project conversation history',
    code: `type SessionRepository interface {
    Save(s domain.Session) error
    GetByProjectPath(projectPath string) (domain.Session, error)
    AppendMessage(projectPath string, msg domain.Message) error
}`,
  },
];

const inboundAdapters = [
  {
    adapter: 'HTTP API',
    pkg: 'httpapi',
    proto: 'REST + SSE (task events, live chunks)',
    port: ':63987',
  },
  {
    adapter: 'Web UI',
    pkg: 'webui',
    proto: 'The Vue UI embedded in the daemon, at /app/',
    port: ':63987',
  },
  {
    adapter: 'MCP Server',
    pkg: 'mcp',
    proto: 'JSON-RPC 2.0 (42 tools, wait_for_task streams progress)',
    port: ':63988',
  },
  {
    adapter: 'CLI Client',
    pkg: 'cli',
    proto: 'HTTP → daemon; nexus connect edits agent configs',
    port: '—',
  },
  {
    adapter: 'Origin / Host guard',
    pkg: 'httpguard',
    proto: 'Rejects foreign browser origins and DNS rebinding',
    port: '—',
  },
  {
    adapter: 'Wails GUI',
    pkg: 'app.go',
    proto: 'Native window over the same HTTP API',
    port: ':63987',
  },
];

const outboundAdapters = [
  {
    adapter: 'LM Studio',
    pkg: 'llm_lmstudio',
    backend: 'OpenAI-compatible API at :1234 (streams)',
  },
  { adapter: 'Ollama', pkg: 'llm_ollama', backend: 'Ollama API at :11434 (streams NDJSON)' },
  {
    adapter: 'OpenAI-compatible',
    pkg: 'llm_openaicompat',
    backend: 'OpenAI, GitHub Copilot, the VS Code model bridge, vLLM, LocalAI (streams SSE)',
  },
  { adapter: 'Anthropic', pkg: 'llm_anthropic', backend: 'Anthropic Messages API (streams)' },
  {
    adapter: 'Google Gemini',
    pkg: 'llm_gemini',
    backend: 'generateContent / streamGenerateContent',
  },
  {
    adapter: 'CLI subscription',
    pkg: 'llm_cli',
    backend: 'A logged-in AI CLI such as claude -p (Claude Pro/Max)',
  },
  {
    adapter: 'SSE reader',
    pkg: 'sse',
    backend: 'Shared server-sent-events parser of the streaming adapters',
  },
  { adapter: 'SQLite', pkg: 'repo_sqlite', backend: 'Local SQLite (FTS5 brain) via go-sqlite3' },
  {
    adapter: 'Filesystem',
    pkg: 'fs_writer, fs_watcher',
    backend: 'Disk write-back and project change watching',
  },
  {
    adapter: 'Activity readers',
    pkg: 'activity_*',
    backend: 'Claude Code transcripts, Continue, network activity',
  },
  { adapter: 'Scanners', pkg: 'sys_scanner', backend: 'Provider, agent and plan-file discovery' },
];

const concurrencyPoints = [
  'Shared state is protected with <code class="text-slate-300">sync.Mutex</code> in OrchestratorService',
  'The background worker goroutine processes tasks sequentially — only one LLM call is ever in flight',
  'Shutdown is coordinated via a <code class="text-slate-300">stopCh chan struct{}</code> channel',
  '<code class="text-slate-300">Stop()</code> is idempotent via <code class="text-slate-300">sync.Once</code>',
  'No goroutines inside core services — goroutine lifecycle is an infrastructure concern owned by inbound adapters',
  'SSE broadcasting uses a separate Hub with its own mutex to avoid lock nesting; a slow subscriber drops events instead of blocking the engine',
  'Every attempt publishes task.restart first, then task.chunk events (at most one per 50 ms), so a client needs one rule: restart clears, chunk appends',
];
</script>
