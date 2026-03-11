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
            <i class="pi pi-server text-xs"></i>
            HTTP REST + MCP
          </div>
          <h1 class="text-4xl font-black mb-4"><span class="gradient-text">API Reference</span></h1>
          <p class="text-lg text-slate-400">Complete reference for all HTTP REST API endpoints and MCP tools.</p>
          <div class="mt-4 flex flex-wrap gap-3">
            <span class="px-3 py-1 rounded-lg bg-[#0d0d14] border border-white/8 text-xs text-slate-400 font-mono">Base URL: http://localhost:9999</span>
            <span class="px-3 py-1 rounded-lg bg-[#0d0d14] border border-white/8 text-xs text-slate-400 font-mono">MCP: http://localhost:9998</span>
          </div>
        </div>

        <!-- HTTP REST API -->
        <section id="http-api" class="reveal">
          <h2 class="text-2xl font-black mb-8 flex items-center gap-2">
            <span class="text-violet-400">#</span> HTTP REST API
          </h2>

          <!-- Each endpoint -->
          <div class="space-y-8">
            <div v-for="endpoint in endpoints" :key="endpoint.id" :id="endpoint.id" class="rounded-xl border border-white/8 bg-[#0d0d14] overflow-hidden">
              <!-- Endpoint header -->
              <div class="px-4 py-3 bg-[#14141f] border-b border-white/5 flex items-center gap-3 flex-wrap">
                <span :class="`px-2 py-0.5 rounded text-xs font-bold font-mono ${methodColor(endpoint.method)}`">
                  {{ endpoint.method }}
                </span>
                <code class="text-sm text-slate-200 font-mono">{{ endpoint.path }}</code>
                <span class="ml-auto text-xs text-slate-500">{{ endpoint.title }}</span>
              </div>
              <div class="p-5 space-y-4">
                <p class="text-sm text-slate-400">{{ endpoint.desc }}</p>

                <div v-if="endpoint.requestBody">
                  <p class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">Request Body</p>
                  <CodeBlock language="json" :code="endpoint.requestBody" />
                </div>

                <div v-if="endpoint.fields && endpoint.fields.length">
                  <p class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">Fields</p>
                  <div class="overflow-x-auto rounded-lg border border-white/5">
                    <table class="w-full text-xs">
                      <thead>
                        <tr class="bg-[#111118]">
                          <th class="px-3 py-2 text-left text-slate-400">Field</th>
                          <th class="px-3 py-2 text-left text-slate-400">Required</th>
                          <th class="px-3 py-2 text-left text-slate-400">Description</th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr v-for="(f, fi) in endpoint.fields" :key="f.field" :class="fi % 2 === 0 ? 'bg-[#0a0a10]' : 'bg-[#0d0d14]'">
                          <td class="px-3 py-2 font-mono text-violet-300">{{ f.field }}</td>
                          <td class="px-3 py-2">
                            <span v-if="f.required" class="text-emerald-400">Yes</span>
                            <span v-else class="text-slate-500">No</span>
                          </td>
                          <td class="px-3 py-2 text-slate-400">{{ f.desc }}</td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
                </div>

                <div v-if="endpoint.response">
                  <p class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">
                    Response <span class="text-emerald-400 normal-case font-normal">{{ endpoint.responseStatus }}</span>
                  </p>
                  <CodeBlock language="json" :code="endpoint.response" />
                </div>
              </div>
            </div>
          </div>
        </section>

        <!-- SSE Events -->
        <section id="sse-events" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> SSE Event Stream
          </h2>
          <div class="rounded-xl border border-white/8 bg-[#0d0d14] overflow-hidden mb-4">
            <div class="px-4 py-3 bg-[#14141f] border-b border-white/5 flex items-center gap-3">
              <span class="px-2 py-0.5 rounded text-xs font-bold font-mono bg-blue-500/20 text-blue-300">GET</span>
              <code class="text-sm text-slate-200 font-mono">/api/events</code>
            </div>
            <div class="p-5">
              <p class="text-sm text-slate-400 mb-4">Server-Sent Events stream for real-time task lifecycle updates.</p>
              <div class="overflow-x-auto rounded-lg border border-white/5 mb-4">
                <table class="w-full text-xs">
                  <thead>
                    <tr class="bg-[#111118]">
                      <th class="px-3 py-2 text-left text-slate-400">Event</th>
                      <th class="px-3 py-2 text-left text-slate-400">Description</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="(ev, i) in sseEvents" :key="ev.event" :class="i % 2 === 0 ? 'bg-[#0a0a10]' : 'bg-[#0d0d14]'">
                      <td class="px-3 py-2 font-mono text-cyan-400">{{ ev.event }}</td>
                      <td class="px-3 py-2 text-slate-400">{{ ev.desc }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <p class="text-xs text-slate-500 mb-2">Event Format:</p>
              <CodeBlock language="text" :code="sseFormat" />
            </div>
          </div>
        </section>

        <!-- MCP Server -->
        <section id="mcp-server" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> MCP Server
          </h2>
          <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
            <div v-for="info in mcpInfo" :key="info.label" class="rounded-lg border border-white/8 bg-[#0d0d14] p-3 text-center">
              <div class="text-xs text-slate-500 mb-1">{{ info.label }}</div>
              <div class="font-mono text-xs text-violet-300 font-bold">{{ info.value }}</div>
            </div>
          </div>

          <h3 class="text-lg font-bold mb-3 text-slate-200">Available Tools</h3>
          <div class="overflow-x-auto rounded-xl border border-white/8 mb-6">
            <table class="w-full text-sm">
              <thead>
                <tr class="bg-[#0d0d14]">
                  <th class="px-4 py-3 text-left text-slate-400">Tool</th>
                  <th class="px-4 py-3 text-left text-slate-400">Description</th>
                  <th class="px-4 py-3 text-left text-slate-400">Parameters</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(tool, i) in mcpTools" :key="tool.name" :class="i % 2 === 0 ? 'bg-[#0a0a10]' : 'bg-[#0d0d14]'">
                  <td class="px-4 py-2.5 font-mono text-violet-300 text-xs">{{ tool.name }}</td>
                  <td class="px-4 py-2.5 text-slate-400 text-xs">{{ tool.desc }}</td>
                  <td class="px-4 py-2.5 font-mono text-cyan-400 text-xs">{{ tool.params }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <h3 class="text-lg font-bold mb-3 text-slate-200">Example: Submit Task via MCP</h3>
          <CodeBlock language="json" :code="mcpSubmit" />
        </section>

        <!-- Environment Variables -->
        <section id="env-vars" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> Environment Variables
          </h2>
          <div class="overflow-x-auto rounded-xl border border-white/8">
            <table class="w-full text-sm">
              <thead>
                <tr class="bg-[#0d0d14]">
                  <th class="px-4 py-3 text-left text-slate-400">Variable</th>
                  <th class="px-4 py-3 text-left text-slate-400">Default</th>
                  <th class="px-4 py-3 text-left text-slate-400">Description</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(row, i) in envVars" :key="row.var" :class="i % 2 === 0 ? 'bg-[#0a0a10]' : 'bg-[#0d0d14]'">
                  <td class="px-4 py-2.5 font-mono text-violet-300 text-xs">{{ row.var }}</td>
                  <td class="px-4 py-2.5 font-mono text-slate-500 text-xs">{{ row.default || '—' }}</td>
                  <td class="px-4 py-2.5 text-slate-400 text-xs">{{ row.desc }}</td>
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
import CodeBlock from '../components/CodeBlock.vue'
import { scrollToId } from '../utils/scroll'

const toc = [
  { id: 'http-api', label: 'HTTP REST API' },
  { id: 'sse-events', label: 'SSE Event Stream' },
  { id: 'mcp-server', label: 'MCP Server' },
  { id: 'env-vars', label: 'Environment Variables' },
]

function methodColor(method: string) {
  if (method === 'POST') return 'bg-emerald-500/20 text-emerald-300'
  if (method === 'GET') return 'bg-blue-500/20 text-blue-300'
  if (method === 'DELETE') return 'bg-red-500/20 text-red-300'
  return 'bg-slate-500/20 text-slate-300'
}

const endpoints = [
  {
    id: 'submit-task',
    method: 'POST',
    path: '/api/tasks',
    title: 'Submit Task',
    desc: 'Submit a new code-generation task to the queue.',
    requestBody: `{
  "projectPath": "/path/to/project",
  "targetFile": "output.go",
  "instruction": "Write a function that sorts strings",
  "contextFiles": ["main.go", "utils.go"],
  "modelId": "codellama",
  "providerHint": "LM Studio",
  "command": "execute"
}`,
    fields: [
      { field: 'projectPath', required: true, desc: 'Absolute path to the project directory' },
      { field: 'targetFile', required: true, desc: 'Relative path for the generated output file' },
      { field: 'instruction', required: true, desc: 'Natural language prompt for the LLM' },
      { field: 'contextFiles', required: false, desc: 'List of files to include as context' },
      { field: 'modelId', required: false, desc: 'Constrain to a specific model' },
      { field: 'providerHint', required: false, desc: 'Prefer a specific provider by name' },
      { field: 'command', required: false, desc: 'Task type: plan, execute, or auto (default: auto)' },
    ],
    responseStatus: '201 Created',
    response: `{
  "id": "a1b2c3d4-e5f6-...",
  "projectPath": "/path/to/project",
  "targetFile": "output.go",
  "instruction": "Write a function that sorts strings",
  "status": "QUEUED",
  "command": "execute",
  "createdAt": "2025-01-01T00:00:00Z",
  "updatedAt": "2025-01-01T00:00:00Z"
}`,
  },
  {
    id: 'list-tasks',
    method: 'GET',
    path: '/api/tasks',
    title: 'List Tasks',
    desc: 'Returns all pending (QUEUED or PROCESSING) tasks.',
    responseStatus: '200 OK',
    response: `[
  {
    "id": "...",
    "status": "QUEUED",
    "instruction": "..."
  }
]`,
  },
  {
    id: 'get-task',
    method: 'GET',
    path: '/api/tasks/{id}',
    title: 'Get Task',
    desc: 'Retrieve a single task by ID.',
    responseStatus: '200 OK',
    response: `{
  "id": "a1b2c3d4-...",
  "status": "COMPLETED",
  "logs": "generated code output..."
}`,
  },
  {
    id: 'cancel-task',
    method: 'DELETE',
    path: '/api/tasks/{id}',
    title: 'Cancel Task',
    desc: 'Cancel a queued task before it is processed. Returns 204 No Content on success, 404 Not Found if task doesn\'t exist or already processed.',
    responseStatus: '204 No Content',
    response: '',
  },
  {
    id: 'list-providers',
    method: 'GET',
    path: '/api/providers',
    title: 'List Providers',
    desc: 'Returns all registered LLM providers with their liveness status.',
    responseStatus: '200 OK',
    response: `[
  {
    "name": "LM Studio",
    "active": true,
    "activeModel": "codellama",
    "models": ["codellama", "deepseek-coder"]
  },
  {
    "name": "Ollama",
    "active": false
  }
]`,
  },
  {
    id: 'register-provider',
    method: 'POST',
    path: '/api/providers',
    title: 'Register Provider',
    desc: 'Dynamically register a new cloud LLM provider.',
    requestBody: `{
  "name": "My OpenAI",
  "kind": "openai-compat",
  "baseURL": "https://api.openai.com/v1",
  "apiKey": "sk-...",
  "model": "gpt-4o-mini"
}`,
    fields: [
      { field: 'name', required: true, desc: 'Display name for the provider' },
      { field: 'kind', required: true, desc: 'Provider type: lmstudio, ollama, openai-compat, anthropic' },
      { field: 'baseURL', required: true, desc: 'API endpoint URL' },
      { field: 'apiKey', required: false, desc: 'Required for cloud providers' },
      { field: 'model', required: false, desc: 'Default model to use' },
    ],
    responseStatus: '201 Created',
    response: '',
  },
  {
    id: 'remove-provider',
    method: 'DELETE',
    path: '/api/providers/{name}',
    title: 'Remove Provider',
    desc: 'Deregister a provider by name.',
    responseStatus: '204 No Content',
    response: '',
  },
  {
    id: 'health',
    method: 'GET',
    path: '/api/health',
    title: 'Health Check',
    desc: 'Returns daemon health status.',
    responseStatus: '200 OK',
    response: `{"status": "ok"}`,
  },
]

const sseEvents = [
  { event: 'task.queued', desc: 'Task was added to the queue' },
  { event: 'task.processing', desc: 'Task is being processed by an LLM' },
  { event: 'task.completed', desc: 'Task completed successfully' },
  { event: 'task.failed', desc: 'Task processing failed' },
  { event: 'task.cancelled', desc: 'Task was cancelled' },
  { event: 'task.too_large', desc: 'Task exceeded context window' },
  { event: 'task.no_provider', desc: 'No provider available for the task' },
]

const sseFormat = `event: task.completed
data: {"type":"task.completed","taskId":"abc-123","status":"COMPLETED"}`

const mcpInfo = [
  { label: 'Standard', value: 'JSON-RPC 2.0' },
  { label: 'Version', value: '2024-11-05' },
  { label: 'Endpoint', value: 'POST /mcp' },
  { label: 'Default Port', value: '9998' },
]

const mcpTools = [
  { name: 'submit_task', desc: 'Submit a code-generation task', params: 'projectPath, targetFile, instruction, contextFiles, command' },
  { name: 'get_task', desc: 'Get task by ID', params: 'taskId' },
  { name: 'get_queue', desc: 'List all pending tasks', params: '—' },
  { name: 'cancel_task', desc: 'Cancel a queued task', params: 'taskId' },
  { name: 'get_providers', desc: 'List LLM providers', params: '—' },
  { name: 'health', desc: 'Check daemon status', params: '—' },
]

const mcpSubmit = `// Request
{
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
}

// Response
{
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
}`

const envVars = [
  { var: 'NEXUS_DB_PATH', default: 'nexus.db', desc: 'SQLite database file path' },
  { var: 'NEXUS_LISTEN_ADDR', default: '127.0.0.1:9999', desc: 'HTTP API listen address' },
  { var: 'NEXUS_MCP_ADDR', default: '127.0.0.1:9998', desc: 'MCP server listen address' },
  { var: 'NEXUS_OPENAI_API_KEY', default: '', desc: 'OpenAI API key (enables OpenAI provider)' },
  { var: 'NEXUS_OPENAI_MODEL', default: 'gpt-4o-mini', desc: 'Default OpenAI model' },
  { var: 'NEXUS_ANTHROPIC_API_KEY', default: '', desc: 'Anthropic API key (enables Anthropic provider)' },
  { var: 'NEXUS_ANTHROPIC_MODEL', default: 'claude-3-5-sonnet-20241022', desc: 'Default Anthropic model' },
  { var: 'NEXUS_GITHUBCOPILOT_TOKEN', default: '', desc: 'GitHub Copilot token' },
  { var: 'NEXUS_GITHUBCOPILOT_MODEL', default: 'gpt-4o', desc: 'Default GitHub Copilot model' },
]
</script>
