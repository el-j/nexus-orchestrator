import{C as e,S as t,_ as n,d as r,f as i,h as a,i as o,l as s,m as c,n as ee,o as l,p as u,s as d,t as f,u as p,v as m,x as h,y as g}from"./app-BmR6q0Jo.js";import{t as te}from"./scroll-DdeK3J_A.js";var _=[{id:`what-is-mcp`,label:`What is MCP?`},{id:`claude-desktop`,label:`Claude Desktop Setup`},{id:`vscode-setup`,label:`VS Code Setup`},{id:`available-tools`,label:`Available Tools`},{id:`examples`,label:`Usage Examples`},{id:`brain-examples`,label:`Brain Tools`},{id:`protocol`,label:`Protocol Details`},{id:`troubleshooting`,label:`Troubleshooting`},{id:`github-actions`,label:`GitHub Actions`}],v=`{
  "mcpServers": {
    "nexus-orchestrator": {
      "url": "http://localhost:63988/mcp"
    }
  }
}`,y=[{title:`Submit a Task`,request:`{
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
}`,response:``}],b=`{
  "servers": {
    "Nexus Orchestrator": {
      "type": "http",
      "url": "http://127.0.0.1:63988/mcp"
    }
  }
}`,x=[{title:`Check Brain Status`,request:`{
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
}`,response:``}],S=[{label:`Protocol`,value:`JSON-RPC 2.0`},{label:`Version`,value:`2024-11-05`},{label:`Endpoint`,value:`POST /mcp`},{label:`Default Port`,value:`63988`}],C=[{issue:`Connection refused`,solution:`Start nexus-daemon first: <code class="text-slate-300">./nexus-daemon</code>`},{issue:`Port conflict`,solution:`Use <code class="text-slate-300">NEXUS_MCP_ADDR=:9090</code> to change the MCP port`},{issue:`No tools in Claude`,solution:`Check URL ends with <code class="text-slate-300">/mcp</code>, restart Claude Desktop`},{issue:`Task stuck in QUEUED`,solution:`Check <code class="text-slate-300">GET /api/providers</code> — ensure at least one LLM provider is active`}],w=`# .github/workflows/ai-task.yml
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
        run: echo "\${{ steps.nexus.outputs.logs }}"`,T=`# Single agent — identity loaded automatically from el-j/agency-agents
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
          git push`,E=`# Swarm — load all engineering agents and orchestrate as a team
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
          openai_model: gpt-4o`,D=[{name:`instruction`,default:`—`,desc:`Task text sent to the LLM (required unless task_file is set)`},{name:`task_file`,default:`—`,desc:`Path to a .md/.txt file whose content becomes the instruction`},{name:`agent`,default:`""`,desc:`Single agent slug from el-j/agency-agents (e.g. "engineering-backend-architect")`},{name:`agents`,default:`""`,desc:`Comma-separated agent slugs — generates a combined swarm prompt`},{name:`agent_category`,default:`""`,desc:`Load ALL agents in a category (e.g. "engineering", "design", "testing")`},{name:`agent_ref`,default:`main`,desc:`Git ref of el-j/agency-agents to fetch from (branch, tag, SHA)`},{name:`system_prompt`,default:`""`,desc:`Raw system prompt override — takes precedence over agent/agents/agent_category`},{name:`target_file`,default:`""`,desc:`Relative path the LLM should write (e.g. src/utils.go)`},{name:`command`,default:`auto`,desc:`"plan" (orchestration), "execute" (code generation, needs a prior completed plan) or "auto"`},{name:`openai_api_key`,default:`""`,desc:`OpenAI API key — enables GPT models on the local daemon`},{name:`anthropic_api_key`,default:`""`,desc:`Anthropic API key — enables Claude models on the local daemon`},{name:`github_copilot_token`,default:`""`,desc:`GitHub Copilot token — enables GPT-4o via Copilot`},{name:`timeout_seconds`,default:`300`,desc:`Seconds to wait for task completion before failing`},{name:`nexus_version`,default:`latest`,desc:`Release version to install (e.g. v0.2.0)`},{name:`start_daemon`,default:`true`,desc:`Download + start a local daemon; set false to use daemon_url`},{name:`daemon_url`,default:`http://127.0.0.1:63987`,desc:`URL of existing daemon (only when start_daemon=false)`}],O=[{category:`Tasks`,name:`submit_task`,desc:`Submit a new code-generation task to the orchestrator`},{category:`Tasks`,name:`get_task`,desc:`Get the current status and output of a task by ID`},{category:`Tasks`,name:`wait_for_task`,desc:`Wait until a task has finished and return it; streams the reply as progress notifications when asked`},{category:`Tasks`,name:`get_queue`,desc:`List all tasks currently in the queue`},{category:`Tasks`,name:`get_all_tasks`,desc:`Return every task regardless of status (QUEUED, PROCESSING, DRAFT, BACKLOG, COMPLETED, FAILED, CANCELLED)`},{category:`Tasks`,name:`cancel_task`,desc:`Cancel a pending task by ID`},{category:`Tasks`,name:`update_task`,desc:`Update mutable fields on an existing task (instruction, priority, provider, tags, status)`},{category:`Tasks`,name:`create_draft`,desc:`Create a draft idea for a project without entering the execution queue`},{category:`Tasks`,name:`get_backlog`,desc:`List draft and backlog items for a project, ordered by priority`},{category:`Tasks`,name:`promote_task`,desc:`Promote a draft or backlog task to the execution queue`},{category:`Tasks`,name:`claim_task`,desc:`Claim a QUEUED task for execution by the specified AI session, transitioning it to PROCESSING`},{category:`Tasks`,name:`update_task_status`,desc:`Report task completion or failure from the executing AI session`},{category:`Tasks`,name:`heartbeat_task`,desc:`Keep a PROCESSING task alive — prevents the watchdog from marking it failed`},{category:`Tasks`,name:`terminate_ai_session`,desc:`Terminate an external AI agent session (SIGTERM by default, SIGKILL when force=true)`},{category:`AI Sessions`,name:`register_session`,desc:`Announce this AI agent session to nexusOrchestrator for visualisation and orchestration`},{category:`AI Sessions`,name:`get_ai_sessions`,desc:`Return the list of all known external AI agent sessions registered with this instance`},{category:`AI Sessions`,name:`deregister_ai_session`,desc:`Soft-disconnect an AI agent session, marking it as disconnected without killing the process`},{category:`AI Sessions`,name:`heartbeat_ai_session`,desc:`Refresh the last-activity timestamp of an AI session to keep it alive`},{category:`AI Sessions`,name:`purge_disconnected_sessions`,desc:`Delete all AI sessions with status "disconnected" inactive for more than 2 hours`},{category:`Providers`,name:`get_providers`,desc:`List available LLM providers and their models`},{category:`Providers`,name:`discover_providers`,desc:`Scan the local system for installed AI providers/agents and return discovered results`},{category:`Providers`,name:`promote_provider`,desc:`Promote a discovered provider to an active LLM backend`},{category:`Providers`,name:`list_provider_configs`,desc:`List all persisted LLM provider configuration records`},{category:`Providers`,name:`add_provider_config`,desc:`Add a new LLM provider configuration and register it when enabled=true`},{category:`Providers`,name:`update_provider_config`,desc:`Update an existing LLM provider configuration by ID`},{category:`Providers`,name:`remove_provider_config`,desc:`Delete a persisted provider configuration and deregister its adapter`},{category:`Brain / Knowledge`,name:`get_brain_status`,desc:`Check the knowledge repository status for a project`},{category:`Brain / Knowledge`,name:`ingest_knowledge`,desc:`Parse and ingest a markdown file (often CLAUDE.md) into the project knowledge repository`},{category:`Brain / Knowledge`,name:`get_project_context`,desc:`Get macro context for a project (Architectures, Conventions, File Maps) bounded by a token budget`},{category:`Brain / Knowledge`,name:`get_onboarding_context`,desc:`Get a token-budgeted (< 800 tokens) onboarding summary for agent bootstrap: stack, invariants, test commands, active plan and tasks`},{category:`Brain / Knowledge`,name:`get_focused_context`,desc:`Get task-specific micro context (Learning, Definitions, Gotchas) bounded by a token budget`},{category:`Brain / Knowledge`,name:`search_knowledge`,desc:`Perform full-text search across the project's knowledge base`},{category:`Brain / Knowledge`,name:`init_project`,desc:`Auto-ingest CLAUDE.md and initialize a project's knowledge base in one step`},{category:`Brain / Knowledge`,name:`list_knowledge`,desc:`List all knowledge documents stored for a project, optionally filtered by kind`},{category:`Brain / Knowledge`,name:`delete_knowledge`,desc:`Delete a knowledge document by ID from the project repository`},{category:`Brain / Knowledge`,name:`get_file_map`,desc:`Retrieve the cached or structured file map knowledge document for a project`},{category:`Discovery & System`,name:`get_discovered_agents`,desc:`Return AI agent tools detected on the local system (Claude CLI, VS Code Copilot, etc.)`},{category:`Discovery & System`,name:`delegate_to_nexus`,desc:`Delegate an AI agent session to the nexus task queue and return the workflow instruction string`},{category:`Discovery & System`,name:`get_discovered_plans`,desc:`Scan for plan/task/orchestration files in a project directory`},{category:`Discovery & System`,name:`howto`,desc:`Return a complete integration guide — all tools, workflow patterns, and HTTP endpoint reference`},{category:`Discovery & System`,name:`howto_brief`,desc:`Get the ultra-compact integration guide (~200 tokens), recommended for small-context models`},{category:`Discovery & System`,name:`health`,desc:`Check that the nexusOrchestrator daemon is reachable`}],k={class:`max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16`},A={class:`grid grid-cols-1 lg:grid-cols-4 gap-12`},j={class:`lg:col-span-1`},M={class:`sticky top-24 space-y-1`},N={class:`space-y-1`},P=[`href`,`onClick`],F={class:`lg:col-span-3 space-y-16`},I={class:`reveal`},L={class:`inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-violet-500/30 bg-violet-500/5 text-sm text-violet-300 mb-4`},R={id:`claude-desktop`,class:`reveal`},z={class:`mt-4 rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 flex items-start gap-3`},B={id:`vscode-setup`,class:`reveal`},V={class:`mt-4 rounded-xl border border-blue-500/20 bg-blue-500/5 p-4 flex items-start gap-3`},H={id:`available-tools`,class:`reveal`},U={class:`overflow-x-auto rounded-xl border border-white/8`},W={class:`w-full text-sm`},ne={class:`px-4 py-2.5 font-mono text-violet-300 text-xs`},G={class:`px-4 py-2.5 text-slate-400 text-xs`},K={id:`examples`,class:`reveal`},q={class:`space-y-8`},J={class:`text-lg font-bold mb-3 text-slate-200`},Y={class:`grid grid-cols-1 lg:grid-cols-2 gap-4`},X={key:0},Z={id:`brain-examples`,class:`reveal`},re={class:`space-y-8`},ie={class:`text-lg font-bold mb-3 text-slate-200`},ae={class:`grid grid-cols-1 lg:grid-cols-2 gap-4`},oe={key:0},se={id:`protocol`,class:`reveal`},ce={class:`grid grid-cols-2 sm:grid-cols-4 gap-3`},le={class:`text-xs text-slate-500 mb-1`},ue={class:`font-mono text-xs text-violet-300 font-bold`},de={id:`troubleshooting`,class:`reveal`},fe={class:`rounded-xl border border-red-500/20 bg-red-500/5 p-4 flex items-start gap-3 mb-4`},pe={class:`rounded-xl border border-blue-500/20 bg-blue-500/5 p-4 flex items-start gap-3 mb-6`},me={class:`overflow-x-auto rounded-xl border border-white/8`},Q={class:`w-full text-sm`},he={class:`px-4 py-2.5 text-red-300 text-xs`},ge=[`innerHTML`],_e={id:`github-actions`,class:`reveal`},ve={class:`overflow-x-auto rounded-xl border border-white/8`},ye={class:`w-full text-sm`},be={class:`px-4 py-2.5 font-mono text-violet-300 text-xs`},xe={class:`px-4 py-2.5 font-mono text-slate-500 text-xs`},Se={class:`px-4 py-2.5 text-slate-400 text-xs`},$=a({__name:`McpIntegrationView`,setup(a){return(a,$)=>(n(),r(`div`,k,[s(`div`,A,[s(`aside`,j,[s(`div`,M,[$[0]||=s(`h4`,{class:`text-xs font-semibold uppercase tracking-wider text-slate-500 mb-4`},` On this page `,-1),s(`nav`,N,[(n(!0),r(d,null,m(h(_),t=>(n(),r(`a`,{key:t.id,href:`#${t.id}`,onClick:l(e=>h(te)(t.id),[`prevent`]),class:`block text-sm text-slate-500 hover:text-violet-400 transition-colors py-0.5 cursor-pointer`},e(t.label),9,P))),128))])])]),s(`main`,F,[s(`div`,I,[s(`div`,L,[c(o,{name:`link`,class:`text-xs`}),$[1]||=u(` JSON-RPC 2.0 · MCP 2024-11-05 `,-1)]),$[2]||=s(`h1`,{class:`text-4xl font-black mb-4`},[s(`span`,{class:`gradient-text`},`MCP Integration`)],-1),$[3]||=s(`p`,{class:`text-lg text-slate-400`},` Connect nexus-orchestrator to Claude Desktop and any MCP-compatible client. `,-1)]),$[30]||=i(`<section id="what-is-mcp" class="reveal"><h2 class="text-2xl font-black mb-4 flex items-center gap-2"><span class="text-violet-400">#</span> What is MCP? </h2><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-6"><p class="text-slate-400 text-sm leading-relaxed"> The <a href="https://modelcontextprotocol.io/" target="_blank" rel="noopener" class="text-violet-400 hover:text-violet-300">Model Context Protocol</a> (MCP) is an open standard for connecting AI assistants to external tools and data sources. nexus-orchestrator implements an MCP server using JSON-RPC 2.0, making it compatible with Claude Desktop and any MCP-aware client. </p></div></section>`,1),s(`section`,R,[$[7]||=s(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[s(`span`,{class:`text-violet-400`},`#`),u(` Claude Desktop Setup `)],-1),$[8]||=s(`p`,{class:`text-slate-400 text-sm mb-6`},` Add the following to your Claude Desktop configuration file: `,-1),c(ee,{tabs:[`🍎 macOS`,`🪟 Windows`],label:`Operating system`},{"panel-0":g(()=>[$[4]||=s(`div`,{class:`rounded-xl border border-white/8 bg-[#0d0d14] p-4 mb-3`},[s(`p`,{class:`text-xs text-slate-500 mb-2`},`Config file path:`),s(`code`,{class:`text-sm text-violet-300`},`~/Library/Application Support/Claude/claude_desktop_config.json`)],-1),c(f,{language:`json`,code:h(v)},null,8,[`code`])]),"panel-1":g(()=>[$[5]||=s(`div`,{class:`rounded-xl border border-white/8 bg-[#0d0d14] p-4 mb-3`},[s(`p`,{class:`text-xs text-slate-500 mb-2`},`Config file path:`),s(`code`,{class:`text-sm text-violet-300`},`%APPDATA%\\Claude\\claude_desktop_config.json`)],-1),c(f,{language:`json`,code:h(v)},null,8,[`code`])]),_:1}),s(`div`,z,[c(o,{name:`info-circle`,class:`text-amber-400 text-sm mt-0.5 flex-shrink-0`}),$[6]||=s(`p`,{class:`text-sm text-amber-300`},` Restart Claude Desktop after editing the configuration. The nexus-orchestrator tools will appear in Claude's tool palette. Make sure the nexus-daemon is running before starting Claude Desktop. `,-1)])]),s(`section`,B,[$[10]||=i(`<h2 class="text-2xl font-black mb-4 flex items-center gap-2"><span class="text-violet-400">#</span> VS Code Setup </h2><p class="text-slate-400 text-sm mb-4"> Add the following to your VS Code MCP configuration file: </p><div class="grid grid-cols-1 sm:grid-cols-2 gap-3 mb-4"><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4"><p class="text-xs text-slate-500 mb-1">macOS / Linux</p><code class="text-xs text-violet-300">~/Library/Application Support/Code/User/mcp.json</code></div><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4"><p class="text-xs text-slate-500 mb-1">Windows</p><code class="text-xs text-violet-300">%APPDATA%\\Code\\User\\mcp.json</code></div></div>`,3),c(f,{language:`json`,code:h(b)},null,8,[`code`]),s(`div`,V,[c(o,{name:`info-circle`,class:`text-blue-400 text-sm mt-0.5 flex-shrink-0`}),$[9]||=s(`p`,{class:`text-sm text-blue-300`},[u(` Use `),s(`code`,{class:`text-slate-200`},`"type": "http"`),u(` (Streamable HTTP), `),s(`strong`,null,`not`),u(),s(`code`,{class:`text-slate-200`},`"type": "sse"`),u(`. The SSE transport holds a session in memory; a daemon restart invalidates it, causing `),s(`code`,{class:`text-slate-200`},`400`),u(` errors and `),s(`em`,null,`"terminated / Failed to parse message"`),u(` noise in the VS Code output panel. Streamable HTTP is stateless and reconnects cleanly every time. Reload the VS Code window after saving the configuration. `)],-1)])]),s(`section`,H,[$[12]||=s(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[s(`span`,{class:`text-violet-400`},`#`),u(` Available Tools `)],-1),s(`div`,U,[s(`table`,W,[$[11]||=s(`thead`,null,[s(`tr`,{class:`bg-[#0d0d14]`},[s(`th`,{class:`px-4 py-3 text-left text-slate-400`},`Tool`),s(`th`,{class:`px-4 py-3 text-left text-slate-400`},`Description`)])],-1),s(`tbody`,null,[(n(!0),r(d,null,m(h(O),(i,a)=>(n(),r(`tr`,{key:i.name,class:t(a%2==0?`bg-[#0a0a10]`:`bg-[#0d0d14]`)},[s(`td`,ne,e(i.name),1),s(`td`,G,e(i.desc),1)],2))),128))])])])]),s(`section`,K,[$[15]||=s(`h2`,{class:`text-2xl font-black mb-6 flex items-center gap-2`},[s(`span`,{class:`text-violet-400`},`#`),u(` Usage Examples `)],-1),s(`div`,q,[(n(!0),r(d,null,m(h(y),t=>(n(),r(`div`,{key:t.title},[s(`h3`,J,e(t.title),1),s(`div`,Y,[s(`div`,null,[$[13]||=s(`p`,{class:`text-xs text-slate-500 mb-2`},`Request`,-1),c(f,{language:`json`,code:t.request},null,8,[`code`])]),t.response?(n(),r(`div`,X,[$[14]||=s(`p`,{class:`text-xs text-slate-500 mb-2`},`Response`,-1),c(f,{language:`json`,code:t.response},null,8,[`code`])])):p(``,!0)])]))),128))])]),s(`section`,Z,[$[18]||=s(`h2`,{class:`text-2xl font-black mb-6 flex items-center gap-2`},[s(`span`,{class:`text-violet-400`},`#`),u(` Brain Tools `)],-1),$[19]||=s(`p`,{class:`text-slate-400 text-sm mb-6`},` The Brain / Knowledge tools let AI agents ingest project documentation and retrieve semantically-relevant context without reading files directly. `,-1),s(`div`,re,[(n(!0),r(d,null,m(h(x),t=>(n(),r(`div`,{key:t.title},[s(`h3`,ie,e(t.title),1),s(`div`,ae,[s(`div`,null,[$[16]||=s(`p`,{class:`text-xs text-slate-500 mb-2`},`Request`,-1),c(f,{language:`json`,code:t.request},null,8,[`code`])]),t.response?(n(),r(`div`,oe,[$[17]||=s(`p`,{class:`text-xs text-slate-500 mb-2`},`Response`,-1),c(f,{language:`json`,code:t.response},null,8,[`code`])])):p(``,!0)])]))),128))])]),s(`section`,se,[$[20]||=s(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[s(`span`,{class:`text-violet-400`},`#`),u(` Protocol Details `)],-1),s(`div`,ce,[(n(!0),r(d,null,m(h(S),t=>(n(),r(`div`,{key:t.label,class:`rounded-xl border border-white/8 bg-[#0d0d14] p-4 text-center`},[s(`div`,le,e(t.label),1),s(`div`,ue,e(t.value),1)]))),128))]),$[21]||=s(`p`,{class:`text-slate-500 text-sm mt-4`},[u(` The MCP server supports both `),s(`code`,{class:`text-slate-300`},`initialize`),u(` and `),s(`code`,{class:`text-slate-300`},`tools/list`),u(` lifecycle methods, and all tool invocations via `),s(`code`,{class:`text-slate-300`},`tools/call`),u(`. `)],-1)]),s(`section`,de,[$[25]||=s(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[s(`span`,{class:`text-violet-400`},`#`),u(` Troubleshooting `)],-1),s(`div`,fe,[c(o,{name:`exclamation-triangle`,class:`text-red-400 text-sm mt-0.5 flex-shrink-0`}),$[22]||=s(`p`,{class:`text-sm text-red-300`},[s(`strong`,null,`Connection refused:`),u(` Make sure the nexus-daemon is running and the MCP port (default 63988) is not blocked by a firewall. `)],-1)]),s(`div`,pe,[c(o,{name:`info-circle`,class:`text-blue-400 text-sm mt-0.5 flex-shrink-0`}),$[23]||=s(`p`,{class:`text-sm text-blue-300`},[s(`strong`,null,`No tools appearing:`),u(` Verify the URL in `),s(`code`,null,`claude_desktop_config.json`),u(` ends with `),s(`code`,null,`/mcp`),u(` (not just the host:port). `)],-1)]),s(`div`,me,[s(`table`,Q,[$[24]||=s(`thead`,null,[s(`tr`,{class:`bg-[#0d0d14]`},[s(`th`,{class:`px-4 py-3 text-left text-slate-400`},`Issue`),s(`th`,{class:`px-4 py-3 text-left text-slate-400`},`Solution`)])],-1),s(`tbody`,null,[(n(!0),r(d,null,m(h(C),(i,a)=>(n(),r(`tr`,{key:i.issue,class:t(a%2==0?`bg-[#0a0a10]`:`bg-[#0d0d14]`)},[s(`td`,he,e(i.issue),1),s(`td`,{class:`px-4 py-2.5 text-slate-400 text-xs`,innerHTML:i.solution},null,8,ge)],2))),128))])])])]),s(`section`,_e,[$[27]||=i(`<h2 class="text-2xl font-black mb-4 flex items-center gap-2"><span class="text-violet-400">#</span> GitHub Actions </h2><p class="text-slate-400 text-sm mb-6"> Use nexus-orchestrator directly inside any GitHub Actions workflow. The Node.js 20 action downloads a local daemon, resolves agent identities from <a href="https://github.com/el-j/agency-agents" target="_blank" rel="noopener" class="text-violet-400 hover:text-violet-300">el-j/agency-agents</a>, submits your task to the LLM, polls until completion, and returns the result as step outputs — all in a single step. </p><h3 class="text-lg font-bold mb-3 text-slate-200">Basic Usage</h3>`,3),c(f,{language:`yaml`,code:h(w),class:`mb-8`},null,8,[`code`]),$[28]||=i(`<h3 class="text-lg font-bold mb-3 text-slate-200">Built-in Agent Identities</h3><p class="text-slate-400 text-sm mb-4"> Use the <code class="text-violet-300">agent</code> input to select a specialist from <a href="https://github.com/el-j/agency-agents" target="_blank" rel="noopener" class="text-violet-400 hover:text-violet-300">el-j/agency-agents</a> — no separate step required. The system prompt is fetched and prepended automatically. Use <code class="text-violet-300">agents</code> for a named swarm or <code class="text-violet-300">agent_category</code> to load every agent in a category. </p>`,2),c(f,{language:`yaml`,code:h(T),class:`mb-4`},null,8,[`code`]),c(f,{language:`yaml`,code:h(E),class:`mb-8`},null,8,[`code`]),$[29]||=s(`h3`,{class:`text-lg font-bold mb-3 text-slate-200`},`Key Inputs`,-1),s(`div`,ve,[s(`table`,ye,[$[26]||=s(`thead`,null,[s(`tr`,{class:`bg-[#0d0d14]`},[s(`th`,{class:`px-4 py-3 text-left text-slate-400`},`Input`),s(`th`,{class:`px-4 py-3 text-left text-slate-400`},`Default`),s(`th`,{class:`px-4 py-3 text-left text-slate-400`},`Description`)])],-1),s(`tbody`,null,[(n(!0),r(d,null,m(h(D),(i,a)=>(n(),r(`tr`,{key:i.name,class:t(a%2==0?`bg-[#0a0a10]`:`bg-[#0d0d14]`)},[s(`td`,be,e(i.name),1),s(`td`,xe,e(i.default),1),s(`td`,Se,e(i.desc),1)],2))),128))])])])])])])]))}});export{$ as default};