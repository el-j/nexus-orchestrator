<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
    <div class="grid grid-cols-1 lg:grid-cols-4 gap-12">
      <!-- Sidebar TOC -->
      <aside class="lg:col-span-1">
        <div class="sticky top-24 space-y-1">
          <h4 class="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-4">On this page</h4>
          <nav class="space-y-1">
            <a
              v-for="item in toc"
              :key="item.id"
              :href="`#${item.id}`"
              class="block text-sm text-slate-500 hover:text-violet-400 transition-colors py-0.5"
            >{{ item.label }}</a>
          </nav>
        </div>
      </aside>

      <!-- Main content -->
      <main class="lg:col-span-3 space-y-16">
        <!-- Hero -->
        <div class="reveal">
          <div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-violet-500/30 bg-violet-500/5 text-sm text-violet-300 mb-4">
            <i class="pi pi-book text-xs"></i>
            Step-by-step guide
          </div>
          <h1 class="text-4xl font-black mb-4">Getting <span class="gradient-text">Started</span></h1>
          <p class="text-lg text-slate-400">Be up and running with nexus-orchestrator in under 5 minutes.</p>
        </div>

        <!-- Prerequisites -->
        <section id="prerequisites" class="reveal">
          <h2 class="text-2xl font-black mb-6 flex items-center gap-2">
            <span class="text-violet-400">#</span> Prerequisites
          </h2>
          <div class="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6">
            <div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4">
              <div class="text-lg mb-2">🔷</div>
              <div class="font-bold text-white text-sm mb-1">Go 1.24+</div>
              <div class="text-xs text-slate-500">With CGO_ENABLED=1 and a C compiler (gcc/clang)</div>
            </div>
            <div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4">
              <div class="text-lg mb-2">🗄️</div>
              <div class="font-bold text-white text-sm mb-1">C Compiler</div>
              <div class="text-xs text-slate-500">Required by go-sqlite3 (gcc or clang)</div>
            </div>
            <div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4">
              <div class="text-lg mb-2">🤖</div>
              <div class="font-bold text-white text-sm mb-1">LLM Provider</div>
              <div class="text-xs text-slate-500">LM Studio, Ollama, or cloud API keys</div>
            </div>
          </div>
          <div class="rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 text-sm text-amber-300">
            <strong>Provider options:</strong>
            <ul class="mt-2 space-y-1 list-disc list-inside text-amber-400/80">
              <li><a href="https://lmstudio.ai/" target="_blank" rel="noopener" class="underline">LM Studio</a> running on 127.0.0.1:1234</li>
              <li><a href="https://ollama.ai/" target="_blank" rel="noopener" class="underline">Ollama</a> running on 127.0.0.1:11434</li>
              <li>Cloud API keys for OpenAI, Anthropic, or GitHub Copilot</li>
            </ul>
          </div>
        </section>

        <!-- Installation -->
        <section id="installation" class="reveal">
          <h2 class="text-2xl font-black mb-6 flex items-center gap-2">
            <span class="text-violet-400">#</span> Installation
          </h2>
          <CodeBlock language="bash" :code="codeInstall" />
        </section>

        <!-- Starting the Daemon -->
        <section id="starting-daemon" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> Starting the Daemon
          </h2>
          <CodeBlock language="bash" :code="codeDaemonBasic" />

          <h3 class="text-lg font-bold mt-8 mb-3 text-slate-200">Custom Configuration</h3>
          <CodeBlock language="bash" :code="codeDaemonCustom" />

          <h3 class="text-lg font-bold mt-8 mb-3 text-slate-200">Cloud Provider Configuration</h3>
          <CodeBlock language="bash" :code="codeDaemonCloud" />
        </section>

        <!-- First Task -->
        <section id="first-task" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> Submitting Your First Task
          </h2>
          <CodeBlock language="bash" :code="codeFirstTask" />
          <p class="text-slate-500 text-sm mt-3 mb-2">Response:</p>
          <CodeBlock language="json" :code="codeFirstTaskResponse" />
        </section>

        <!-- Task Management -->
        <section id="task-management" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> Task Management
          </h2>

          <h3 class="text-lg font-bold mb-3 text-slate-200">Checking Task Status</h3>
          <CodeBlock language="bash" :code="codeStatus" />

          <h3 class="text-lg font-bold mt-6 mb-3 text-slate-200">Cancelling a Task</h3>
          <CodeBlock language="bash" :code="codeCancel" />
        </section>

        <!-- Providers -->
        <section id="providers" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> Managing Providers at Runtime
          </h2>
          <CodeBlock language="bash" :code="codeProviders" />
        </section>

        <!-- Dashboard -->
        <section id="dashboard" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> Using the Dashboard
          </h2>
          <div class="rounded-xl border border-white/8 bg-[#0d0d14] p-6">
            <p class="text-slate-400 text-sm mb-3">
              Open <code class="text-violet-300">http://localhost:9999/ui</code> in your browser for a live dashboard that:
            </p>
            <ul class="space-y-2 list-none p-0">
              <li v-for="feature in dashboardFeatures" :key="feature" class="flex items-start gap-2 text-sm text-slate-400">
                <i class="pi pi-check-circle text-emerald-500 text-xs mt-0.5 flex-shrink-0"></i>
                {{ feature }}
              </li>
            </ul>
          </div>
        </section>

        <!-- Testing -->
        <section id="testing" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> Running Tests
          </h2>
          <CodeBlock language="bash" :code="codeTesting" />
        </section>

        <!-- Next Steps -->
        <section id="next-steps" class="reveal">
          <h2 class="text-2xl font-black mb-6 flex items-center gap-2">
            <span class="text-violet-400">#</span> Next Steps
          </h2>
          <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <RouterLink
              v-for="next in nextSteps"
              :key="next.to"
              :to="next.to"
              class="rounded-xl border border-white/8 bg-[#0d0d14] hover:border-violet-500/30 p-5 transition-all group"
            >
              <div class="text-xl mb-2">{{ next.icon }}</div>
              <div class="font-bold text-white text-sm mb-1 group-hover:text-violet-300 transition-colors">{{ next.title }}</div>
              <div class="text-xs text-slate-500">{{ next.desc }}</div>
            </RouterLink>
          </div>
        </section>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
import CodeBlock from '../components/CodeBlock.vue'

const toc = [
  { id: 'prerequisites', label: 'Prerequisites' },
  { id: 'installation', label: 'Installation' },
  { id: 'starting-daemon', label: 'Starting the Daemon' },
  { id: 'first-task', label: 'Your First Task' },
  { id: 'task-management', label: 'Task Management' },
  { id: 'providers', label: 'Managing Providers' },
  { id: 'dashboard', label: 'Using the Dashboard' },
  { id: 'testing', label: 'Running Tests' },
  { id: 'next-steps', label: 'Next Steps' },
]

const codeInstall = `# Clone the repository
git clone https://github.com/el-j/nexus-orchestrator.git
cd nexus-orchestrator

# Build all binaries
CGO_ENABLED=1 go build ./...

# Or build specific binaries
CGO_ENABLED=1 go build -o nexus-daemon ./cmd/nexus-daemon/...
CGO_ENABLED=1 go build -o nexus-cli ./cmd/nexus-cli/...`

const codeDaemonBasic = `# Start with default settings
./nexus-daemon
# HTTP API:   http://127.0.0.1:9999
# MCP server: http://127.0.0.1:9998/mcp
# Dashboard:  http://127.0.0.1:9999/ui`

const codeDaemonCustom = `# Use environment variables for custom settings
NEXUS_DB_PATH=/path/to/nexus.db \\
NEXUS_LISTEN_ADDR=:8080 \\
NEXUS_MCP_ADDR=:8081 \\
./nexus-daemon`

const codeDaemonCloud = `# OpenAI
NEXUS_OPENAI_API_KEY=sk-... NEXUS_OPENAI_MODEL=gpt-4o-mini ./nexus-daemon

# Anthropic
NEXUS_ANTHROPIC_API_KEY=sk-ant-... NEXUS_ANTHROPIC_MODEL=claude-3-5-sonnet-20241022 ./nexus-daemon

# GitHub Copilot
NEXUS_GITHUBCOPILOT_TOKEN=ghu_... NEXUS_GITHUBCOPILOT_MODEL=gpt-4o ./nexus-daemon`

const codeFirstTask = `# Submit a code-generation task
curl -s -X POST http://localhost:9999/api/tasks \\
  -H "Content-Type: application/json" \\
  -d '{
    "projectPath": "'$PWD'",
    "targetFile": "hello.go",
    "instruction": "Write a Go function that returns Hello World"
  }' | jq .`

const codeFirstTaskResponse = `{
  "id": "a1b2c3d4-...",
  "projectPath": "/path/to/project",
  "targetFile": "hello.go",
  "instruction": "Write a Go function...",
  "status": "QUEUED",
  "createdAt": "2025-01-01T00:00:00Z"
}`

const codeStatus = `# Get task by ID
curl -s http://localhost:9999/api/tasks/TASK_ID | jq .

# List all pending tasks
curl -s http://localhost:9999/api/tasks | jq .`

const codeCancel = `curl -X DELETE http://localhost:9999/api/tasks/TASK_ID
# Returns 204 No Content on success`

const codeProviders = `# List all providers
curl -s http://localhost:9999/api/providers | jq .

# Register a new cloud provider
curl -s -X POST http://localhost:9999/api/providers \\
  -H "Content-Type: application/json" \\
  -d '{
    "name": "My OpenAI",
    "kind": "openai-compat",
    "baseURL": "https://api.openai.com/v1",
    "apiKey": "sk-...",
    "model": "gpt-4o-mini"
  }' | jq .

# Remove a provider
curl -X DELETE http://localhost:9999/api/providers/My%20OpenAI`

const codeTesting = `# Full test suite with race detection
CGO_ENABLED=1 go test -race ./...

# Service tests only
CGO_ENABLED=1 go test ./internal/core/services/...

# Lint
go vet ./...`

const dashboardFeatures = [
  'Shows all tasks with real-time status updates via SSE',
  'Allows submitting new tasks directly',
  'Displays provider status and model information',
  'Auto-refreshes every 2 seconds',
]

const nextSteps = [
  { icon: '📡', title: 'API Reference', desc: 'Full HTTP and MCP endpoint docs', to: '/api-reference' },
  { icon: '🔌', title: 'MCP Integration', desc: 'Connect with Claude Desktop', to: '/mcp-integration' },
  { icon: '🏗️', title: 'Architecture', desc: 'Understand the hexagonal design', to: '/architecture' },
]
</script>
