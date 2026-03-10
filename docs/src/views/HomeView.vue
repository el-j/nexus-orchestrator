<template>
  <div>
    <!-- ── HERO ──────────────────────────────────────────── -->
    <section class="relative overflow-hidden min-h-[90vh] flex items-center">
      <!-- Background mesh gradient -->
      <div class="absolute inset-0 -z-10">
        <div class="absolute inset-0 bg-gradient-to-br from-[#050508] via-[#0d0814] to-[#050508]"></div>
        <div class="absolute top-1/4 left-1/4 w-96 h-96 bg-violet-600/10 rounded-full blur-3xl"></div>
        <div class="absolute bottom-1/4 right-1/4 w-64 h-64 bg-cyan-500/8 rounded-full blur-3xl"></div>
        <!-- Grid pattern -->
        <div
          class="absolute inset-0 opacity-[0.03]"
          style="background-image: linear-gradient(rgba(255,255,255,.1) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,.1) 1px, transparent 1px); background-size: 50px 50px;"
        ></div>
      </div>

      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-20 w-full">
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-12 items-center">
          <!-- Left: Copy -->
          <div>
            <!-- Badge -->
            <div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-violet-500/30 bg-violet-500/5 text-sm text-violet-300 mb-6">
              <span class="w-1.5 h-1.5 rounded-full bg-violet-400 animate-pulse"></span>
              Open Source · MIT · Go 1.24+
            </div>

            <h1 class="text-4xl sm:text-5xl xl:text-6xl font-black leading-tight mb-6">
              Route AI tasks to<br>
              <span class="gradient-text">any LLM</span> —<br>
              <span class="text-slate-300">local or cloud.</span>
            </h1>

            <p class="text-lg text-slate-400 leading-relaxed mb-8 max-w-lg">
              nexusOrchestrator is a headless Go daemon with MCP server, REST API,
              and per-project session memory. Connect Claude Desktop, route to Ollama,
              LM Studio, OpenAI, or Anthropic — your AI workflows, your rules.
            </p>

            <div class="flex flex-wrap gap-3 mb-8">
              <RouterLink
                to="/downloads"
                class="inline-flex items-center gap-2 px-5 py-2.5 rounded-lg bg-violet-600 hover:bg-violet-500
                       text-white font-semibold text-sm transition-all glow-purple hover:scale-105"
              >
                <i class="pi pi-download text-xs"></i>
                Download free
              </RouterLink>
              <a
                href="https://github.com/el-j/nexusOrchestrator"
                target="_blank"
                rel="noopener"
                class="inline-flex items-center gap-2 px-5 py-2.5 rounded-lg border border-white/10
                       text-slate-300 hover:border-violet-500/40 hover:text-white font-semibold text-sm transition-all"
              >
                <i class="pi pi-github text-xs"></i>
                View on GitHub
              </a>
              <RouterLink
                to="/getting-started"
                class="inline-flex items-center gap-2 px-5 py-2.5 rounded-lg text-slate-400
                       hover:text-violet-400 font-semibold text-sm transition-all"
              >
                Get Started <i class="pi pi-arrow-right text-xs"></i>
              </RouterLink>
            </div>

            <!-- Trust badges -->
            <div class="flex flex-wrap gap-4 text-xs text-slate-500">
              <span class="flex items-center gap-1"><i class="pi pi-check-circle text-emerald-500"></i> Open Source</span>
              <span class="flex items-center gap-1"><i class="pi pi-check-circle text-emerald-500"></i> Zero telemetry</span>
              <span class="flex items-center gap-1"><i class="pi pi-check-circle text-emerald-500"></i> Local-first</span>
              <span class="flex items-center gap-1"><i class="pi pi-check-circle text-emerald-500"></i> MIT License</span>
            </div>
          </div>

          <!-- Right: Terminal Demo -->
          <div class="reveal">
            <TerminalDemo />
          </div>
        </div>
      </div>
    </section>

    <!-- ── STATS BAR ─────────────────────────────────────── -->
    <section class="border-y border-white/5 bg-[#0d0d14] py-6">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-6 text-center">
          <div v-for="stat in stats" :key="stat.label" class="py-2">
            <div class="text-2xl font-black gradient-text mb-1">{{ stat.value }}</div>
            <div class="text-xs text-slate-500">{{ stat.label }}</div>
          </div>
        </div>
      </div>
    </section>

    <!-- ── FEATURES ──────────────────────────────────────── -->
    <section class="py-24 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="text-center mb-16 reveal">
        <h2 class="text-3xl sm:text-4xl font-black mb-4">
          Everything you need to<br><span class="gradient-text">orchestrate AI at scale</span>
        </h2>
        <p class="text-slate-500 max-w-2xl mx-auto">
          From local LLMs to cloud providers, single tasks to complex pipelines — nexusOrchestrator handles the routing, memory, and orchestration so you can focus on building.
        </p>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        <div
          v-for="(feat, i) in features"
          :key="feat.title"
          :style="{ transitionDelay: `${i * 60}ms` }"
          class="reveal rounded-xl border border-white/5 bg-[#0d0d14] hover:border-violet-500/30
                 hover:bg-[#14141f] p-6 transition-all group cursor-default"
        >
          <div class="text-2xl mb-3">{{ feat.icon }}</div>
          <h3 class="font-bold text-sm mb-2 text-white group-hover:text-violet-300 transition-colors">{{ feat.title }}</h3>
          <p class="text-xs text-slate-500 leading-relaxed">{{ feat.desc }}</p>
        </div>
      </div>
    </section>

    <!-- ── PROVIDERS ─────────────────────────────────────── -->
    <section class="py-16 bg-[#0d0d14] border-y border-white/5">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="text-center mb-10 reveal">
          <h2 class="text-2xl font-black mb-2">Works with the <span class="gradient-text">models you already use</span></h2>
          <p class="text-sm text-slate-500">Local or cloud — any OpenAI-compatible endpoint supported</p>
        </div>
        <div class="flex flex-wrap justify-center gap-3 reveal">
          <span
            v-for="p in providers"
            :key="p"
            class="px-4 py-2 rounded-full border border-white/10 text-sm text-slate-400
                   hover:border-violet-500/40 hover:text-white transition-all bg-[#14141f]"
          >
            {{ p }}
          </span>
        </div>
      </div>
    </section>

    <!-- ── QUICK START ───────────────────────────────────── -->
    <section class="py-24 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-12 items-start">
        <div class="reveal">
          <h2 class="text-3xl font-black mb-6">Up and running <span class="gradient-text">in 60 seconds</span></h2>
          <ol class="space-y-6 list-none p-0">
            <li v-for="(step, i) in quickSteps" :key="i" class="flex gap-4">
              <span class="flex-shrink-0 w-7 h-7 rounded-full bg-violet-600/20 border border-violet-500/30
                           text-violet-300 text-xs font-bold flex items-center justify-center">{{ i + 1 }}</span>
              <div>
                <div class="font-semibold text-sm text-white mb-1">{{ step.title }}</div>
                <div class="text-xs text-slate-500">{{ step.desc }}</div>
              </div>
            </li>
          </ol>
          <RouterLink
            to="/getting-started"
            class="inline-flex items-center gap-2 mt-8 text-sm text-violet-400 hover:text-violet-300 transition-colors font-semibold"
          >
            Full installation guide <i class="pi pi-arrow-right text-xs"></i>
          </RouterLink>
        </div>

        <div class="reveal space-y-3">
          <Tabs value="0">
            <TabList>
              <Tab value="0">Build &amp; Start</Tab>
              <Tab value="1">Submit Task</Tab>
              <Tab value="2">MCP (JSON-RPC)</Tab>
            </TabList>
            <TabPanels>
              <TabPanel value="0">
                <CodeBlock language="bash" :code="codeStart" />
              </TabPanel>
              <TabPanel value="1">
                <CodeBlock language="bash" :code="codeSubmit" />
              </TabPanel>
              <TabPanel value="2">
                <CodeBlock language="json" :code="codeMcp" />
              </TabPanel>
            </TabPanels>
          </Tabs>
        </div>
      </div>
    </section>

    <!-- ── ARCH TEASER ───────────────────────────────────── -->
    <section class="py-16 bg-[#0d0d14] border-y border-white/5">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-10 items-center">
          <div class="reveal">
            <h2 class="text-2xl font-black mb-4">Hexagonal architecture, <span class="gradient-text">ports &amp; adapters</span></h2>
            <p class="text-slate-500 text-sm mb-6">
              The core business logic never imports adapter code. All external concerns — LLMs, SQLite, filesystem — are abstracted behind interfaces for maximum testability and replaceability.
            </p>
            <RouterLink
              to="/architecture"
              class="inline-flex items-center gap-2 text-sm text-violet-400 hover:text-violet-300 font-semibold transition-colors"
            >
              Explore the architecture <i class="pi pi-arrow-right text-xs"></i>
            </RouterLink>
          </div>
          <div class="reveal font-mono text-xs text-slate-500 bg-[#0a0a10] rounded-xl border border-white/5 p-6">
            <pre class="!border-0 !bg-transparent !p-0 text-xs">{{ archDiagram }}</pre>
          </div>
        </div>
      </div>
    </section>

    <!-- ── CTA ───────────────────────────────────────────── -->
    <section class="py-24 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="reveal animated-border rounded-2xl bg-gradient-to-br from-[#14141f] to-[#1a1a2e] p-12 text-center">
        <h2 class="text-3xl font-black mb-4">
          Ready to orchestrate your <span class="gradient-text">AI workflows?</span>
        </h2>
        <p class="text-slate-500 mb-8 max-w-xl mx-auto">
          Download the desktop app or run the headless daemon. Open source, MIT licensed, zero telemetry.
        </p>
        <div class="flex flex-wrap justify-center gap-4">
          <RouterLink
            to="/downloads"
            class="px-6 py-3 rounded-xl bg-violet-600 hover:bg-violet-500 text-white font-bold text-sm transition-all glow-purple hover:scale-105"
          >
            Download for free →
          </RouterLink>
          <a
            href="https://github.com/el-j/nexusOrchestrator"
            target="_blank"
            rel="noopener"
            class="px-6 py-3 rounded-xl border border-white/10 hover:border-violet-500/40 text-slate-300 hover:text-white font-bold text-sm transition-all"
          >
            <i class="pi pi-github"></i> View on GitHub
          </a>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
import Tabs from 'primevue/tabs'
import TabList from 'primevue/tablist'
import Tab from 'primevue/tab'
import TabPanels from 'primevue/tabpanels'
import TabPanel from 'primevue/tabpanel'
import TerminalDemo from '../components/TerminalDemo.vue'
import CodeBlock from '../components/CodeBlock.vue'

const stats = [
  { value: '6+', label: 'LLM Providers' },
  { value: '3', label: 'Interfaces (HTTP/MCP/GUI)' },
  { value: '∞', label: 'Project Sessions' },
  { value: '<1s', label: 'Context Guard' },
  { value: '0', label: 'Telemetry' },
]

const features = [
  { icon: '🔀', title: 'Multi-Backend Routing', desc: 'Auto-discovers and routes to LM Studio, Ollama, OpenAI, Anthropic, or any OpenAI-compatible endpoint. Per-task model selection with provider hints.' },
  { icon: '🧠', title: 'Session Memory', desc: 'Every project path gets its own SQLite-backed conversation history. Multi-turn context maintained automatically across tasks.' },
  { icon: '🔌', title: 'MCP Server', desc: 'Built-in JSON-RPC 2.0 server on port 9998, compatible with Claude Desktop and any MCP client. Six tools ready out of the box.' },
  { icon: '🌐', title: 'Full REST API', desc: 'Complete task management on port 9999 — submit, list, cancel, monitor. Real-time updates via Server-Sent Events.' },
  { icon: '🛡️', title: 'Context Guard', desc: 'Pre-flight token estimation prevents prompt overflow. Tasks exceeding context limits are flagged as TOO_LARGE immediately.' },
  { icon: '⚡', title: 'Smart Discovery', desc: 'Auto-detect running LLM backends with health checks and automatic failover when a provider goes down.' },
  { icon: '🎯', title: 'Command Routing', desc: 'Classify tasks as "plan" or "execute". Enforces that execution tasks have a prior plan, preventing uncoordinated changes.' },
  { icon: '🖥️', title: 'Desktop GUI', desc: 'Native Wails desktop app with embedded HTTP API, real-time task monitoring, system tray, and provider management.' },
]

const providers = ['LM Studio', 'Ollama', 'OpenAI', 'Anthropic', 'GitHub Copilot', 'Any OpenAI-compatible']

const quickSteps = [
  { title: 'Build from source or download', desc: 'CGO_ENABLED=1 go build -o nexus-daemon ./cmd/nexus-daemon/...' },
  { title: 'Start the daemon', desc: 'Run ./nexus-daemon — HTTP API on :9999, MCP on :9998' },
  { title: 'Connect your LLM provider', desc: 'LM Studio, Ollama, or set NEXUS_OPENAI_API_KEY for cloud' },
  { title: 'Submit your first task', desc: 'Via HTTP API, MCP, CLI, or Claude Desktop' },
]

const codeStart = `# Build
CGO_ENABLED=1 go build -o nexus-daemon ./cmd/nexus-daemon/...

# Start
./nexus-daemon
# HTTP API  → http://localhost:9999
# MCP       → http://localhost:9998/mcp
# Dashboard → http://localhost:9999/ui`

const codeSubmit = `curl -X POST http://localhost:9999/api/tasks \\
  -H "Content-Type: application/json" \\
  -d '{
    "projectPath": "/path/to/project",
    "targetFile": "auth.go",
    "instruction": "Add JWT auth middleware",
    "command": "execute"
  }'`

const codeMcp = `{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "submit_task",
    "arguments": {
      "projectPath": "/path/to/project",
      "targetFile": "auth.go",
      "instruction": "Add JWT auth middleware"
    }
  }
}`

const archDiagram = `┌─────────────────────────────────────┐
│         Inbound Adapters            │
│  HTTP API │ MCP (JSON-RPC) │ CLI   │
├─────────────────────────────────────┤
│           Core Services             │
│  OrchestratorService │ Discovery   │
├─────────────────────────────────────┤
│          Port Interfaces            │
│  LLMClient │ TaskRepo │ Session   │
├─────────────────────────────────────┤
│         Outbound Adapters           │
│  LM Studio │ Ollama │ SQLite      │
└─────────────────────────────────────┘`
</script>
