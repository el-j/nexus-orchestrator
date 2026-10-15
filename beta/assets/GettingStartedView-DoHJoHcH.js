import{C as e,_ as t,a as n,d as r,f as i,h as a,i as o,l as s,m as c,o as l,p as u,s as d,t as f,v as p,x as m,y as h}from"./app-BsebaH17.js";import{t as g}from"./scroll-DdeK3J_A.js";var _={class:`max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16`},v={class:`grid grid-cols-1 lg:grid-cols-4 gap-12`},y={class:`lg:col-span-1`},b={class:`sticky top-24 space-y-1`},x={class:`space-y-1`},S=[`href`,`onClick`],C={class:`lg:col-span-3 space-y-16`},w={class:`reveal`},T={class:`inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-violet-500/30 bg-violet-500/5 text-sm text-violet-300 mb-4`},E={id:`installation`,class:`reveal`},D={id:`starting-daemon`,class:`reveal`},O={id:`first-task`,class:`reveal`},k={id:`task-management`,class:`reveal`},A={id:`providers`,class:`reveal`},j={id:`dashboard`,class:`reveal`},M={class:`rounded-xl border border-white/8 bg-[#0d0d14] p-6`},N={class:`space-y-2 list-none p-0`},P={id:`testing`,class:`reveal`},F={id:`next-steps`,class:`reveal`},I={class:`grid grid-cols-1 sm:grid-cols-3 gap-4`},L={class:`text-xl mb-2`},R={class:`font-bold text-white text-sm mb-1 group-hover:text-violet-300 transition-colors`},z={class:`text-xs text-slate-500`},B=`# Clone the repository
git clone https://github.com/el-j/nexus-orchestrator.git
cd nexus-orchestrator

# Build all binaries
CGO_ENABLED=1 go build ./...

# Or build specific binaries
CGO_ENABLED=1 go build -o nexus-daemon ./cmd/nexus-daemon/...
CGO_ENABLED=1 go build -o nexus-cli ./cmd/nexus-cli/...`,V=`# Start with default settings
./nexus-daemon
# HTTP API:   http://127.0.0.1:63987
# MCP server: http://127.0.0.1:63988/mcp
# Dashboard:  http://127.0.0.1:63987/ui`,H=`# Use environment variables for custom settings
NEXUS_DB_PATH=/path/to/nexus.db \\
NEXUS_LISTEN_ADDR=:8080 \\
NEXUS_MCP_ADDR=:8081 \\
./nexus-daemon`,U=`# OpenAI
NEXUS_OPENAI_API_KEY=sk-... NEXUS_OPENAI_MODEL=gpt-4o-mini ./nexus-daemon

# Anthropic
NEXUS_ANTHROPIC_API_KEY=sk-ant-... NEXUS_ANTHROPIC_MODEL=claude-3-5-sonnet-20241022 ./nexus-daemon

# GitHub Copilot
NEXUS_GITHUBCOPILOT_TOKEN=ghu_... NEXUS_GITHUBCOPILOT_MODEL=gpt-4o ./nexus-daemon`,W=`# Submit a code-generation task
curl -s -X POST http://localhost:63987/api/tasks \\
  -H "Content-Type: application/json" \\
  -d '{
    "projectPath": "'$PWD'",
    "targetFile": "hello.go",
    "instruction": "Write a Go function that returns Hello World"
  }' | jq .`,G=`{
  "task_id": "a1b2c3d4-...",
  "status": "QUEUED"
}`,K=`# Get task by ID
curl -s http://localhost:63987/api/tasks/TASK_ID | jq .

# List all pending tasks
curl -s http://localhost:63987/api/tasks | jq .`,q=`curl -X DELETE http://localhost:63987/api/tasks/TASK_ID
# Returns 204 No Content on success`,J=`# List all providers
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
curl -X DELETE http://localhost:63987/api/providers/My%20OpenAI`,Y=`# Full test suite with race detection
CGO_ENABLED=1 go test -race ./...

# Service tests only
CGO_ENABLED=1 go test ./internal/core/services/...

# Lint
go vet ./...`,X=a({__name:`GettingStartedView`,setup(a){let X=[{id:`prerequisites`,label:`Prerequisites`},{id:`installation`,label:`Installation`},{id:`starting-daemon`,label:`Starting the Daemon`},{id:`first-task`,label:`Your First Task`},{id:`task-management`,label:`Task Management`},{id:`providers`,label:`Managing Providers`},{id:`dashboard`,label:`Using the Dashboard`},{id:`testing`,label:`Running Tests`},{id:`next-steps`,label:`Next Steps`}],Z=[`Shows all tasks with real-time status updates via SSE`,`Allows submitting new tasks directly`,`Displays provider status and model information`,`Auto-refreshes every 2 seconds`],Q=[{icon:`📡`,title:`API Reference`,desc:`Full HTTP and MCP endpoint docs`,to:`/api-reference`},{icon:`🔌`,title:`MCP Integration`,desc:`Connect with Claude Desktop`,to:`/mcp-integration`},{icon:`🏗️`,title:`Architecture`,desc:`Understand the hexagonal design`,to:`/architecture`}];return(a,$)=>(t(),r(`div`,_,[s(`div`,v,[s(`aside`,y,[s(`div`,b,[$[0]||=s(`h4`,{class:`text-xs font-semibold uppercase tracking-wider text-slate-500 mb-4`},` On this page `,-1),s(`nav`,x,[(t(),r(d,null,p(X,t=>s(`a`,{key:t.id,href:`#${t.id}`,onClick:l(e=>m(g)(t.id),[`prevent`]),class:`block text-sm text-slate-500 hover:text-violet-400 transition-colors py-0.5 cursor-pointer`},e(t.label),9,S)),64))])])]),s(`main`,C,[s(`div`,w,[s(`div`,T,[c(o,{name:`book`,class:`text-xs`}),$[1]||=u(` Step-by-step guide `,-1)]),$[2]||=s(`h1`,{class:`text-4xl font-black mb-4`},[u(` Getting `),s(`span`,{class:`gradient-text`},`Started`)],-1),$[3]||=s(`p`,{class:`text-lg text-slate-400`},` Be up and running with nexus-orchestrator in under 5 minutes. `,-1)]),$[18]||=i(`<section id="prerequisites" class="reveal"><h2 class="text-2xl font-black mb-6 flex items-center gap-2"><span class="text-violet-400">#</span> Prerequisites </h2><div class="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6"><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4"><div class="text-lg mb-2">🔷</div><div class="font-bold text-white text-sm mb-1">Go 1.27+</div><div class="text-xs text-slate-500"> With CGO_ENABLED=1 and a C compiler (gcc/clang) </div></div><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4"><div class="text-lg mb-2">🗄️</div><div class="font-bold text-white text-sm mb-1">C Compiler</div><div class="text-xs text-slate-500">Required by go-sqlite3 (gcc or clang)</div></div><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4"><div class="text-lg mb-2">🤖</div><div class="font-bold text-white text-sm mb-1">LLM Provider</div><div class="text-xs text-slate-500">LM Studio, Ollama, or cloud API keys</div></div></div><div class="rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 text-sm text-amber-300"><strong>Provider options:</strong><ul class="mt-2 space-y-1 list-disc list-inside text-amber-400/80"><li><a href="https://lmstudio.ai/" target="_blank" rel="noopener" class="underline">LM Studio</a> running on 127.0.0.1:1234 </li><li><a href="https://ollama.ai/" target="_blank" rel="noopener" class="underline">Ollama</a> running on 127.0.0.1:11434 </li><li>Cloud API keys for OpenAI, Anthropic, or GitHub Copilot</li></ul></div></section>`,1),s(`section`,E,[$[4]||=s(`h2`,{class:`text-2xl font-black mb-6 flex items-center gap-2`},[s(`span`,{class:`text-violet-400`},`#`),u(` Installation `)],-1),c(f,{language:`bash`,code:B})]),s(`section`,D,[$[5]||=s(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[s(`span`,{class:`text-violet-400`},`#`),u(` Starting the Daemon `)],-1),c(f,{language:`bash`,code:V}),$[6]||=s(`h3`,{class:`text-lg font-bold mt-8 mb-3 text-slate-200`},`Custom Configuration`,-1),c(f,{language:`bash`,code:H}),$[7]||=s(`h3`,{class:`text-lg font-bold mt-8 mb-3 text-slate-200`},`Cloud Provider Configuration`,-1),c(f,{language:`bash`,code:U})]),s(`section`,O,[$[8]||=s(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[s(`span`,{class:`text-violet-400`},`#`),u(` Submitting Your First Task `)],-1),c(f,{language:`bash`,code:W}),$[9]||=s(`p`,{class:`text-slate-500 text-sm mt-3 mb-2`},`Response:`,-1),c(f,{language:`json`,code:G})]),s(`section`,k,[$[10]||=s(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[s(`span`,{class:`text-violet-400`},`#`),u(` Task Management `)],-1),$[11]||=s(`h3`,{class:`text-lg font-bold mb-3 text-slate-200`},`Checking Task Status`,-1),c(f,{language:`bash`,code:K}),$[12]||=s(`h3`,{class:`text-lg font-bold mt-6 mb-3 text-slate-200`},`Cancelling a Task`,-1),c(f,{language:`bash`,code:q})]),s(`section`,A,[$[13]||=s(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[s(`span`,{class:`text-violet-400`},`#`),u(` Managing Providers at Runtime `)],-1),c(f,{language:`bash`,code:J})]),s(`section`,j,[$[15]||=s(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[s(`span`,{class:`text-violet-400`},`#`),u(` Using the Dashboard `)],-1),s(`div`,M,[$[14]||=s(`p`,{class:`text-slate-400 text-sm mb-3`},[u(` Open `),s(`code`,{class:`text-violet-300`},`http://localhost:63987/ui`),u(` in your browser for a live dashboard that: `)],-1),s(`ul`,N,[(t(),r(d,null,p(Z,t=>s(`li`,{key:t,class:`flex items-start gap-2 text-sm text-slate-400`},[c(o,{name:`check-circle`,class:`text-emerald-500 text-xs mt-0.5 flex-shrink-0`}),u(` `+e(t),1)])),64))])])]),s(`section`,P,[$[16]||=s(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[s(`span`,{class:`text-violet-400`},`#`),u(` Running Tests `)],-1),c(f,{language:`bash`,code:Y})]),s(`section`,F,[$[17]||=s(`h2`,{class:`text-2xl font-black mb-6 flex items-center gap-2`},[s(`span`,{class:`text-violet-400`},`#`),u(` Next Steps `)],-1),s(`div`,I,[(t(),r(d,null,p(Q,t=>c(m(n),{key:t.to,to:t.to,class:`rounded-xl border border-white/8 bg-[#0d0d14] hover:border-violet-500/30 p-5 transition-all group`},{default:h(()=>[s(`div`,L,e(t.icon),1),s(`div`,R,e(t.title),1),s(`div`,z,e(t.desc),1)]),_:2},1032,[`to`])),64))])])])])]))}});export{X as default};