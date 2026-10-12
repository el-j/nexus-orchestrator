<!--
  MCP Integration page: connection settings, the tool catalogue, usage examples and
  GitHub Action samples (from data/mcpIntegration.ts and data/mcpTools.ts).
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
            <UiIcon name="link" class="text-xs" />
            JSON-RPC 2.0 · MCP 2024-11-05
          </div>
          <h1 class="text-4xl font-black mb-4">
            <span class="gradient-text">MCP Integration</span>
          </h1>
          <p class="text-lg text-slate-400">
            Connect nexus-orchestrator to Claude Desktop and any MCP-compatible client.
          </p>
        </div>

        <!-- What is MCP -->
        <section id="what-is-mcp" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> What is MCP?
          </h2>
          <div class="rounded-xl border border-white/8 bg-[#0d0d14] p-6">
            <p class="text-slate-400 text-sm leading-relaxed">
              The
              <a
                href="https://modelcontextprotocol.io/"
                target="_blank"
                rel="noopener"
                class="text-violet-400 hover:text-violet-300"
                >Model Context Protocol</a
              >
              (MCP) is an open standard for connecting AI assistants to external tools and data
              sources. nexus-orchestrator implements an MCP server using JSON-RPC 2.0, making it
              compatible with Claude Desktop and any MCP-aware client.
            </p>
          </div>
        </section>

        <!-- Claude Desktop Setup -->
        <section id="claude-desktop" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> Claude Desktop Setup
          </h2>
          <p class="text-slate-400 text-sm mb-6">
            Add the following to your Claude Desktop configuration file:
          </p>

          <!-- OS Tabs -->
          <UiTabs :tabs="['🍎 macOS', '🪟 Windows']" label="Operating system">
            <template #panel-0>
              <div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4 mb-3">
                <p class="text-xs text-slate-500 mb-2">Config file path:</p>
                <code class="text-sm text-violet-300"
                  >~/Library/Application Support/Claude/claude_desktop_config.json</code
                >
              </div>
              <CodeBlock language="json" :code="claudeConfig" />
            </template>
            <template #panel-1>
              <div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4 mb-3">
                <p class="text-xs text-slate-500 mb-2">Config file path:</p>
                <code class="text-sm text-violet-300"
                  >%APPDATA%\Claude\claude_desktop_config.json</code
                >
              </div>
              <CodeBlock language="json" :code="claudeConfig" />
            </template>
          </UiTabs>

          <div
            class="mt-4 rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 flex items-start gap-3"
          >
            <UiIcon name="info-circle" class="text-amber-400 text-sm mt-0.5 flex-shrink-0" />
            <p class="text-sm text-amber-300">
              Restart Claude Desktop after editing the configuration. The nexus-orchestrator tools
              will appear in Claude's tool palette. Make sure the nexus-daemon is running before
              starting Claude Desktop.
            </p>
          </div>
        </section>

        <!-- VS Code Setup -->
        <section id="vscode-setup" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> VS Code Setup
          </h2>
          <p class="text-slate-400 text-sm mb-4">
            Add the following to your VS Code MCP configuration file:
          </p>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 mb-4">
            <div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4">
              <p class="text-xs text-slate-500 mb-1">macOS / Linux</p>
              <code class="text-xs text-violet-300"
                >~/Library/Application Support/Code/User/mcp.json</code
              >
            </div>
            <div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4">
              <p class="text-xs text-slate-500 mb-1">Windows</p>
              <code class="text-xs text-violet-300">%APPDATA%\Code\User\mcp.json</code>
            </div>
          </div>
          <CodeBlock language="json" :code="vsCodeConfig" />
          <div
            class="mt-4 rounded-xl border border-blue-500/20 bg-blue-500/5 p-4 flex items-start gap-3"
          >
            <UiIcon name="info-circle" class="text-blue-400 text-sm mt-0.5 flex-shrink-0" />
            <p class="text-sm text-blue-300">
              Use <code class="text-slate-200">"type": "http"</code> (Streamable HTTP),
              <strong>not</strong> <code class="text-slate-200">"type": "sse"</code>. The SSE
              transport holds a session in memory; a daemon restart invalidates it, causing
              <code class="text-slate-200">400</code> errors and
              <em>"terminated / Failed to parse message"</em> noise in the VS Code output panel.
              Streamable HTTP is stateless and reconnects cleanly every time. Reload the VS Code
              window after saving the configuration.
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
                <tr
                  v-for="(tool, i) in tools"
                  :key="tool.name"
                  :class="i % 2 === 0 ? 'bg-[#0a0a10]' : 'bg-[#0d0d14]'"
                >
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

        <!-- Brain Tools Examples -->
        <section id="brain-examples" class="reveal">
          <h2 class="text-2xl font-black mb-6 flex items-center gap-2">
            <span class="text-violet-400">#</span> Brain Tools
          </h2>
          <p class="text-slate-400 text-sm mb-6">
            The Brain / Knowledge tools let AI agents ingest project documentation and retrieve
            semantically-relevant context without reading files directly.
          </p>

          <div class="space-y-8">
            <div v-for="example in brainExamples" :key="example.title">
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
            <div
              v-for="detail in protocolDetails"
              :key="detail.label"
              class="rounded-xl border border-white/8 bg-[#0d0d14] p-4 text-center"
            >
              <div class="text-xs text-slate-500 mb-1">{{ detail.label }}</div>
              <div class="font-mono text-xs text-violet-300 font-bold">{{ detail.value }}</div>
            </div>
          </div>
          <p class="text-slate-500 text-sm mt-4">
            The MCP server supports both <code class="text-slate-300">initialize</code> and
            <code class="text-slate-300">tools/list</code> lifecycle methods, and all tool
            invocations via <code class="text-slate-300">tools/call</code>.
          </p>
        </section>

        <!-- Troubleshooting -->
        <section id="troubleshooting" class="reveal">
          <h2 class="text-2xl font-black mb-4 flex items-center gap-2">
            <span class="text-violet-400">#</span> Troubleshooting
          </h2>

          <!-- Warning box -->
          <div
            class="rounded-xl border border-red-500/20 bg-red-500/5 p-4 flex items-start gap-3 mb-4"
          >
            <UiIcon name="exclamation-triangle" class="text-red-400 text-sm mt-0.5 flex-shrink-0" />
            <p class="text-sm text-red-300">
              <strong>Connection refused:</strong> Make sure the nexus-daemon is running and the MCP
              port (default 63988) is not blocked by a firewall.
            </p>
          </div>
          <div
            class="rounded-xl border border-blue-500/20 bg-blue-500/5 p-4 flex items-start gap-3 mb-6"
          >
            <UiIcon name="info-circle" class="text-blue-400 text-sm mt-0.5 flex-shrink-0" />
            <p class="text-sm text-blue-300">
              <strong>No tools appearing:</strong> Verify the URL in
              <code>claude_desktop_config.json</code> ends with <code>/mcp</code> (not just the
              host:port).
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
                <tr
                  v-for="(row, i) in troubleshootingRows"
                  :key="row.issue"
                  :class="i % 2 === 0 ? 'bg-[#0a0a10]' : 'bg-[#0d0d14]'"
                >
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
            Use nexus-orchestrator directly inside any GitHub Actions workflow. The Node.js 20
            action downloads a local daemon, resolves agent identities from
            <a
              href="https://github.com/el-j/agency-agents"
              target="_blank"
              rel="noopener"
              class="text-violet-400 hover:text-violet-300"
              >el-j/agency-agents</a
            >, submits your task to the LLM, polls until completion, and returns the result as step
            outputs — all in a single step.
          </p>

          <!-- Basic usage -->
          <h3 class="text-lg font-bold mb-3 text-slate-200">Basic Usage</h3>
          <CodeBlock language="yaml" :code="ghActionBasic" class="mb-8" />

          <!-- Built-in agent identity -->
          <h3 class="text-lg font-bold mb-3 text-slate-200">Built-in Agent Identities</h3>
          <p class="text-slate-400 text-sm mb-4">
            Use the <code class="text-violet-300">agent</code> input to select a specialist from
            <a
              href="https://github.com/el-j/agency-agents"
              target="_blank"
              rel="noopener"
              class="text-violet-400 hover:text-violet-300"
              >el-j/agency-agents</a
            >
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
                <tr
                  v-for="(inp, i) in actionInputs"
                  :key="inp.name"
                  :class="i % 2 === 0 ? 'bg-[#0a0a10]' : 'bg-[#0d0d14]'"
                >
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
import UiIcon from '../ui/UiIcon.vue';
import UiTabs from '../ui/UiTabs.vue';
import CodeBlock from '../components/CodeBlock.vue';
import { scrollToId } from '../utils/scroll';

import {
  toc,
  claudeConfig,
  examples,
  vsCodeConfig,
  brainExamples,
  protocolDetails,
  troubleshootingRows,
  ghActionBasic,
  ghActionAgents,
  ghActionSwarm,
  actionInputs,
} from '../data/mcpIntegration';
import { tools } from '../data/mcpTools';
</script>
