import{C as e,D as t,E as n,S as r,T as i,_ as a,f as o,g as s,h as c,i as l,l as ee,m as u,o as d,r as f,u as p,v as m,x as h,y as te}from"./app-BCmDooNI.js";import{t as g}from"./scroll-DdeK3J_A.js";var _=[{id:`what-is-mcp`,label:`What is MCP?`},{id:`connect-command`,label:`Connect Your Agents`},{id:`claude-desktop`,label:`Claude Desktop Setup`},{id:`vscode-setup`,label:`VS Code Setup`},{id:`available-tools`,label:`Available Tools`},{id:`examples`,label:`Usage Examples`},{id:`brain-examples`,label:`Brain Tools`},{id:`protocol`,label:`Protocol Details`},{id:`troubleshooting`,label:`Troubleshooting`},{id:`github-actions`,label:`GitHub Actions`}],v=`# Which agents on this machine can call Nexus? (changes nothing)
nexus-cli connect

# Connect them: Antigravity, Cursor, Claude Code
nexus-cli connect --write
nexus-cli connect --write --agent antigravity`,y=`{
  "mcpServers": {
    "nexus-orchestrator": {
      "url": "http://localhost:63988/mcp"
    }
  }
}`,b=[{title:`Submit a Task`,request:`{
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
}`,response:`{
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
}`},{title:`Get Task Status`,request:`{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "get_task",
    "arguments": {
      "id": "abc-123"
    }
  }
}`,response:``},{title:`Check Available Providers`,request:`{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "get_providers"
  }
}`,response:`{
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
}`},{title:`Health Check`,request:`{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "health"
  }
}`,response:``}],x=`{
  "servers": {
    "Nexus Orchestrator": {
      "type": "http",
      "url": "http://127.0.0.1:63988/mcp"
    }
  }
}`,S=[{title:`Check Brain Status`,request:`{
  "jsonrpc": "2.0",
  "id": 10,
  "method": "tools/call",
  "params": {
    "name": "get_brain_status",
    "arguments": {
      "projectPath": "/your/project"
    }
  }
}`,response:`{
  "jsonrpc": "2.0",
  "id": 10,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\\"projectPath\\":\\"/your/project\\",\\"entryCount\\":42,\\"lastIngested\\":\\"2025-01-01T00:00:00Z\\"}"
      }
    ]
  }
}`},{title:`Ingest a Knowledge File`,request:`{
  "jsonrpc": "2.0",
  "id": 11,
  "method": "tools/call",
  "params": {
    "name": "ingest_knowledge",
    "arguments": {
      "projectPath": "/your/project",
      "filePath": "/your/project/CLAUDE.md"
    }
  }
}`,response:`{
  "jsonrpc": "2.0",
  "id": 11,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\\"ingested\\":12,\\"projectPath\\":\\"/your/project\\"}"
      }
    ]
  }
}`},{title:`Get Project Context`,request:`{
  "jsonrpc": "2.0",
  "id": 12,
  "method": "tools/call",
  "params": {
    "name": "get_project_context",
    "arguments": {
      "projectPath": "/your/project",
      "maxTokens": 800
    }
  }
}`,response:``},{title:`Search Knowledge Base`,request:`{
  "jsonrpc": "2.0",
  "id": 13,
  "method": "tools/call",
  "params": {
    "name": "search_knowledge",
    "arguments": {
      "projectPath": "/your/project",
      "query": "architecture",
      "limit": 5
    }
  }
}`,response:``}],C=[{label:`Protocol`,value:`JSON-RPC 2.0`},{label:`Version`,value:`2024-11-05`},{label:`Endpoint`,value:`POST /mcp`},{label:`Default Port`,value:`63988`}],w=[{issue:`Connection refused`,solution:`Start nexus-daemon first: <code class="text-slate-300">./nexus-daemon</code>`},{issue:`Port conflict`,solution:`Use <code class="text-slate-300">NEXUS_MCP_ADDR=:9090</code> to change the MCP port`},{issue:`No tools in Claude`,solution:`Check URL ends with <code class="text-slate-300">/mcp</code>, restart Claude Desktop`},{issue:`Task stuck in QUEUED`,solution:`Check <code class="text-slate-300">GET /api/providers</code> — ensure at least one LLM provider is active`}],T=`# .github/workflows/ai-task.yml
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
        run: echo "\${{ steps.nexus.outputs.logs }}"`,E=`# Single agent — identity loaded automatically from el-j/agency-agents
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
          git push`,D=`# Swarm — load all engineering agents and orchestrate as a team
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
          openai_model: gpt-4o`,O=[{name:`instruction`,default:`—`,desc:`Task text sent to the LLM (required unless task_file is set)`},{name:`task_file`,default:`—`,desc:`Path to a .md/.txt file whose content becomes the instruction`},{name:`agent`,default:`""`,desc:`Single agent slug from el-j/agency-agents (e.g. "engineering-backend-architect")`},{name:`agents`,default:`""`,desc:`Comma-separated agent slugs — generates a combined swarm prompt`},{name:`agent_category`,default:`""`,desc:`Load ALL agents in a category (e.g. "engineering", "design", "testing")`},{name:`agent_ref`,default:`main`,desc:`Git ref of el-j/agency-agents to fetch from (branch, tag, SHA)`},{name:`system_prompt`,default:`""`,desc:`Raw system prompt override — takes precedence over agent/agents/agent_category`},{name:`target_file`,default:`""`,desc:`Relative path the LLM should write (e.g. src/utils.go)`},{name:`command`,default:`auto`,desc:`"plan" (orchestration), "execute" (code generation, needs a prior completed plan) or "auto"`},{name:`openai_api_key`,default:`""`,desc:`OpenAI API key — enables GPT models on the local daemon`},{name:`anthropic_api_key`,default:`""`,desc:`Anthropic API key — enables Claude models on the local daemon`},{name:`github_copilot_token`,default:`""`,desc:`GitHub Copilot token — enables GPT-4o via Copilot`},{name:`timeout_seconds`,default:`300`,desc:`Seconds to wait for task completion before failing`},{name:`nexus_version`,default:`latest`,desc:`Release version to install (e.g. v0.2.0)`},{name:`start_daemon`,default:`true`,desc:`Download + start a local daemon; set false to use daemon_url`},{name:`daemon_url`,default:`http://127.0.0.1:63987`,desc:`URL of existing daemon (only when start_daemon=false)`}],k=[{category:`Tasks`,name:`submit_task`,desc:`Submit a new code-generation task to the orchestrator`},{category:`Tasks`,name:`get_task`,desc:`Get the current status and output of a task by ID`},{category:`Tasks`,name:`wait_for_task`,desc:`Wait until a task has finished and return it; streams the reply as progress notifications when asked`},{category:`Tasks`,name:`get_queue`,desc:`List all tasks currently in the queue`},{category:`Tasks`,name:`get_all_tasks`,desc:`Return every task regardless of status (QUEUED, PROCESSING, DRAFT, BACKLOG, COMPLETED, FAILED, CANCELLED)`},{category:`Tasks`,name:`cancel_task`,desc:`Cancel a pending task by ID`},{category:`Tasks`,name:`update_task`,desc:`Update mutable fields on an existing task (instruction, priority, provider, tags, status)`},{category:`Tasks`,name:`create_draft`,desc:`Create a draft idea for a project without entering the execution queue`},{category:`Tasks`,name:`get_backlog`,desc:`List draft and backlog items for a project, ordered by priority`},{category:`Tasks`,name:`promote_task`,desc:`Promote a draft or backlog task to the execution queue`},{category:`Tasks`,name:`claim_task`,desc:`Claim a QUEUED task for execution by the specified AI session, transitioning it to PROCESSING`},{category:`Tasks`,name:`update_task_status`,desc:`Report task completion or failure from the executing AI session`},{category:`Tasks`,name:`heartbeat_task`,desc:`Keep a PROCESSING task alive — prevents the watchdog from marking it failed`},{category:`Tasks`,name:`terminate_ai_session`,desc:`Terminate an external AI agent session (SIGTERM by default, SIGKILL when force=true)`},{category:`AI Sessions`,name:`register_session`,desc:`Announce this AI agent session to nexusOrchestrator for visualisation and orchestration`},{category:`AI Sessions`,name:`get_ai_sessions`,desc:`Return the list of all known external AI agent sessions registered with this instance`},{category:`AI Sessions`,name:`deregister_ai_session`,desc:`Soft-disconnect an AI agent session, marking it as disconnected without killing the process`},{category:`AI Sessions`,name:`heartbeat_ai_session`,desc:`Refresh the last-activity timestamp of an AI session to keep it alive`},{category:`AI Sessions`,name:`purge_disconnected_sessions`,desc:`Delete all AI sessions with status "disconnected" inactive for more than 2 hours`},{category:`Providers`,name:`get_providers`,desc:`List available LLM providers and their models`},{category:`Providers`,name:`discover_providers`,desc:`Scan the local system for installed AI providers/agents and return discovered results`},{category:`Providers`,name:`promote_provider`,desc:`Promote a discovered provider to an active LLM backend`},{category:`Providers`,name:`list_provider_configs`,desc:`List all persisted LLM provider configuration records`},{category:`Providers`,name:`add_provider_config`,desc:`Add a new LLM provider configuration and register it when enabled=true`},{category:`Providers`,name:`update_provider_config`,desc:`Update an existing LLM provider configuration by ID`},{category:`Providers`,name:`remove_provider_config`,desc:`Delete a persisted provider configuration and deregister its adapter`},{category:`Brain / Knowledge`,name:`get_brain_status`,desc:`Check the knowledge repository status for a project`},{category:`Brain / Knowledge`,name:`ingest_knowledge`,desc:`Parse and ingest a markdown file (often CLAUDE.md) into the project knowledge repository`},{category:`Brain / Knowledge`,name:`get_project_context`,desc:`Get macro context for a project (Architectures, Conventions, File Maps) bounded by a token budget`},{category:`Brain / Knowledge`,name:`get_onboarding_context`,desc:`Get a token-budgeted (< 800 tokens) onboarding summary for agent bootstrap: stack, invariants, test commands, active plan and tasks`},{category:`Brain / Knowledge`,name:`get_focused_context`,desc:`Get task-specific micro context (Learning, Definitions, Gotchas) bounded by a token budget`},{category:`Brain / Knowledge`,name:`search_knowledge`,desc:`Perform full-text search across the project's knowledge base`},{category:`Brain / Knowledge`,name:`init_project`,desc:`Auto-ingest CLAUDE.md and initialize a project's knowledge base in one step`},{category:`Brain / Knowledge`,name:`list_knowledge`,desc:`List all knowledge documents stored for a project, optionally filtered by kind`},{category:`Brain / Knowledge`,name:`delete_knowledge`,desc:`Delete a knowledge document by ID from the project repository`},{category:`Brain / Knowledge`,name:`get_file_map`,desc:`Retrieve the cached or structured file map knowledge document for a project`},{category:`Discovery & System`,name:`get_discovered_agents`,desc:`Return AI agent tools detected on the local system (Claude CLI, VS Code Copilot, etc.)`},{category:`Discovery & System`,name:`delegate_to_nexus`,desc:`Delegate an AI agent session to the nexus task queue and return the workflow instruction string`},{category:`Discovery & System`,name:`get_discovered_plans`,desc:`Scan for plan/task/orchestration files in a project directory`},{category:`Discovery & System`,name:`howto`,desc:`Return a complete integration guide — all tools, workflow patterns, and HTTP endpoint reference`},{category:`Discovery & System`,name:`howto_brief`,desc:`Get the ultra-compact integration guide (~200 tokens), recommended for small-context models`},{category:`Discovery & System`,name:`health`,desc:`Check that the nexusOrchestrator daemon is reachable`},{category:`Discovery & System`,name:`get_routing`,desc:`How providers would be ordered for a role in a project: model ladder, cooldowns and quota readings`},{category:`Discovery & System`,name:`get_usage`,desc:`Tokens and cost per provider, model and role from the usage ledger (since: 24h, 7d, an RFC 3339 time or all)`}],A={class:`max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16`},j={class:`grid grid-cols-1 lg:grid-cols-4 gap-12`},M={class:`lg:col-span-1`},N={class:`sticky top-24 space-y-1`},P={class:`space-y-1`},F=[`href`,`onClick`],I={class:`lg:col-span-3 space-y-16`},L={class:`reveal`},R={class:`inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-violet-500/30 bg-violet-500/5 text-sm text-violet-300 mb-4`},z={id:`connect-command`,class:`reveal`},B={id:`claude-desktop`,class:`reveal`},V={class:`mt-4 rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 flex items-start gap-3`},H={id:`vscode-setup`,class:`reveal`},U={class:`mt-4 rounded-xl border border-blue-500/20 bg-blue-500/5 p-4 flex items-start gap-3`},ne={id:`available-tools`,class:`reveal`},W={class:`overflow-x-auto rounded-xl border border-white/8`},G={class:`w-full text-sm`},K={class:`px-4 py-2.5 font-mono text-violet-300 text-xs`},q={class:`px-4 py-2.5 text-slate-400 text-xs`},J={id:`examples`,class:`reveal`},Y={class:`space-y-8`},X={class:`text-lg font-bold mb-3 text-slate-200`},Z={class:`grid grid-cols-1 lg:grid-cols-2 gap-4`},re={key:0},ie={id:`brain-examples`,class:`reveal`},ae={class:`space-y-8`},oe={class:`text-lg font-bold mb-3 text-slate-200`},se={class:`grid grid-cols-1 lg:grid-cols-2 gap-4`},ce={key:0},le={id:`protocol`,class:`reveal`},ue={class:`grid grid-cols-2 sm:grid-cols-4 gap-3`},de={class:`text-xs text-slate-500 mb-1`},fe={class:`font-mono text-xs text-violet-300 font-bold`},pe={id:`troubleshooting`,class:`reveal`},me={class:`rounded-xl border border-red-500/20 bg-red-500/5 p-4 flex items-start gap-3 mb-4`},Q={class:`rounded-xl border border-blue-500/20 bg-blue-500/5 p-4 flex items-start gap-3 mb-6`},he={class:`overflow-x-auto rounded-xl border border-white/8`},ge={class:`w-full text-sm`},_e={class:`px-4 py-2.5 text-red-300 text-xs`},ve=[`innerHTML`],ye={id:`github-actions`,class:`reveal`},be={class:`overflow-x-auto rounded-xl border border-white/8`},xe={class:`w-full text-sm`},Se={class:`px-4 py-2.5 font-mono text-violet-300 text-xs`},Ce={class:`px-4 py-2.5 font-mono text-slate-500 text-xs`},we={class:`px-4 py-2.5 text-slate-400 text-xs`},$=te({__name:`McpIntegrationView`,setup(te){return(te,$)=>(h(),c(`div`,A,[o(`div`,j,[o(`aside`,M,[o(`div`,N,[$[0]||=o(`h4`,{class:`text-xs font-semibold uppercase tracking-wider text-slate-500 mb-4`},` On this page `,-1),o(`nav`,P,[(h(!0),c(p,null,r(i(_),e=>(h(),c(`a`,{key:e.id,href:`#${e.id}`,onClick:ee(t=>i(g)(e.id),[`prevent`]),class:`block text-sm text-slate-500 hover:text-violet-400 transition-colors py-0.5 cursor-pointer`},t(e.label),9,F))),128))])])]),o(`main`,I,[o(`div`,L,[o(`div`,R,[m(d,{name:`link`,class:`text-xs`}),$[1]||=a(` JSON-RPC 2.0 · MCP 2024-11-05 `,-1)]),$[2]||=o(`h1`,{class:`text-4xl font-black mb-4`},[o(`span`,{class:`gradient-text`},`MCP Integration`)],-1),$[3]||=o(`p`,{class:`text-lg text-slate-400`},` Connect Claude Code, Claude Desktop, Cursor, Antigravity and any MCP-compatible client to nexus-orchestrator. `,-1)]),$[33]||=s(`<section id="what-is-mcp" class="reveal"><h2 class="text-2xl font-black mb-4 flex items-center gap-2"><span class="text-violet-400">#</span> What is MCP? </h2><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-6"><p class="text-slate-400 text-sm leading-relaxed"> The <a href="https://modelcontextprotocol.io/" target="_blank" rel="noopener" class="text-violet-400 hover:text-violet-300">Model Context Protocol</a> (MCP) is an open standard for connecting AI assistants to external tools and data sources. nexus-orchestrator implements an MCP server using JSON-RPC 2.0, making it compatible with Claude Desktop and any MCP-aware client. </p></div></section>`,1),o(`section`,z,[$[4]||=o(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[o(`span`,{class:`text-violet-400`},`#`),a(` Connect Your Agents `)],-1),$[5]||=o(`p`,{class:`text-slate-400 text-sm mb-4`},[a(` An agent that cannot call Nexus tools can only work around them. This finds the agents installed on your machine, shows which are not connected, and with `),o(`code`,{class:`text-violet-300`},`--write`),a(` adds the nexus server to their config (other servers stay, the old file is kept as a backup): `)],-1),m(f,{language:`bash`,code:i(v)},null,8,[`code`]),$[6]||=o(`p`,{class:`text-slate-500 text-sm mt-3`},[a(` Antigravity is connected through the `),o(`code`,null,`nexus-mcp-stdio`),a(` bridge, Cursor through the HTTP endpoint, Claude Code through `),o(`code`,null,`claude mcp add`),a(`. The VS Code extension does this for the editor's own agent: VS Code 1.101+ through its MCP support, Antigravity after one prompt. Tools that stream: `),o(`code`,null,`wait_for_task`),a(` sends the model's text as progress notifications when the client asks for them. `)],-1)]),o(`section`,B,[$[10]||=o(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[o(`span`,{class:`text-violet-400`},`#`),a(` Claude Desktop Setup `)],-1),$[11]||=o(`p`,{class:`text-slate-400 text-sm mb-6`},` Add the following to your Claude Desktop configuration file: `,-1),m(l,{tabs:[`🍎 macOS`,`🪟 Windows`],label:`Operating system`},{"panel-0":e(()=>[$[7]||=o(`div`,{class:`rounded-xl border border-white/8 bg-[#0d0d14] p-4 mb-3`},[o(`p`,{class:`text-xs text-slate-500 mb-2`},`Config file path:`),o(`code`,{class:`text-sm text-violet-300`},`~/Library/Application Support/Claude/claude_desktop_config.json`)],-1),m(f,{language:`json`,code:i(y)},null,8,[`code`])]),"panel-1":e(()=>[$[8]||=o(`div`,{class:`rounded-xl border border-white/8 bg-[#0d0d14] p-4 mb-3`},[o(`p`,{class:`text-xs text-slate-500 mb-2`},`Config file path:`),o(`code`,{class:`text-sm text-violet-300`},`%APPDATA%\\Claude\\claude_desktop_config.json`)],-1),m(f,{language:`json`,code:i(y)},null,8,[`code`])]),_:1}),o(`div`,V,[m(d,{name:`info-circle`,class:`text-amber-400 text-sm mt-0.5 flex-shrink-0`}),$[9]||=o(`p`,{class:`text-sm text-amber-300`},` Restart Claude Desktop after editing the configuration. The nexus-orchestrator tools will appear in Claude's tool palette. Make sure the nexus-daemon is running before starting Claude Desktop. `,-1)])]),o(`section`,H,[$[13]||=s(`<h2 class="text-2xl font-black mb-4 flex items-center gap-2"><span class="text-violet-400">#</span> VS Code Setup </h2><p class="text-slate-400 text-sm mb-4"> Add the following to your VS Code MCP configuration file: </p><div class="grid grid-cols-1 sm:grid-cols-2 gap-3 mb-4"><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4"><p class="text-xs text-slate-500 mb-1">macOS / Linux</p><code class="text-xs text-violet-300">~/Library/Application Support/Code/User/mcp.json</code></div><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4"><p class="text-xs text-slate-500 mb-1">Windows</p><code class="text-xs text-violet-300">%APPDATA%\\Code\\User\\mcp.json</code></div></div>`,3),m(f,{language:`json`,code:i(x)},null,8,[`code`]),o(`div`,U,[m(d,{name:`info-circle`,class:`text-blue-400 text-sm mt-0.5 flex-shrink-0`}),$[12]||=o(`p`,{class:`text-sm text-blue-300`},[a(` Use `),o(`code`,{class:`text-slate-200`},`"type": "http"`),a(` (Streamable HTTP), `),o(`strong`,null,`not`),a(),o(`code`,{class:`text-slate-200`},`"type": "sse"`),a(`. The SSE transport holds a session in memory; a daemon restart invalidates it, causing `),o(`code`,{class:`text-slate-200`},`400`),a(` errors and `),o(`em`,null,`"terminated / Failed to parse message"`),a(` noise in the VS Code output panel. Streamable HTTP is stateless and reconnects cleanly every time. Reload the VS Code window after saving the configuration. `)],-1)])]),o(`section`,ne,[$[15]||=o(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[o(`span`,{class:`text-violet-400`},`#`),a(` Available Tools `)],-1),o(`div`,W,[o(`table`,G,[$[14]||=o(`thead`,null,[o(`tr`,{class:`bg-[#0d0d14]`},[o(`th`,{class:`px-4 py-3 text-left text-slate-400`},`Tool`),o(`th`,{class:`px-4 py-3 text-left text-slate-400`},`Description`)])],-1),o(`tbody`,null,[(h(!0),c(p,null,r(i(k),(e,r)=>(h(),c(`tr`,{key:e.name,class:n(r%2==0?`bg-[#0a0a10]`:`bg-[#0d0d14]`)},[o(`td`,K,t(e.name),1),o(`td`,q,t(e.desc),1)],2))),128))])])])]),o(`section`,J,[$[18]||=o(`h2`,{class:`text-2xl font-black mb-6 flex items-center gap-2`},[o(`span`,{class:`text-violet-400`},`#`),a(` Usage Examples `)],-1),o(`div`,Y,[(h(!0),c(p,null,r(i(b),e=>(h(),c(`div`,{key:e.title},[o(`h3`,X,t(e.title),1),o(`div`,Z,[o(`div`,null,[$[16]||=o(`p`,{class:`text-xs text-slate-500 mb-2`},`Request`,-1),m(f,{language:`json`,code:e.request},null,8,[`code`])]),e.response?(h(),c(`div`,re,[$[17]||=o(`p`,{class:`text-xs text-slate-500 mb-2`},`Response`,-1),m(f,{language:`json`,code:e.response},null,8,[`code`])])):u(``,!0)])]))),128))])]),o(`section`,ie,[$[21]||=o(`h2`,{class:`text-2xl font-black mb-6 flex items-center gap-2`},[o(`span`,{class:`text-violet-400`},`#`),a(` Brain Tools `)],-1),$[22]||=o(`p`,{class:`text-slate-400 text-sm mb-6`},` The Brain / Knowledge tools let AI agents ingest project documentation and retrieve semantically-relevant context without reading files directly. `,-1),o(`div`,ae,[(h(!0),c(p,null,r(i(S),e=>(h(),c(`div`,{key:e.title},[o(`h3`,oe,t(e.title),1),o(`div`,se,[o(`div`,null,[$[19]||=o(`p`,{class:`text-xs text-slate-500 mb-2`},`Request`,-1),m(f,{language:`json`,code:e.request},null,8,[`code`])]),e.response?(h(),c(`div`,ce,[$[20]||=o(`p`,{class:`text-xs text-slate-500 mb-2`},`Response`,-1),m(f,{language:`json`,code:e.response},null,8,[`code`])])):u(``,!0)])]))),128))])]),o(`section`,le,[$[23]||=o(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[o(`span`,{class:`text-violet-400`},`#`),a(` Protocol Details `)],-1),o(`div`,ue,[(h(!0),c(p,null,r(i(C),e=>(h(),c(`div`,{key:e.label,class:`rounded-xl border border-white/8 bg-[#0d0d14] p-4 text-center`},[o(`div`,de,t(e.label),1),o(`div`,fe,t(e.value),1)]))),128))]),$[24]||=o(`p`,{class:`text-slate-500 text-sm mt-4`},[a(` The MCP server supports both `),o(`code`,{class:`text-slate-300`},`initialize`),a(` and `),o(`code`,{class:`text-slate-300`},`tools/list`),a(` lifecycle methods, and all tool invocations via `),o(`code`,{class:`text-slate-300`},`tools/call`),a(`. `)],-1)]),o(`section`,pe,[$[28]||=o(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[o(`span`,{class:`text-violet-400`},`#`),a(` Troubleshooting `)],-1),o(`div`,me,[m(d,{name:`exclamation-triangle`,class:`text-red-400 text-sm mt-0.5 flex-shrink-0`}),$[25]||=o(`p`,{class:`text-sm text-red-300`},[o(`strong`,null,`Connection refused:`),a(` Make sure the nexus-daemon is running and the MCP port (default 63988) is not blocked by a firewall. `)],-1)]),o(`div`,Q,[m(d,{name:`info-circle`,class:`text-blue-400 text-sm mt-0.5 flex-shrink-0`}),$[26]||=o(`p`,{class:`text-sm text-blue-300`},[o(`strong`,null,`No tools appearing:`),a(` Verify the URL in `),o(`code`,null,`claude_desktop_config.json`),a(` ends with `),o(`code`,null,`/mcp`),a(` (not just the host:port). `)],-1)]),o(`div`,he,[o(`table`,ge,[$[27]||=o(`thead`,null,[o(`tr`,{class:`bg-[#0d0d14]`},[o(`th`,{class:`px-4 py-3 text-left text-slate-400`},`Issue`),o(`th`,{class:`px-4 py-3 text-left text-slate-400`},`Solution`)])],-1),o(`tbody`,null,[(h(!0),c(p,null,r(i(w),(e,r)=>(h(),c(`tr`,{key:e.issue,class:n(r%2==0?`bg-[#0a0a10]`:`bg-[#0d0d14]`)},[o(`td`,_e,t(e.issue),1),o(`td`,{class:`px-4 py-2.5 text-slate-400 text-xs`,innerHTML:e.solution},null,8,ve)],2))),128))])])])]),o(`section`,ye,[$[30]||=s(`<h2 class="text-2xl font-black mb-4 flex items-center gap-2"><span class="text-violet-400">#</span> GitHub Actions </h2><p class="text-slate-400 text-sm mb-6"> Use nexus-orchestrator directly inside any GitHub Actions workflow. The Node.js 20 action downloads a local daemon, resolves agent identities from <a href="https://github.com/el-j/agency-agents" target="_blank" rel="noopener" class="text-violet-400 hover:text-violet-300">el-j/agency-agents</a>, submits your task to the LLM, polls until completion, and returns the result as step outputs — all in a single step. </p><h3 class="text-lg font-bold mb-3 text-slate-200">Basic Usage</h3>`,3),m(f,{language:`yaml`,code:i(T),class:`mb-8`},null,8,[`code`]),$[31]||=s(`<h3 class="text-lg font-bold mb-3 text-slate-200">Built-in Agent Identities</h3><p class="text-slate-400 text-sm mb-4"> Use the <code class="text-violet-300">agent</code> input to select a specialist from <a href="https://github.com/el-j/agency-agents" target="_blank" rel="noopener" class="text-violet-400 hover:text-violet-300">el-j/agency-agents</a> — no separate step required. The system prompt is fetched and prepended automatically. Use <code class="text-violet-300">agents</code> for a named swarm or <code class="text-violet-300">agent_category</code> to load every agent in a category. </p>`,2),m(f,{language:`yaml`,code:i(E),class:`mb-4`},null,8,[`code`]),m(f,{language:`yaml`,code:i(D),class:`mb-8`},null,8,[`code`]),$[32]||=o(`h3`,{class:`text-lg font-bold mb-3 text-slate-200`},`Key Inputs`,-1),o(`div`,be,[o(`table`,xe,[$[29]||=o(`thead`,null,[o(`tr`,{class:`bg-[#0d0d14]`},[o(`th`,{class:`px-4 py-3 text-left text-slate-400`},`Input`),o(`th`,{class:`px-4 py-3 text-left text-slate-400`},`Default`),o(`th`,{class:`px-4 py-3 text-left text-slate-400`},`Description`)])],-1),o(`tbody`,null,[(h(!0),c(p,null,r(i(O),(e,r)=>(h(),c(`tr`,{key:e.name,class:n(r%2==0?`bg-[#0a0a10]`:`bg-[#0d0d14]`)},[o(`td`,Se,t(e.name),1),o(`td`,Ce,t(e.default),1),o(`td`,we,t(e.desc),1)],2))),128))])])])])])])]))}});export{$ as default};