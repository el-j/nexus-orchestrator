import{D as e,H as t,P as n,T as r,at as i,g as a,h as o,m as s,p as c,r as l,s as u,u as d}from"./runtime-core.esm-bundler-C1RA0-Zv.js";import{c as f,s as p,t as m}from"./app-Bi72j18L.js";import{t as h}from"./scroll-DdeK3J_A.js";var g={class:`max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16`},_={class:`grid grid-cols-1 lg:grid-cols-4 gap-12`},v={class:`lg:col-span-1`},y={class:`sticky top-24 space-y-1`},b={class:`space-y-1`},x=[`href`,`onClick`],S={class:`lg:col-span-3 space-y-16`},C={id:`installation`,class:`reveal`},w={id:`starting-daemon`,class:`reveal`},T={id:`first-task`,class:`reveal`},E={id:`task-management`,class:`reveal`},D={id:`providers`,class:`reveal`},O={id:`dashboard`,class:`reveal`},k={class:`rounded-xl border border-white/8 bg-[#0d0d14] p-6`},A={class:`space-y-2 list-none p-0`},j={id:`testing`,class:`reveal`},M={id:`next-steps`,class:`reveal`},N={class:`grid grid-cols-1 sm:grid-cols-3 gap-4`},P={class:`text-xl mb-2`},F={class:`font-bold text-white text-sm mb-1 group-hover:text-violet-300 transition-colors`},I={class:`text-xs text-slate-500`},L=`# Clone the repository
git clone https://github.com/el-j/nexus-orchestrator.git
cd nexus-orchestrator

# Build all binaries
CGO_ENABLED=1 go build ./...

# Or build specific binaries
CGO_ENABLED=1 go build -o nexus-daemon ./cmd/nexus-daemon/...
CGO_ENABLED=1 go build -o nexus-cli ./cmd/nexus-cli/...`,R=`# Start with default settings
./nexus-daemon
# HTTP API:   http://127.0.0.1:63987
# MCP server: http://127.0.0.1:63988/mcp
# Dashboard:  http://127.0.0.1:63987/ui`,z=`# Use environment variables for custom settings
NEXUS_DB_PATH=/path/to/nexus.db \\
NEXUS_LISTEN_ADDR=:8080 \\
NEXUS_MCP_ADDR=:8081 \\
./nexus-daemon`,B=`# OpenAI
NEXUS_OPENAI_API_KEY=sk-... NEXUS_OPENAI_MODEL=gpt-4o-mini ./nexus-daemon

# Anthropic
NEXUS_ANTHROPIC_API_KEY=sk-ant-... NEXUS_ANTHROPIC_MODEL=claude-3-5-sonnet-20241022 ./nexus-daemon

# GitHub Copilot
NEXUS_GITHUBCOPILOT_TOKEN=ghu_... NEXUS_GITHUBCOPILOT_MODEL=gpt-4o ./nexus-daemon`,V=`# Submit a code-generation task
curl -s -X POST http://localhost:63987/api/tasks \\
  -H "Content-Type: application/json" \\
  -d '{
    "projectPath": "'$PWD'",
    "targetFile": "hello.go",
    "instruction": "Write a Go function that returns Hello World"
  }' | jq .`,H=`{
  "task_id": "a1b2c3d4-...",
  "status": "QUEUED"
}`,U=`# Get task by ID
curl -s http://localhost:63987/api/tasks/TASK_ID | jq .

# List all pending tasks
curl -s http://localhost:63987/api/tasks | jq .`,W=`curl -X DELETE http://localhost:63987/api/tasks/TASK_ID
# Returns 204 No Content on success`,G=`# List all providers
curl -s http://localhost:63987/api/providers | jq .

# Register a new cloud provider
curl -s -X POST http://localhost:63987/api/providers \\
  -H "Content-Type: application/json" \\
  -d '{
    "name": "My OpenAI",
    "kind": "openai-compat",
    "baseURL": "https://api.openai.com/v1",
    "apiKey": "sk-...",
    "model": "gpt-4o-mini"
  }' | jq .

# Remove a provider
curl -X DELETE http://localhost:63987/api/providers/My%20OpenAI`,K=`# Full test suite with race detection
CGO_ENABLED=1 go test -race ./...

# Service tests only
CGO_ENABLED=1 go test ./internal/core/services/...

# Lint
go vet ./...`,q=a({__name:`GettingStartedView`,setup(a){let q=[{id:`prerequisites`,label:`Prerequisites`},{id:`installation`,label:`Installation`},{id:`starting-daemon`,label:`Starting the Daemon`},{id:`first-task`,label:`Your First Task`},{id:`task-management`,label:`Task Management`},{id:`providers`,label:`Managing Providers`},{id:`dashboard`,label:`Using the Dashboard`},{id:`testing`,label:`Running Tests`},{id:`next-steps`,label:`Next Steps`}],J=[`Shows all tasks with real-time status updates via SSE`,`Allows submitting new tasks directly`,`Displays provider status and model information`,`Auto-refreshes every 2 seconds`],Y=[{icon:`📡`,title:`API Reference`,desc:`Full HTTP and MCP endpoint docs`,to:`/api-reference`},{icon:`🔌`,title:`MCP Integration`,desc:`Connect with Claude Desktop`,to:`/mcp-integration`},{icon:`🏗️`,title:`Architecture`,desc:`Understand the hexagonal design`,to:`/architecture`}];return(a,X)=>(r(),d(`div`,g,[u(`div`,_,[u(`aside`,v,[u(`div`,y,[X[0]||=u(`h4`,{class:`text-xs font-semibold uppercase tracking-wider text-slate-500 mb-4`},` On this page `,-1),u(`nav`,b,[(r(),d(l,null,e(q,e=>u(`a`,{key:e.id,href:`#${e.id}`,onClick:f(n=>t(h)(e.id),[`prevent`]),class:`block text-sm text-slate-500 hover:text-violet-400 transition-colors py-0.5 cursor-pointer`},i(e.label),9,x)),64))])])]),u(`main`,S,[X[16]||=c(`<div class="reveal"><div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-violet-500/30 bg-violet-500/5 text-sm text-violet-300 mb-4"><i class="pi pi-book text-xs"></i> Step-by-step guide </div><h1 class="text-4xl font-black mb-4"> Getting <span class="gradient-text">Started</span></h1><p class="text-lg text-slate-400"> Be up and running with nexus-orchestrator in under 5 minutes. </p></div><section id="prerequisites" class="reveal"><h2 class="text-2xl font-black mb-6 flex items-center gap-2"><span class="text-violet-400">#</span> Prerequisites </h2><div class="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6"><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4"><div class="text-lg mb-2">🔷</div><div class="font-bold text-white text-sm mb-1">Go 1.27+</div><div class="text-xs text-slate-500"> With CGO_ENABLED=1 and a C compiler (gcc/clang) </div></div><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4"><div class="text-lg mb-2">🗄️</div><div class="font-bold text-white text-sm mb-1">C Compiler</div><div class="text-xs text-slate-500">Required by go-sqlite3 (gcc or clang)</div></div><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4"><div class="text-lg mb-2">🤖</div><div class="font-bold text-white text-sm mb-1">LLM Provider</div><div class="text-xs text-slate-500">LM Studio, Ollama, or cloud API keys</div></div></div><div class="rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 text-sm text-amber-300"><strong>Provider options:</strong><ul class="mt-2 space-y-1 list-disc list-inside text-amber-400/80"><li><a href="https://lmstudio.ai/" target="_blank" rel="noopener" class="underline">LM Studio</a> running on 127.0.0.1:1234 </li><li><a href="https://ollama.ai/" target="_blank" rel="noopener" class="underline">Ollama</a> running on 127.0.0.1:11434 </li><li>Cloud API keys for OpenAI, Anthropic, or GitHub Copilot</li></ul></div></section>`,2),u(`section`,C,[X[1]||=u(`h2`,{class:`text-2xl font-black mb-6 flex items-center gap-2`},[u(`span`,{class:`text-violet-400`},`#`),s(` Installation `)],-1),o(m,{language:`bash`,code:L})]),u(`section`,w,[X[2]||=u(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[u(`span`,{class:`text-violet-400`},`#`),s(` Starting the Daemon `)],-1),o(m,{language:`bash`,code:R}),X[3]||=u(`h3`,{class:`text-lg font-bold mt-8 mb-3 text-slate-200`},`Custom Configuration`,-1),o(m,{language:`bash`,code:z}),X[4]||=u(`h3`,{class:`text-lg font-bold mt-8 mb-3 text-slate-200`},`Cloud Provider Configuration`,-1),o(m,{language:`bash`,code:B})]),u(`section`,T,[X[5]||=u(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[u(`span`,{class:`text-violet-400`},`#`),s(` Submitting Your First Task `)],-1),o(m,{language:`bash`,code:V}),X[6]||=u(`p`,{class:`text-slate-500 text-sm mt-3 mb-2`},`Response:`,-1),o(m,{language:`json`,code:H})]),u(`section`,E,[X[7]||=u(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[u(`span`,{class:`text-violet-400`},`#`),s(` Task Management `)],-1),X[8]||=u(`h3`,{class:`text-lg font-bold mb-3 text-slate-200`},`Checking Task Status`,-1),o(m,{language:`bash`,code:U}),X[9]||=u(`h3`,{class:`text-lg font-bold mt-6 mb-3 text-slate-200`},`Cancelling a Task`,-1),o(m,{language:`bash`,code:W})]),u(`section`,D,[X[10]||=u(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[u(`span`,{class:`text-violet-400`},`#`),s(` Managing Providers at Runtime `)],-1),o(m,{language:`bash`,code:G})]),u(`section`,O,[X[13]||=u(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[u(`span`,{class:`text-violet-400`},`#`),s(` Using the Dashboard `)],-1),u(`div`,k,[X[12]||=u(`p`,{class:`text-slate-400 text-sm mb-3`},[s(` Open `),u(`code`,{class:`text-violet-300`},`http://localhost:63987/ui`),s(` in your browser for a live dashboard that: `)],-1),u(`ul`,A,[(r(),d(l,null,e(J,e=>u(`li`,{key:e,class:`flex items-start gap-2 text-sm text-slate-400`},[X[11]||=u(`i`,{class:`pi pi-check-circle text-emerald-500 text-xs mt-0.5 flex-shrink-0`},null,-1),s(` `+i(e),1)])),64))])])]),u(`section`,j,[X[14]||=u(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[u(`span`,{class:`text-violet-400`},`#`),s(` Running Tests `)],-1),o(m,{language:`bash`,code:K})]),u(`section`,M,[X[15]||=u(`h2`,{class:`text-2xl font-black mb-6 flex items-center gap-2`},[u(`span`,{class:`text-violet-400`},`#`),s(` Next Steps `)],-1),u(`div`,N,[(r(),d(l,null,e(Y,e=>o(t(p),{key:e.to,to:e.to,class:`rounded-xl border border-white/8 bg-[#0d0d14] hover:border-violet-500/30 p-5 transition-all group`},{default:n(()=>[u(`div`,P,i(e.icon),1),u(`div`,F,i(e.title),1),u(`div`,I,i(e.desc),1)]),_:2},1032,[`to`])),64))])])])])]))}});export{q as default};