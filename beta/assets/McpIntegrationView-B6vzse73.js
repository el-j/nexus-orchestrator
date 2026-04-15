import{d as j,c as l,a as e,F as c,r as u,b as m,e as a,f as o,w as p,u as d,s as S,k as h,l as T,m as b,_ as r,p as I,j as L,t as n,n as x,h as f,i}from"./app-BlpOqse5.js";import{s as D}from"./scroll-DpH_xQsu.js";const M={class:"max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16"},E={class:"grid grid-cols-1 lg:grid-cols-4 gap-12"},q={class:"lg:col-span-1"},R={class:"sticky top-24 space-y-1"},U={class:"space-y-1"},G=["href","onClick"],O={class:"lg:col-span-3 space-y-16"},N={id:"claude-desktop",class:"reveal"},B={id:"vscode-setup",class:"reveal"},K={id:"available-tools",class:"reveal"},H={class:"overflow-x-auto rounded-xl border border-white/8"},V={class:"w-full text-sm"},$={class:"px-4 py-2.5 font-mono text-violet-300 text-xs"},F={class:"px-4 py-2.5 text-slate-400 text-xs"},W={id:"examples",class:"reveal"},J={class:"space-y-8"},Q={class:"text-lg font-bold mb-3 text-slate-200"},Y={class:"grid grid-cols-1 lg:grid-cols-2 gap-4"},z={key:0},X={id:"brain-examples",class:"reveal"},Z={class:"space-y-8"},ee={class:"text-lg font-bold mb-3 text-slate-200"},te={class:"grid grid-cols-1 lg:grid-cols-2 gap-4"},se={key:0},oe={id:"protocol",class:"reveal"},ae={class:"grid grid-cols-2 sm:grid-cols-4 gap-3"},ne={class:"text-xs text-slate-500 mb-1"},le={class:"font-mono text-xs text-violet-300 font-bold"},re={id:"troubleshooting",class:"reveal"},ie={class:"overflow-x-auto rounded-xl border border-white/8"},de={class:"w-full text-sm"},ce={class:"px-4 py-2.5 text-red-300 text-xs"},ue=["innerHTML"],pe={id:"github-actions",class:"reveal"},ge={class:"overflow-x-auto rounded-xl border border-white/8"},me={class:"w-full text-sm"},xe={class:"px-4 py-2.5 font-mono text-violet-300 text-xs"},he={class:"px-4 py-2.5 font-mono text-slate-500 text-xs"},be={class:"px-4 py-2.5 text-slate-400 text-xs"},v=`{
  "mcpServers": {
    "nexus-orchestrator": {
      "url": "http://localhost:63988/mcp"
    }
  }
}`,fe=`{
  "servers": {
    "Nexus Orchestrator": {
      "type": "http",
      "url": "http://127.0.0.1:63988/mcp"
    }
  }
}`,ve=`# .github/workflows/ai-task.yml
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
        run: echo "\${{ steps.nexus.outputs.logs }}"`,ye=`# Single agent — identity loaded automatically from el-j/agency-agents
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
          git push`,ke=`# Swarm — load all engineering agents and orchestrate as a team
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
          openai_model: gpt-4o`,Pe=j({__name:"McpIntegrationView",setup(_e){const y=[{id:"what-is-mcp",label:"What is MCP?"},{id:"claude-desktop",label:"Claude Desktop Setup"},{id:"vscode-setup",label:"VS Code Setup"},{id:"available-tools",label:"Available Tools"},{id:"examples",label:"Usage Examples"},{id:"brain-examples",label:"Brain Tools"},{id:"protocol",label:"Protocol Details"},{id:"troubleshooting",label:"Troubleshooting"},{id:"github-actions",label:"GitHub Actions"}],k=[{category:"Tasks",name:"submit_task",desc:"Submit a new code-generation task to the orchestrator"},{category:"Tasks",name:"get_task",desc:"Get the current status and output of a task by ID"},{category:"Tasks",name:"get_queue",desc:"List all tasks currently in the queue"},{category:"Tasks",name:"get_all_tasks",desc:"Return every task regardless of status (QUEUED, PROCESSING, DRAFT, BACKLOG, COMPLETED, FAILED, CANCELLED)"},{category:"Tasks",name:"cancel_task",desc:"Cancel a pending task by ID"},{category:"Tasks",name:"update_task",desc:"Update mutable fields on an existing task (instruction, priority, provider, tags, status)"},{category:"Tasks",name:"create_draft",desc:"Create a draft idea for a project without entering the execution queue"},{category:"Tasks",name:"get_backlog",desc:"List draft and backlog items for a project, ordered by priority"},{category:"Tasks",name:"promote_task",desc:"Promote a draft or backlog task to the execution queue"},{category:"Tasks",name:"claim_task",desc:"Claim a QUEUED task for execution by the specified AI session, transitioning it to PROCESSING"},{category:"Tasks",name:"update_task_status",desc:"Report task completion or failure from the executing AI session"},{category:"Tasks",name:"heartbeat_task",desc:"Keep a PROCESSING task alive — prevents the watchdog from marking it failed"},{category:"Tasks",name:"terminate_ai_session",desc:"Terminate an external AI agent session (SIGTERM by default, SIGKILL when force=true)"},{category:"AI Sessions",name:"register_session",desc:"Announce this AI agent session to nexusOrchestrator for visualisation and orchestration"},{category:"AI Sessions",name:"get_ai_sessions",desc:"Return the list of all known external AI agent sessions registered with this instance"},{category:"AI Sessions",name:"deregister_ai_session",desc:"Soft-disconnect an AI agent session, marking it as disconnected without killing the process"},{category:"AI Sessions",name:"heartbeat_ai_session",desc:"Refresh the last-activity timestamp of an AI session to keep it alive"},{category:"AI Sessions",name:"purge_disconnected_sessions",desc:'Delete all AI sessions with status "disconnected" inactive for more than 2 hours'},{category:"Providers",name:"get_providers",desc:"List available LLM providers and their models"},{category:"Providers",name:"discover_providers",desc:"Scan the local system for installed AI providers/agents and return discovered results"},{category:"Providers",name:"promote_provider",desc:"Promote a discovered provider to an active LLM backend"},{category:"Providers",name:"list_provider_configs",desc:"List all persisted LLM provider configuration records"},{category:"Providers",name:"add_provider_config",desc:"Add a new LLM provider configuration and register it when enabled=true"},{category:"Providers",name:"update_provider_config",desc:"Update an existing LLM provider configuration by ID"},{category:"Providers",name:"remove_provider_config",desc:"Delete a persisted provider configuration and deregister its adapter"},{category:"Brain / Knowledge",name:"get_brain_status",desc:"Check the knowledge repository status for a project"},{category:"Brain / Knowledge",name:"ingest_knowledge",desc:"Parse and ingest a markdown file (often CLAUDE.md) into the project knowledge repository"},{category:"Brain / Knowledge",name:"get_project_context",desc:"Get macro context for a project (Architectures, Conventions, File Maps) bounded by a token budget"},{category:"Brain / Knowledge",name:"get_focused_context",desc:"Get task-specific micro context (Learning, Definitions, Gotchas) bounded by a token budget"},{category:"Brain / Knowledge",name:"search_knowledge",desc:"Perform full-text search across the project's knowledge base"},{category:"Discovery & System",name:"get_discovered_agents",desc:"Return AI agent tools detected on the local system (Claude CLI, VS Code Copilot, etc.)"},{category:"Discovery & System",name:"delegate_to_nexus",desc:"Delegate an AI agent session to the nexus task queue and return the workflow instruction string"},{category:"Discovery & System",name:"get_discovered_plans",desc:"Scan for plan/task/orchestration files in a project directory"},{category:"Discovery & System",name:"howto",desc:"Return a complete integration guide — all tools, workflow patterns, and HTTP endpoint reference"},{category:"Discovery & System",name:"howto_brief",desc:"Get the ultra-compact integration guide (~200 tokens), recommended for small-context models"},{category:"Discovery & System",name:"health",desc:"Check that the nexusOrchestrator daemon is reachable"}],_=[{title:"Submit a Task",request:`{
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
}`},{title:"Get Task Status",request:`{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "get_task",
    "arguments": {
      "id": "abc-123"
    }
  }
}`,response:""},{title:"Check Available Providers",request:`{
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
}`},{title:"Health Check",request:`{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "health"
  }
}`,response:""}],w=[{title:"Check Brain Status",request:`{
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
}`},{title:"Ingest a Knowledge File",request:`{
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
}`},{title:"Get Project Context",request:`{
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
}`,response:""},{title:"Search Knowledge Base",request:`{
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
}`,response:""}],C=[{label:"Protocol",value:"JSON-RPC 2.0"},{label:"Version",value:"2024-11-05"},{label:"Endpoint",value:"POST /mcp"},{label:"Default Port",value:"63988"}],A=[{issue:"Connection refused",solution:'Start nexus-daemon first: <code class="text-slate-300">./nexus-daemon</code>'},{issue:"Port conflict",solution:'Use <code class="text-slate-300">NEXUS_MCP_ADDR=:9090</code> to change the MCP port'},{issue:"No tools in Claude",solution:'Check URL ends with <code class="text-slate-300">/mcp</code>, restart Claude Desktop'},{issue:"Task stuck in QUEUED",solution:'Check <code class="text-slate-300">GET /api/providers</code> — ensure at least one LLM provider is active'}],P=[{name:"instruction",default:"—",desc:"Task text sent to the LLM (required unless task_file is set)"},{name:"task_file",default:"—",desc:"Path to a .md/.txt file whose content becomes the instruction"},{name:"agent",default:'""',desc:'Single agent slug from el-j/agency-agents (e.g. "engineering-backend-architect")'},{name:"agents",default:'""',desc:"Comma-separated agent slugs — generates a combined swarm prompt"},{name:"agent_category",default:'""',desc:'Load ALL agents in a category (e.g. "engineering", "design", "testing")'},{name:"agent_ref",default:"main",desc:"Git ref of el-j/agency-agents to fetch from (branch, tag, SHA)"},{name:"system_prompt",default:'""',desc:"Raw system prompt override — takes precedence over agent/agents/agent_category"},{name:"target_file",default:'""',desc:"Relative path the LLM should write (e.g. src/utils.go)"},{name:"command",default:"execute",desc:'"plan" (orchestration) or "execute" (code generation)'},{name:"openai_api_key",default:'""',desc:"OpenAI API key — enables GPT models on the local daemon"},{name:"anthropic_api_key",default:'""',desc:"Anthropic API key — enables Claude models on the local daemon"},{name:"github_copilot_token",default:'""',desc:"GitHub Copilot token — enables GPT-4o via Copilot"},{name:"timeout_seconds",default:"300",desc:"Seconds to wait for task completion before failing"},{name:"nexus_version",default:"latest",desc:"Release version to install (e.g. v0.2.0)"},{name:"start_daemon",default:"true",desc:"Download + start a local daemon; set false to use daemon_url"},{name:"daemon_url",default:"http://127.0.0.1:63987",desc:"URL of existing daemon (only when start_daemon=false)"}];return(we,t)=>(i(),l("div",M,[e("div",E,[e("aside",q,[e("div",R,[t[0]||(t[0]=e("h4",{class:"text-xs font-semibold uppercase tracking-wider text-slate-500 mb-4"}," On this page ",-1)),e("nav",U,[(i(),l(c,null,u(y,s=>e("a",{key:s.id,href:`#${s.id}`,onClick:L(g=>d(D)(s.id),["prevent"]),class:"block text-sm text-slate-500 hover:text-violet-400 transition-colors py-0.5 cursor-pointer"},n(s.label),9,G)),64))])])]),e("main",O,[t[27]||(t[27]=m('<div class="reveal"><div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-violet-500/30 bg-violet-500/5 text-sm text-violet-300 mb-4"><i class="pi pi-link text-xs"></i> JSON-RPC 2.0 · MCP 2024-11-05 </div><h1 class="text-4xl font-black mb-4"><span class="gradient-text">MCP Integration</span></h1><p class="text-lg text-slate-400"> Connect nexus-orchestrator to Claude Desktop and any MCP-compatible client. </p></div><section id="what-is-mcp" class="reveal"><h2 class="text-2xl font-black mb-4 flex items-center gap-2"><span class="text-violet-400">#</span> What is MCP? </h2><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-6"><p class="text-slate-400 text-sm leading-relaxed"> The <a href="https://modelcontextprotocol.io/" target="_blank" rel="noopener" class="text-violet-400 hover:text-violet-300">Model Context Protocol</a> (MCP) is an open standard for connecting AI assistants to external tools and data sources. nexus-orchestrator implements an MCP server using JSON-RPC 2.0, making it compatible with Claude Desktop and any MCP-aware client. </p></div></section>',2)),e("section",N,[t[5]||(t[5]=e("h2",{class:"text-2xl font-black mb-4 flex items-center gap-2"},[e("span",{class:"text-violet-400"},"#"),a(" Claude Desktop Setup ")],-1)),t[6]||(t[6]=e("p",{class:"text-slate-400 text-sm mb-6"}," Add the following to your Claude Desktop configuration file: ",-1)),o(d(I),{value:"0"},{default:p(()=>[o(d(S),null,{default:p(()=>[o(d(h),{value:"0"},{default:p(()=>[...t[1]||(t[1]=[a("🍎 macOS",-1)])]),_:1}),o(d(h),{value:"1"},{default:p(()=>[...t[2]||(t[2]=[a("🪟 Windows",-1)])]),_:1})]),_:1}),o(d(T),null,{default:p(()=>[o(d(b),{value:"0"},{default:p(()=>[t[3]||(t[3]=e("div",{class:"rounded-xl border border-white/8 bg-[#0d0d14] p-4 mb-3"},[e("p",{class:"text-xs text-slate-500 mb-2"},"Config file path:"),e("code",{class:"text-sm text-violet-300"},"~/Library/Application Support/Claude/claude_desktop_config.json")],-1)),o(r,{language:"json",code:v})]),_:1}),o(d(b),{value:"1"},{default:p(()=>[t[4]||(t[4]=e("div",{class:"rounded-xl border border-white/8 bg-[#0d0d14] p-4 mb-3"},[e("p",{class:"text-xs text-slate-500 mb-2"},"Config file path:"),e("code",{class:"text-sm text-violet-300"},"%APPDATA%\\Claude\\claude_desktop_config.json")],-1)),o(r,{language:"json",code:v})]),_:1})]),_:1})]),_:1}),t[7]||(t[7]=e("div",{class:"mt-4 rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 flex items-start gap-3"},[e("i",{class:"pi pi-info-circle text-amber-400 text-sm mt-0.5 flex-shrink-0"}),e("p",{class:"text-sm text-amber-300"}," Restart Claude Desktop after editing the configuration. The nexus-orchestrator tools will appear in Claude's tool palette. Make sure the nexus-daemon is running before starting Claude Desktop. ")],-1))]),e("section",B,[t[8]||(t[8]=m('<h2 class="text-2xl font-black mb-4 flex items-center gap-2"><span class="text-violet-400">#</span> VS Code Setup </h2><p class="text-slate-400 text-sm mb-4"> Add the following to your VS Code MCP configuration file: </p><div class="grid grid-cols-1 sm:grid-cols-2 gap-3 mb-4"><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4"><p class="text-xs text-slate-500 mb-1">macOS / Linux</p><code class="text-xs text-violet-300">~/Library/Application Support/Code/User/mcp.json</code></div><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4"><p class="text-xs text-slate-500 mb-1">Windows</p><code class="text-xs text-violet-300">%APPDATA%\\Code\\User\\mcp.json</code></div></div>',3)),o(r,{language:"json",code:fe}),t[9]||(t[9]=m('<div class="mt-4 rounded-xl border border-blue-500/20 bg-blue-500/5 p-4 flex items-start gap-3"><i class="pi pi-info-circle text-blue-400 text-sm mt-0.5 flex-shrink-0"></i><p class="text-sm text-blue-300"> Use <code class="text-slate-200">&quot;type&quot;: &quot;http&quot;</code> (Streamable HTTP), <strong>not</strong> <code class="text-slate-200">&quot;type&quot;: &quot;sse&quot;</code>. The SSE transport holds a session in memory; a daemon restart invalidates it, causing <code class="text-slate-200">400</code> errors and <em>&quot;terminated / Failed to parse message&quot;</em> noise in the VS Code output panel. Streamable HTTP is stateless and reconnects cleanly every time. Reload the VS Code window after saving the configuration. </p></div>',1))]),e("section",K,[t[11]||(t[11]=e("h2",{class:"text-2xl font-black mb-4 flex items-center gap-2"},[e("span",{class:"text-violet-400"},"#"),a(" Available Tools ")],-1)),e("div",H,[e("table",V,[t[10]||(t[10]=e("thead",null,[e("tr",{class:"bg-[#0d0d14]"},[e("th",{class:"px-4 py-3 text-left text-slate-400"},"Tool"),e("th",{class:"px-4 py-3 text-left text-slate-400"},"Description")])],-1)),e("tbody",null,[(i(),l(c,null,u(k,(s,g)=>e("tr",{key:s.name,class:x(g%2===0?"bg-[#0a0a10]":"bg-[#0d0d14]")},[e("td",$,n(s.name),1),e("td",F,n(s.desc),1)],2)),64))])])])]),e("section",W,[t[14]||(t[14]=e("h2",{class:"text-2xl font-black mb-6 flex items-center gap-2"},[e("span",{class:"text-violet-400"},"#"),a(" Usage Examples ")],-1)),e("div",J,[(i(),l(c,null,u(_,s=>e("div",{key:s.title},[e("h3",Q,n(s.title),1),e("div",Y,[e("div",null,[t[12]||(t[12]=e("p",{class:"text-xs text-slate-500 mb-2"},"Request",-1)),o(r,{language:"json",code:s.request},null,8,["code"])]),s.response?(i(),l("div",z,[t[13]||(t[13]=e("p",{class:"text-xs text-slate-500 mb-2"},"Response",-1)),o(r,{language:"json",code:s.response},null,8,["code"])])):f("",!0)])])),64))])]),e("section",X,[t[17]||(t[17]=e("h2",{class:"text-2xl font-black mb-6 flex items-center gap-2"},[e("span",{class:"text-violet-400"},"#"),a(" Brain Tools ")],-1)),t[18]||(t[18]=e("p",{class:"text-slate-400 text-sm mb-6"}," The Brain / Knowledge tools let AI agents ingest project documentation and retrieve semantically-relevant context without reading files directly. ",-1)),e("div",Z,[(i(),l(c,null,u(w,s=>e("div",{key:s.title},[e("h3",ee,n(s.title),1),e("div",te,[e("div",null,[t[15]||(t[15]=e("p",{class:"text-xs text-slate-500 mb-2"},"Request",-1)),o(r,{language:"json",code:s.request},null,8,["code"])]),s.response?(i(),l("div",se,[t[16]||(t[16]=e("p",{class:"text-xs text-slate-500 mb-2"},"Response",-1)),o(r,{language:"json",code:s.response},null,8,["code"])])):f("",!0)])])),64))])]),e("section",oe,[t[19]||(t[19]=e("h2",{class:"text-2xl font-black mb-4 flex items-center gap-2"},[e("span",{class:"text-violet-400"},"#"),a(" Protocol Details ")],-1)),e("div",ae,[(i(),l(c,null,u(C,s=>e("div",{key:s.label,class:"rounded-xl border border-white/8 bg-[#0d0d14] p-4 text-center"},[e("div",ne,n(s.label),1),e("div",le,n(s.value),1)])),64))]),t[20]||(t[20]=e("p",{class:"text-slate-500 text-sm mt-4"},[a(" The MCP server supports both "),e("code",{class:"text-slate-300"},"initialize"),a(" and "),e("code",{class:"text-slate-300"},"tools/list"),a(" lifecycle methods, and all tool invocations via "),e("code",{class:"text-slate-300"},"tools/call"),a(". ")],-1))]),e("section",re,[t[22]||(t[22]=m('<h2 class="text-2xl font-black mb-4 flex items-center gap-2"><span class="text-violet-400">#</span> Troubleshooting </h2><div class="rounded-xl border border-red-500/20 bg-red-500/5 p-4 flex items-start gap-3 mb-4"><i class="pi pi-exclamation-triangle text-red-400 text-sm mt-0.5 flex-shrink-0"></i><p class="text-sm text-red-300"><strong>Connection refused:</strong> Make sure the nexus-daemon is running and the MCP port (default 63988) is not blocked by a firewall. </p></div><div class="rounded-xl border border-blue-500/20 bg-blue-500/5 p-4 flex items-start gap-3 mb-6"><i class="pi pi-info-circle text-blue-400 text-sm mt-0.5 flex-shrink-0"></i><p class="text-sm text-blue-300"><strong>No tools appearing:</strong> Verify the URL in <code>claude_desktop_config.json</code> ends with <code>/mcp</code> (not just the host:port). </p></div>',3)),e("div",ie,[e("table",de,[t[21]||(t[21]=e("thead",null,[e("tr",{class:"bg-[#0d0d14]"},[e("th",{class:"px-4 py-3 text-left text-slate-400"},"Issue"),e("th",{class:"px-4 py-3 text-left text-slate-400"},"Solution")])],-1)),e("tbody",null,[(i(),l(c,null,u(A,(s,g)=>e("tr",{key:s.issue,class:x(g%2===0?"bg-[#0a0a10]":"bg-[#0d0d14]")},[e("td",ce,n(s.issue),1),e("td",{class:"px-4 py-2.5 text-slate-400 text-xs",innerHTML:s.solution},null,8,ue)],2)),64))])])])]),e("section",pe,[t[24]||(t[24]=m('<h2 class="text-2xl font-black mb-4 flex items-center gap-2"><span class="text-violet-400">#</span> GitHub Actions </h2><p class="text-slate-400 text-sm mb-6"> Use nexus-orchestrator directly inside any GitHub Actions workflow. The Node.js 20 action downloads a local daemon, resolves agent identities from <a href="https://github.com/el-j/agency-agents" target="_blank" rel="noopener" class="text-violet-400 hover:text-violet-300">el-j/agency-agents</a>, submits your task to the LLM, polls until completion, and returns the result as step outputs — all in a single step. </p><h3 class="text-lg font-bold mb-3 text-slate-200">Basic Usage</h3>',3)),o(r,{language:"yaml",code:ve,class:"mb-8"}),t[25]||(t[25]=m('<h3 class="text-lg font-bold mb-3 text-slate-200">Built-in Agent Identities</h3><p class="text-slate-400 text-sm mb-4"> Use the <code class="text-violet-300">agent</code> input to select a specialist from <a href="https://github.com/el-j/agency-agents" target="_blank" rel="noopener" class="text-violet-400 hover:text-violet-300">el-j/agency-agents</a> — no separate step required. The system prompt is fetched and prepended automatically. Use <code class="text-violet-300">agents</code> for a named swarm or <code class="text-violet-300">agent_category</code> to load every agent in a category. </p>',2)),o(r,{language:"yaml",code:ye,class:"mb-4"}),o(r,{language:"yaml",code:ke,class:"mb-8"}),t[26]||(t[26]=e("h3",{class:"text-lg font-bold mb-3 text-slate-200"},"Key Inputs",-1)),e("div",ge,[e("table",me,[t[23]||(t[23]=e("thead",null,[e("tr",{class:"bg-[#0d0d14]"},[e("th",{class:"px-4 py-3 text-left text-slate-400"},"Input"),e("th",{class:"px-4 py-3 text-left text-slate-400"},"Default"),e("th",{class:"px-4 py-3 text-left text-slate-400"},"Description")])],-1)),e("tbody",null,[(i(),l(c,null,u(P,(s,g)=>e("tr",{key:s.name,class:x(g%2===0?"bg-[#0a0a10]":"bg-[#0d0d14]")},[e("td",xe,n(s.name),1),e("td",he,n(s.default),1),e("td",be,n(s.desc),1)],2)),64))])])])])])])]))}});export{Pe as default};
