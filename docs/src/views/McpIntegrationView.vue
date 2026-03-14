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
              @click.prevent="scrollToId(item.id)"
              class="block text-sm text-slate-500 hover:text-violet-400 transition-colors py-0.5 cursor-pointer"
            >{{ item.label }}</a>
          </nav>
        </div>
      </aside>

      <!-- Main content -->
      <main class="lg:col-span-3 space-y-16">
        <!-- Hero -->
        <div class="reveal">
          <div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-violet-500/30 bg-violet-500/5 text-sm text-violet-300 mb-4">
            <i class="pi pi-link text-xs"></i>
            JSON-RPC 2.0 · MCP 2024-11-05
          </div>
          <h1 class="text-4xl font-black mb-4"><span class="gradient-text">MCP Integration</span></h1>
          <p class="text-lg text-slate-400">Connect nexus-orchestrator to Claude Desktop and any MCP-compatible client.</p>
        </div>

        <!-- What is MCP -->
        <section id="what-is-mcp" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> What is MCP?
          </h2>
          <div class="rounded-xl border border-white/8 bg-[#0d0d14] p-6">
            <p class="text-slate-400 text-sm leading-relaxed">
              The <a href="https://modelcontextprotocol.io/" target="_blank" rel="noopener" class="text-violet-400 hover:text-violet-300">Model Context Protocol</a> (MCP)
              is an open standard for connecting AI assistants to external tools and data sources.
              nexus-orchestrator implements an MCP server using JSON-RPC 2.0, making it compatible with Claude Desktop and any MCP-aware client.
            </p>
          </div>
        </section>

        <!-- Claude Desktop Setup -->
        <section id="claude-desktop" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> Claude Desktop Setup
          </h2>
          <p class="text-slate-400 text-sm mb-6">Add the following to your Claude Desktop configuration file:</p>

          <!-- OS Tabs -->
          <Tabs value="0">
            <TabList>
              <Tab value="0">🍎 macOS</Tab>
              <Tab value="1">🪟 Windows</Tab>
            </TabList>
            <TabPanels>
              <TabPanel value="0">
                <div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4 mb-3">
                  <p class="text-xs text-slate-500 mb-2">Config file path:</p>
                  <code class="text-sm text-violet-300">~/Library/Application Support/Claude/claude_desktop_config.json</code>
                </div>
                <CodeBlock language="json" :code="claudeConfig" />
              </TabPanel>
              <TabPanel value="1">
                <div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4 mb-3">
                  <p class="text-xs text-slate-500 mb-2">Config file path:</p>
                  <code class="text-sm text-violet-300">%APPDATA%\Claude\claude_desktop_config.json</code>
                </div>
                <CodeBlock language="json" :code="claudeConfig" />
              </TabPanel>
            </TabPanels>
          </Tabs>

          <div class="mt-4 rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 flex items-start gap-3">
            <i class="pi pi-info-circle text-amber-400 text-sm mt-0.5 flex-shrink-0"></i>
            <p class="text-sm text-amber-300">
              Restart Claude Desktop after editing the configuration. The nexus-orchestrator tools will appear in Claude's tool palette.
              Make sure the nexus-daemon is running before starting Claude Desktop.
            </p>
          </div>
        </section>

        <!-- Available Tools -->
        <section id="available-tools" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> Available Tools
          </h2>
          <div class="overflow-x-auto rounded-xl border border-white/8">
            <table class="w-full text-sm">
              <thead>
                <tr class="bg-[#0d0d14]">
                  <th class="px-4 py-3 text-left text-slate-400">Tool</th>
                  <th class="px-4 py-3 text-left text-slate-400">Description</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(tool, i) in tools" :key="tool.name" :class="i % 2 === 0 ? 'bg-[#0a0a10]' : 'bg-[#0d0d14]'">
                  <td class="px-4 py-2.5 font-mono text-violet-300 text-xs">{{ tool.name }}</td>
                  <td class="px-4 py-2.5 text-slate-400 text-xs">{{ tool.desc }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- Usage Examples -->
        <section id="examples" class="reveal">
          <h2 class="text-2xl font-black mb-6 flex items-center gap-2">
            <span class="text-violet-400">#</span> Usage Examples
          </h2>

          <div class="space-y-8">
            <div v-for="example in examples" :key="example.title">
              <h3 class="text-lg font-bold mb-3 text-slate-200">{{ example.title }}</h3>
              <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
                <div>
                  <p class="text-xs text-slate-500 mb-2">Request</p>
                  <CodeBlock language="json" :code="example.request" />
                </div>
                <div v-if="example.response">
                  <p class="text-xs text-slate-500 mb-2">Response</p>
                  <CodeBlock language="json" :code="example.response" />
                </div>
              </div>
            </div>
          </div>
        </section>

        <!-- Protocol Details -->
        <section id="protocol" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> Protocol Details
          </h2>
          <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <div v-for="detail in protocolDetails" :key="detail.label" class="rounded-xl border border-white/8 bg-[#0d0d14] p-4 text-center">
              <div class="text-xs text-slate-500 mb-1">{{ detail.label }}</div>
              <div class="font-mono text-xs text-violet-300 font-bold">{{ detail.value }}</div>
            </div>
          </div>
          <p class="text-slate-500 text-sm mt-4">
            The MCP server supports both <code class="text-slate-300">initialize</code> and <code class="text-slate-300">tools/list</code> lifecycle methods,
            and all tool invocations via <code class="text-slate-300">tools/call</code>.
          </p>
        </section>

        <!-- Troubleshooting -->
        <section id="troubleshooting" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> Troubleshooting
          </h2>

          <!-- Warning box -->
          <div class="rounded-xl border border-red-500/20 bg-red-500/5 p-4 flex items-start gap-3 mb-4">
            <i class="pi pi-exclamation-triangle text-red-400 text-sm mt-0.5 flex-shrink-0"></i>
            <p class="text-sm text-red-300">
              <strong>Connection refused:</strong> Make sure the nexus-daemon is running and the MCP port (default 63988) is not blocked by a firewall.
            </p>
          </div>
          <div class="rounded-xl border border-blue-500/20 bg-blue-500/5 p-4 flex items-start gap-3 mb-6">
            <i class="pi pi-info-circle text-blue-400 text-sm mt-0.5 flex-shrink-0"></i>
            <p class="text-sm text-blue-300">
              <strong>No tools appearing:</strong> Verify the URL in <code>claude_desktop_config.json</code> ends with <code>/mcp</code> (not just the host:port).
            </p>
          </div>

          <div class="overflow-x-auto rounded-xl border border-white/8">
            <table class="w-full text-sm">
              <thead>
                <tr class="bg-[#0d0d14]">
                  <th class="px-4 py-3 text-left text-slate-400">Issue</th>
                  <th class="px-4 py-3 text-left text-slate-400">Solution</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(row, i) in troubleshootingRows" :key="row.issue" :class="i % 2 === 0 ? 'bg-[#0a0a10]' : 'bg-[#0d0d14]'">
                  <td class="px-4 py-2.5 text-red-300 text-xs">{{ row.issue }}</td>
                  <td class="px-4 py-2.5 text-slate-400 text-xs" v-html="row.solution"></td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- GitHub Actions -->
        <section id="github-actions" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> GitHub Actions
          </h2>
          <p class="text-slate-400 text-sm mb-6">
            Use nexus-orchestrator directly inside any GitHub Actions workflow. The Node.js 20 action
            downloads a local daemon, resolves agent identities from
            <a href="https://github.com/el-j/agency-agents" target="_blank" rel="noopener" class="text-violet-400 hover:text-violet-300">el-j/agency-agents</a>,
            submits your task to the LLM, polls until completion, and returns the result as step
            outputs — all in a single step.
          </p>

          <!-- Basic usage -->
          <h3 class="text-lg font-bold mb-3 text-slate-200">Basic Usage</h3>
          <CodeBlock language="yaml" :code="ghActionBasic" class="mb-8" />

          <!-- Built-in agent identity -->
          <h3 class="text-lg font-bold mb-3 text-slate-200">Built-in Agent Identities</h3>
          <p class="text-slate-400 text-sm mb-4">
            Use the <code class="text-violet-300">agent</code> input to select a specialist from
            <a href="https://github.com/el-j/agency-agents" target="_blank" rel="noopener" class="text-violet-400 hover:text-violet-300">el-j/agency-agents</a>
            — no separate step required. The system prompt is fetched and prepended automatically.
            Use <code class="text-violet-300">agents</code> for a named swarm or
            <code class="text-violet-300">agent_category</code> to load every agent in a category.
          </p>
          <CodeBlock language="yaml" :code="ghActionAgents" class="mb-4" />
          <CodeBlock language="yaml" :code="ghActionSwarm" class="mb-8" />

          <!-- Action inputs reference -->
          <h3 class="text-lg font-bold mb-3 text-slate-200">Key Inputs</h3>
          <div class="overflow-x-auto rounded-xl border border-white/8">
            <table class="w-full text-sm">
              <thead>
                <tr class="bg-[#0d0d14]">
                  <th class="px-4 py-3 text-left text-slate-400">Input</th>
                  <th class="px-4 py-3 text-left text-slate-400">Default</th>
                  <th class="px-4 py-3 text-left text-slate-400">Description</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(inp, i) in actionInputs" :key="inp.name" :class="i % 2 === 0 ? 'bg-[#0a0a10]' : 'bg-[#0d0d14]'">
                  <td class="px-4 py-2.5 font-mono text-violet-300 text-xs">{{ inp.name }}</td>
                  <td class="px-4 py-2.5 font-mono text-slate-500 text-xs">{{ inp.default }}</td>
                  <td class="px-4 py-2.5 text-slate-400 text-xs">{{ inp.desc }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import Tabs from 'primevue/tabs'
import TabList from 'primevue/tablist'
import Tab from 'primevue/tab'
import TabPanels from 'primevue/tabpanels'
import TabPanel from 'primevue/tabpanel'
import CodeBlock from '../components/CodeBlock.vue'
import { scrollToId } from '../utils/scroll'

const toc = [
  { id: 'what-is-mcp', label: 'What is MCP?' },
  { id: 'claude-desktop', label: 'Claude Desktop Setup' },
  { id: 'available-tools', label: 'Available Tools' },
  { id: 'examples', label: 'Usage Examples' },
  { id: 'protocol', label: 'Protocol Details' },
  { id: 'troubleshooting', label: 'Troubleshooting' },
  { id: 'github-actions', label: 'GitHub Actions' },
]

const claudeConfig = `{
  "mcpServers": {
    "nexus-orchestrator": {
      "url": "http://localhost:63988/mcp"
    }
  }
}`

const tools = [
  { name: 'submit_task', desc: 'Submit a code-generation task with project path, target file, and instruction' },
  { name: 'get_task', desc: 'Retrieve the status and output of a task by its ID' },
  { name: 'get_queue', desc: 'List all pending (QUEUED/PROCESSING) tasks' },
  { name: 'cancel_task', desc: 'Cancel a queued task before it is processed' },
  { name: 'get_providers', desc: 'List all registered LLM providers and their liveness status' },
  { name: 'health', desc: 'Check if the orchestrator daemon is running and responsive' },
]

const examples = [
  {
    title: 'Submit a Task',
    request: `{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "submit_task",
    "arguments": {
      "projectPath": "/path/to/project",
      "targetFile": "handler.go",
      "instruction": "Add error handling to the HTTP handler",
      "command": "execute"
    }
  }
}`,
    response: `{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\\"id\\":\\"abc-123\\",\\"status\\":\\"QUEUED\\"}"
      }
    ]
  }
}`,
  },
  {
    title: 'Get Task Status',
    request: `{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "get_task",
    "arguments": {
      "taskId": "abc-123"
    }
  }
}`,
    response: '',
  },
  {
    title: 'Check Available Providers',
    request: `{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "get_providers"
  }
}`,
    response: `{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "[{\\"name\\":\\"LM Studio\\",\\"active\\":true,\\"activeModel\\":\\"codellama\\"},{\\"name\\":\\"Ollama\\",\\"active\\":false}]"
      }
    ]
  }
}`,
  },
  {
    title: 'Health Check',
    request: `{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "health"
  }
}`,
    response: '',
  },
]

const protocolDetails = [
  { label: 'Protocol', value: 'JSON-RPC 2.0' },
  { label: 'Version', value: '2024-11-05' },
  { label: 'Endpoint', value: 'POST /mcp' },
  { label: 'Default Port', value: '63988' },
]

const troubleshootingRows = [
  { issue: 'Connection refused', solution: 'Start nexus-daemon first: <code class="text-slate-300">./nexus-daemon</code>' },
  { issue: 'Port conflict', solution: 'Use <code class="text-slate-300">NEXUS_MCP_ADDR=:9090</code> to change the MCP port' },
  { issue: 'No tools in Claude', solution: 'Check URL ends with <code class="text-slate-300">/mcp</code>, restart Claude Desktop' },
  { issue: 'Task stuck in QUEUED', solution: 'Check <code class="text-slate-300">GET /api/providers</code> — ensure at least one LLM provider is active' },
]

const ghActionBasic = `# .github/workflows/ai-task.yml
name: AI Code Generation

on:
  workflow_dispatch:
    inputs:
      instruction:
        description: 'What should the LLM implement?'
        required: true

jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Submit AI task
        id: nexus
        uses: el-j/nexus-orchestrator@v1
        with:
          instruction: \${{ github.event.inputs.instruction }}
          target_file: 'output.go'
          command: execute
          openai_api_key: \${{ secrets.OPENAI_API_KEY }}

      - name: Show result
        run: echo "\${{ steps.nexus.outputs.logs }}"`

const ghActionAgents = `# Single agent — identity loaded automatically from el-j/agency-agents
jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Submit AI task as Backend Architect
        id: nexus
        uses: el-j/nexus-orchestrator@v1
        with:
          agent: engineering-backend-architect   # fetched from el-j/agency-agents@main
          instruction: 'Add JWT authentication middleware to the HTTP handler'
          target_file: 'internal/middleware/auth.go'
          command: execute
          anthropic_api_key: \${{ secrets.ANTHROPIC_API_KEY }}

      - name: Commit generated code
        if: steps.nexus.outputs.status == 'COMPLETED'
        run: |
          git config user.name "nexus-bot"
          git config user.email "nexus-bot@users.noreply.github.com"
          git add -A
          git commit -m "feat: AI-generated auth middleware" || echo "nothing to commit"
          git push`

const ghActionSwarm = `# Swarm — load all engineering agents and orchestrate as a team
jobs:
  swarm:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Submit task to engineering swarm
        id: nexus
        uses: el-j/nexus-orchestrator@v1
        with:
          agent_category: engineering        # loads all agents in the category
          instruction: 'Design and implement a rate-limiter for the API'
          command: plan
          openai_api_key: \${{ secrets.OPENAI_API_KEY }}
          openai_model: gpt-4o`

const actionInputs = [
  { name: 'instruction', default: '—', desc: 'Task text sent to the LLM (required unless task_file is set)' },
  { name: 'task_file', default: '—', desc: 'Path to a .md/.txt file whose content becomes the instruction' },
  { name: 'agent', default: '""', desc: 'Single agent slug from el-j/agency-agents (e.g. "engineering-backend-architect")' },
  { name: 'agents', default: '""', desc: 'Comma-separated agent slugs — generates a combined swarm prompt' },
  { name: 'agent_category', default: '""', desc: 'Load ALL agents in a category (e.g. "engineering", "design", "testing")' },
  { name: 'agent_ref', default: 'main', desc: 'Git ref of el-j/agency-agents to fetch from (branch, tag, SHA)' },
  { name: 'system_prompt', default: '""', desc: 'Raw system prompt override — takes precedence over agent/agents/agent_category' },
  { name: 'target_file', default: '""', desc: 'Relative path the LLM should write (e.g. src/utils.go)' },
  { name: 'command', default: 'execute', desc: '"plan" (orchestration) or "execute" (code generation)' },
  { name: 'openai_api_key', default: '""', desc: 'OpenAI API key — enables GPT models on the local daemon' },
  { name: 'anthropic_api_key', default: '""', desc: 'Anthropic API key — enables Claude models on the local daemon' },
  { name: 'github_copilot_token', default: '""', desc: 'GitHub Copilot token — enables GPT-4o via Copilot' },
  { name: 'timeout_seconds', default: '300', desc: 'Seconds to wait for task completion before failing' },
  { name: 'nexus_version', default: 'latest', desc: 'Release version to install (e.g. v0.2.0)' },
  { name: 'start_daemon', default: 'true', desc: 'Download + start a local daemon; set false to use daemon_url' },
  { name: 'daemon_url', default: 'http://127.0.0.1:63987', desc: 'URL of existing daemon (only when start_daemon=false)' },
]
</script>
