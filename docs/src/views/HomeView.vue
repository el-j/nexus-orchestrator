<!--
  Home page: what Nexus Orchestrator is, its main capabilities and the terminal demo.
-->
<template>
  <div>
    <!-- ── HERO ──────────────────────────────────────────── -->
    <section class="relative overflow-hidden min-h-[90vh] flex items-center">
      <!-- Background mesh gradient -->
      <div class="absolute inset-0 -z-10">
        <div
          class="absolute inset-0 bg-gradient-to-br from-[#050508] via-[#0d0814] to-[#050508]"
        ></div>
        <div
          class="absolute top-1/4 left-1/4 w-96 h-96 bg-violet-600/10 rounded-full blur-3xl"
        ></div>
        <div
          class="absolute bottom-1/4 right-1/4 w-64 h-64 bg-cyan-500/8 rounded-full blur-3xl"
        ></div>
        <!-- Grid pattern -->
        <div
          class="absolute inset-0 opacity-[0.03]"
          style="
            background-image:
              linear-gradient(rgba(255, 255, 255, 0.1) 1px, transparent 1px),
              linear-gradient(90deg, rgba(255, 255, 255, 0.1) 1px, transparent 1px);
            background-size: 50px 50px;
          "
        ></div>
      </div>

      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-20 w-full">
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-12 items-center">
          <!-- Left: Copy -->
          <div>
            <!-- Badge -->
            <div
              class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-violet-500/30 bg-violet-500/5 text-sm text-violet-300 mb-6"
            >
              <span class="w-1.5 h-1.5 rounded-full bg-violet-400 animate-pulse"></span>
              Open Source · MIT · Go 1.27+
            </div>

            <h1 class="text-4xl sm:text-5xl xl:text-6xl font-black leading-tight mb-6">
              Run AI tasks on<br />
              <span class="gradient-text">any model</span> —<br />
              <span class="text-slate-300">and remember the project.</span>
            </h1>

            <p class="text-lg text-slate-400 leading-relaxed mb-8 max-w-lg">
              Nexus runs coding tasks on the models you already have — local, API, or your Claude
              and Copilot subscriptions — streams the result while it is written, verifies it, and
              remembers each project. Agents that work on their own report through the same MCP
              tools, so one place knows what happened.
            </p>

            <div class="flex flex-wrap gap-3 mb-8">
              <RouterLink
                to="/downloads"
                class="inline-flex items-center gap-2 px-5 py-2.5 rounded-lg bg-violet-600 hover:bg-violet-500 text-white font-semibold text-sm transition-all glow-purple hover:scale-105"
              >
                <UiIcon name="download" class="text-xs" />
                Download free
              </RouterLink>
              <a
                href="https://github.com/el-j/nexus-orchestrator"
                target="_blank"
                rel="noopener"
                class="inline-flex items-center gap-2 px-5 py-2.5 rounded-lg border border-white/10 hover:border-violet-500/40 text-slate-300 hover:text-white font-semibold text-sm transition-all"
              >
                <UiIcon name="github" class="text-xs" />
                View on GitHub
              </a>
              <RouterLink
                to="/getting-started"
                class="inline-flex items-center gap-2 px-5 py-2.5 rounded-lg text-slate-400 hover:text-violet-400 font-semibold text-sm transition-all"
              >
                Get Started <UiIcon name="arrow-right" class="text-xs" />
              </RouterLink>
            </div>

            <!-- Trust badges -->
            <div class="flex flex-wrap gap-4 text-xs text-slate-500">
              <span class="flex items-center gap-1"
                ><UiIcon name="check-circle" class="text-emerald-500" /> Open Source</span
              >
              <span class="flex items-center gap-1"
                ><UiIcon name="check-circle" class="text-emerald-500" /> Zero telemetry</span
              >
              <span class="flex items-center gap-1"
                ><UiIcon name="check-circle" class="text-emerald-500" /> Local-first</span
              >
              <span class="flex items-center gap-1"
                ><UiIcon name="check-circle" class="text-emerald-500" /> MIT License</span
              >
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
          Everything you need to<br /><span class="gradient-text">orchestrate AI at scale</span>
        </h2>
        <p class="text-slate-500 max-w-2xl mx-auto">
          From local LLMs to cloud providers, single tasks to complex pipelines — nexus-orchestrator
          handles the routing, memory, and orchestration so you can focus on building.
        </p>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        <div
          v-for="(feat, i) in features"
          :key="feat.title"
          :style="{ transitionDelay: `${i * 60}ms` }"
          class="reveal rounded-xl border border-white/5 bg-[#0d0d14] hover:border-violet-500/30 hover:bg-[#14141f] p-6 transition-all group cursor-default"
        >
          <div class="text-2xl mb-3">{{ feat.icon }}</div>
          <h3
            class="font-bold text-sm mb-2 text-white group-hover:text-violet-300 transition-colors"
          >
            {{ feat.title }}
          </h3>
          <p class="text-xs text-slate-500 leading-relaxed">{{ feat.desc }}</p>
        </div>
      </div>
    </section>

    <!-- ── PROVIDERS ─────────────────────────────────────── -->
    <section class="py-16 bg-[#0d0d14] border-y border-white/5">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="text-center mb-10 reveal">
          <h2 class="text-2xl font-black mb-2">
            Works with the <span class="gradient-text">models you already use</span>
          </h2>
          <p class="text-sm text-slate-500">
            Local or cloud — any OpenAI-compatible endpoint supported
          </p>
        </div>
        <div class="flex flex-wrap justify-center gap-3 reveal">
          <span
            v-for="p in providers"
            :key="p"
            class="px-4 py-2 rounded-full border border-white/10 text-sm text-slate-400 hover:border-violet-500/40 hover:text-white transition-all bg-[#14141f]"
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
          <h2 class="text-3xl font-black mb-6">
            Up and running <span class="gradient-text">in 60 seconds</span>
          </h2>
          <ol class="space-y-6 list-none p-0">
            <li v-for="(step, i) in quickSteps" :key="i" class="flex gap-4">
              <span
                class="flex-shrink-0 w-7 h-7 rounded-full bg-violet-600/20 border border-violet-500/30 text-violet-300 text-xs font-bold flex items-center justify-center"
                >{{ i + 1 }}</span
              >
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
            Full installation guide <UiIcon name="arrow-right" class="text-xs" />
          </RouterLink>
        </div>

        <div class="reveal space-y-3">
          <UiTabs
            :tabs="['Start', 'Connect agents', 'Submit Task', 'MCP (JSON-RPC)']"
            label="Quick start"
          >
            <template #panel-0>
              <CodeBlock language="bash" :code="codeStart" />
            </template>
            <template #panel-1>
              <CodeBlock language="bash" :code="codeConnect" />
            </template>
            <template #panel-2>
              <CodeBlock language="bash" :code="codeSubmit" />
            </template>
            <template #panel-3>
              <CodeBlock language="json" :code="codeMcp" />
            </template>
          </UiTabs>
        </div>
      </div>
    </section>

    <!-- ── ARCH TEASER ───────────────────────────────────── -->
    <section class="py-16 bg-[#0d0d14] border-y border-white/5">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-10 items-center">
          <div class="reveal">
            <h2 class="text-2xl font-black mb-4">
              Hexagonal architecture, <span class="gradient-text">ports &amp; adapters</span>
            </h2>
            <p class="text-slate-500 text-sm mb-6">
              The core business logic never imports adapter code. All external concerns — LLMs,
              SQLite, filesystem — are abstracted behind interfaces for maximum testability and
              replaceability.
            </p>
            <RouterLink
              to="/architecture"
              class="inline-flex items-center gap-2 text-sm text-violet-400 hover:text-violet-300 font-semibold transition-colors"
            >
              Explore the architecture <UiIcon name="arrow-right" class="text-xs" />
            </RouterLink>
          </div>
          <div
            class="reveal font-mono text-xs text-slate-500 bg-[#0a0a10] rounded-xl border border-white/5 p-6"
          >
            <pre class="!border-0 !bg-transparent !p-0 text-xs">{{ archDiagram }}</pre>
          </div>
        </div>
      </div>
    </section>

    <!-- ── CTA ───────────────────────────────────────────── -->
    <section class="py-24 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div
        class="reveal animated-border rounded-2xl bg-gradient-to-br from-[#14141f] to-[#1a1a2e] p-12 text-center"
      >
        <h2 class="text-3xl font-black mb-4">
          Ready to orchestrate your <span class="gradient-text">AI workflows?</span>
        </h2>
        <p class="text-slate-500 mb-8 max-w-xl mx-auto">
          Download the desktop app or run the headless daemon. Open source, MIT licensed, zero
          telemetry.
        </p>
        <div class="flex flex-wrap justify-center gap-4">
          <RouterLink
            to="/downloads"
            class="px-6 py-3 rounded-xl bg-violet-600 hover:bg-violet-500 text-white font-bold text-sm transition-all glow-purple hover:scale-105"
          >
            Download for free →
          </RouterLink>
          <a
            href="https://github.com/el-j/nexus-orchestrator"
            target="_blank"
            rel="noopener"
            class="px-6 py-3 rounded-xl border border-white/10 hover:border-violet-500/40 text-slate-300 hover:text-white font-bold text-sm transition-all"
          >
            <UiIcon name="github" /> View on GitHub
          </a>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import UiIcon from '../ui/UiIcon.vue';
import { RouterLink } from 'vue-router';
import UiTabs from '../ui/UiTabs.vue';
import TerminalDemo from '../components/TerminalDemo.vue';
import CodeBlock from '../components/CodeBlock.vue';

const stats = [
  { value: '10+', label: 'LLM Providers (local, API, subscriptions)' },
  { value: '43', label: 'MCP tools' },
  { value: 'Live', label: 'Streamed output' },
  { value: '1', label: 'Project brain per repo' },
  { value: '0', label: 'Telemetry' },
];

const features = [
  {
    icon: '🔀',
    title: 'Any model, one queue',
    desc: 'LM Studio, Ollama, OpenAI, Anthropic, Gemini and any OpenAI-compatible endpoint, with discovery, health checks, role-based routing and automatic failover.',
  },
  {
    icon: '🎟️',
    title: 'Your subscriptions',
    desc: 'Delegate to Claude Pro/Max through the logged-in claude CLI, and to GitHub Copilot and other editor models through the VS Code model bridge. No API key.',
  },
  {
    icon: '📡',
    title: 'Live output',
    desc: 'The reply streams to the UI, VS Code and MCP clients while the model writes it. A failing provider hands over to the next one; partial text never reaches your file.',
  },
  {
    icon: '🧠',
    title: 'Project brain',
    desc: 'SQLite FTS5 knowledge per repository, learned from CLAUDE.md, AGENTS.md, GEMINI.md and what agents ingest. Every session starts with a token-budgeted briefing.',
  },
  {
    icon: '📐',
    title: 'Rules that are enforced',
    desc: '.nexus/policy.json: default verification command, file-size limit, tests first, free-text rules. The engine enforces them on its own runs and remembers each outcome.',
  },
  {
    icon: '🗂️',
    title: 'Plans in any folder',
    desc: 'PLAN-n / TASK-n files are found anywhere in the project (or in the folders you configure) and linked to each other automatically, whichever agent wrote them.',
  },
  {
    icon: '🔌',
    title: 'MCP server, 43 tools',
    desc: 'Claude Code, Claude Desktop, Cursor, Antigravity and VS Code Copilot all use the same tools; wait_for_task streams progress. nexus-cli connect wires up the agents on your machine.',
  },
  {
    icon: '🤝',
    title: 'Agents report here',
    desc: 'Agents register, claim a task and report its status through MCP, so what you see is what happened. Hand-edited state is shown as such: a task says who ran it.',
  },
  {
    icon: '🖥️',
    title: 'Full UI, everywhere',
    desc: 'Mission Control, Agents, Providers, Projects (activity, plans, history, backlog, brain) and Settings, served by the daemon at /app/ and as a desktop app for macOS, Windows and Linux.',
  },
  {
    icon: '🧩',
    title: 'VS Code extension',
    desc: "Task queue, live output, plan view and brain commands in your editor; shares the editor models with Nexus and connects the editor's own agent (VS Code 1.101+, Antigravity).",
  },
  {
    icon: '🌐',
    title: 'REST API + CLI',
    desc: 'Everything over HTTP with an SSE event stream, plus nexus-cli, nexus-submit and a GitHub Action for CI.',
  },
  {
    icon: '🛡️',
    title: 'Safe by default',
    desc: 'Loopback only, origin and host guards, pre-flight context guard (TOO_LARGE), plan-first gate, verification with self-healing turns, zero telemetry.',
  },
];

const providers = [
  'LM Studio',
  'Ollama',
  'OpenAI',
  'Anthropic',
  'Google Gemini',
  'Claude (subscription)',
  'GitHub Copilot',
  'vLLM / LocalAI',
  'Any OpenAI-compatible',
];

const quickSteps = [
  {
    title: 'Start Nexus',
    desc: 'make start — daemon, REST on :63987, MCP on :63988, full UI at /app/ (or install the desktop app)',
  },
  {
    title: 'Connect your agents',
    desc: 'nexus-cli connect --write adds Nexus to Antigravity, Cursor and Claude Code; the VS Code extension connects its own editor',
  },
  {
    title: 'Add a provider',
    desc: 'LM Studio or Ollama are found automatically; add a cloud key, the claude CLI or the editor models',
  },
  {
    title: 'Introduce a project and submit a task',
    desc: 'make init-project PROJECT=/path/to/repo, then submit via the UI, MCP, CLI or your editor',
  },
];

const codeStart = `# From source (Node >= 22 builds the full UI)
make start
# REST      → http://localhost:63987
# Full UI   → http://localhost:63987/app/
# MCP       → http://localhost:63988/mcp

# macOS desktop app, no Gatekeeper dialog
curl -sSfL https://raw.githubusercontent.com/el-j/nexus-orchestrator/main/scripts/install.sh | sh -s -- --desktop`;

const codeConnect = `# Which agents on this machine can call Nexus? (changes nothing)
nexus-cli connect

# Connect them: Antigravity, Cursor, Claude Code
nexus-cli connect --write

# Teach Nexus a project once
make init-project PROJECT=/path/to/your/repo`;

const codeSubmit = `curl -X POST http://localhost:63987/api/tasks \\
  -H "Content-Type: application/json" \\
  -d '{
    "projectPath": "/path/to/project",
    "targetFile": "auth.go",
    "instruction": "Add JWT auth middleware",
    "command": "execute"
  }'`;

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
}`;

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
└─────────────────────────────────────┘`;
</script>
