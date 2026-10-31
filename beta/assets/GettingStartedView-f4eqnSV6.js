import{C as e,D as t,S as n,T as r,_ as i,f as a,g as o,h as s,l as c,o as l,r as u,s as ee,u as d,v as f,x as p,y as m}from"./app-RJ9i1ivr.js";import{t as h}from"./scroll-DdeK3J_A.js";var g={class:`max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16`},_={class:`grid grid-cols-1 lg:grid-cols-4 gap-12`},v={class:`lg:col-span-1`},y={class:`sticky top-24 space-y-1`},b={class:`space-y-1`},x=[`href`,`onClick`],S={class:`lg:col-span-3 space-y-16`},C={class:`reveal`},w={class:`inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-violet-500/30 bg-violet-500/5 text-sm text-violet-300 mb-4`},T={id:`installation`,class:`reveal`},E={id:`starting-daemon`,class:`reveal`},D={id:`connect-agents`,class:`reveal`},O={id:`first-task`,class:`reveal`},k={id:`task-management`,class:`reveal`},A={id:`providers`,class:`reveal`},j={id:`dashboard`,class:`reveal`},M={class:`rounded-xl border border-white/8 bg-[#0d0d14] p-6`},N={class:`space-y-2 list-none p-0`},te={id:`testing`,class:`reveal`},P={id:`next-steps`,class:`reveal`},F={class:`grid grid-cols-1 sm:grid-cols-3 gap-4`},I={class:`text-xl mb-2`},L={class:`font-bold text-white text-sm mb-1 group-hover:text-violet-300 transition-colors`},R={class:`text-xs text-slate-500`},z=`# macOS desktop app (no Gatekeeper dialog), or the CLI + daemon on any system
curl -sSfL https://raw.githubusercontent.com/el-j/nexus-orchestrator/main/scripts/install.sh | sh -s -- --desktop
curl -sSfL https://raw.githubusercontent.com/el-j/nexus-orchestrator/main/scripts/install.sh | sh

# Or from source: one command builds and starts everything
git clone https://github.com/el-j/nexus-orchestrator.git
cd nexus-orchestrator
nvm use          # Node >= 22 builds the full UI
make start`,B=`# Which agents on this machine can call Nexus? (changes nothing)
nexus-cli connect

# Connect them: Antigravity, Cursor, Claude Code
nexus-cli connect --write`,V=`# Start with default settings
./nexus-daemon
# HTTP API:   http://127.0.0.1:63987
# MCP server: http://127.0.0.1:63988/mcp
# Full UI:    http://127.0.0.1:63987/app/   (light dashboard: /ui)`,H=`# Use environment variables for custom settings (default database: ~/.nexus/nexus.db)
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
go vet ./...`,X=m({__name:`GettingStartedView`,setup(m){let X=[{id:`prerequisites`,label:`Prerequisites`},{id:`installation`,label:`Installation`},{id:`starting-daemon`,label:`Starting the Daemon`},{id:`connect-agents`,label:`Connect Your Agents`},{id:`first-task`,label:`Your First Task`},{id:`task-management`,label:`Task Management`},{id:`providers`,label:`Managing Providers`},{id:`dashboard`,label:`Using the Dashboard`},{id:`testing`,label:`Running Tests`},{id:`next-steps`,label:`Next Steps`}],Z=[`Mission Control: active work, recent completions, agents and providers, task submission`,`Live output of a running task while the model writes it`,`Projects: activity, plans, history, backlog and the project brain as tabs`,`Each task says who ran it: Nexus on a provider, or an agent that only reported a status`],Q=[{icon:`📡`,title:`API Reference`,desc:`Full HTTP and MCP endpoint docs`,to:`/api-reference`},{icon:`🔌`,title:`MCP Integration`,desc:`Connect Claude Code, Cursor, Antigravity and Claude Desktop`,to:`/mcp-integration`},{icon:`🏗️`,title:`Architecture`,desc:`Understand the hexagonal design`,to:`/architecture`}];return(m,$)=>(p(),s(`div`,g,[a(`div`,_,[a(`aside`,v,[a(`div`,y,[$[0]||=a(`h4`,{class:`text-xs font-semibold uppercase tracking-wider text-slate-500 mb-4`},` On this page `,-1),a(`nav`,b,[(p(),s(d,null,n(X,e=>a(`a`,{key:e.id,href:`#${e.id}`,onClick:c(t=>r(h)(e.id),[`prevent`]),class:`block text-sm text-slate-500 hover:text-violet-400 transition-colors py-0.5 cursor-pointer`},t(e.label),9,x)),64))])])]),a(`main`,S,[a(`div`,C,[a(`div`,w,[f(l,{name:`book`,class:`text-xs`}),$[1]||=i(` Step-by-step guide `,-1)]),$[2]||=a(`h1`,{class:`text-4xl font-black mb-4`},[i(` Getting `),a(`span`,{class:`gradient-text`},`Started`)],-1),$[3]||=a(`p`,{class:`text-lg text-slate-400`},` Be up and running with nexus-orchestrator in under 5 minutes. `,-1)]),$[21]||=o(`<section id="prerequisites" class="reveal"><h2 class="text-2xl font-black mb-6 flex items-center gap-2"><span class="text-violet-400">#</span> Prerequisites </h2><div class="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6"><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4"><div class="text-lg mb-2">🔷</div><div class="font-bold text-white text-sm mb-1">Go 1.27+</div><div class="text-xs text-slate-500"> With CGO_ENABLED=1 and a C compiler (gcc/clang) </div></div><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4"><div class="text-lg mb-2">🗄️</div><div class="font-bold text-white text-sm mb-1">C Compiler</div><div class="text-xs text-slate-500">Required by go-sqlite3 (gcc or clang)</div></div><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4"><div class="text-lg mb-2">🤖</div><div class="font-bold text-white text-sm mb-1">LLM Provider</div><div class="text-xs text-slate-500">LM Studio, Ollama, or cloud API keys</div></div></div><div class="rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 text-sm text-amber-300"><strong>Provider options:</strong><ul class="mt-2 space-y-1 list-disc list-inside text-amber-400/80"><li><a href="https://lmstudio.ai/" target="_blank" rel="noopener" class="underline">LM Studio</a> running on 127.0.0.1:1234 </li><li><a href="https://ollama.ai/" target="_blank" rel="noopener" class="underline">Ollama</a> running on 127.0.0.1:11434 </li><li>Cloud API keys for OpenAI, Anthropic, or GitHub Copilot</li></ul></div></section>`,1),a(`section`,T,[$[4]||=a(`h2`,{class:`text-2xl font-black mb-6 flex items-center gap-2`},[a(`span`,{class:`text-violet-400`},`#`),i(` Installation `)],-1),f(u,{language:`bash`,code:z})]),a(`section`,E,[$[5]||=a(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[a(`span`,{class:`text-violet-400`},`#`),i(` Starting the Daemon `)],-1),f(u,{language:`bash`,code:V}),$[6]||=a(`h3`,{class:`text-lg font-bold mt-8 mb-3 text-slate-200`},`Custom Configuration`,-1),f(u,{language:`bash`,code:H}),$[7]||=a(`h3`,{class:`text-lg font-bold mt-8 mb-3 text-slate-200`},`Cloud Provider Configuration`,-1),f(u,{language:`bash`,code:U})]),a(`section`,D,[$[8]||=a(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[a(`span`,{class:`text-violet-400`},`#`),i(` Connect Your Agents `)],-1),$[9]||=a(`p`,{class:`text-slate-400 text-sm mb-4`},` An agent that cannot call Nexus tools can only work around them, for example by editing the database by hand, and then Nexus shows state it never ran. Connect the agents on your machine once: `,-1),f(u,{language:`bash`,code:B}),$[10]||=a(`p`,{class:`text-slate-500 text-sm mt-3`},[i(` With the VS Code extension installed, the editor's own agent is connected for you (VS Code 1.101+ through its MCP support, Antigravity after one prompt). Then `),a(`code`,{class:`text-violet-300`},`make init-project PROJECT=/path/to/repo`),i(` teaches Nexus the project, and every session starts with its briefing. `)],-1)]),a(`section`,O,[$[11]||=a(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[a(`span`,{class:`text-violet-400`},`#`),i(` Submitting Your First Task `)],-1),f(u,{language:`bash`,code:W}),$[12]||=a(`p`,{class:`text-slate-500 text-sm mt-3 mb-2`},`Response:`,-1),f(u,{language:`json`,code:G})]),a(`section`,k,[$[13]||=a(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[a(`span`,{class:`text-violet-400`},`#`),i(` Task Management `)],-1),$[14]||=a(`h3`,{class:`text-lg font-bold mb-3 text-slate-200`},`Checking Task Status`,-1),f(u,{language:`bash`,code:K}),$[15]||=a(`h3`,{class:`text-lg font-bold mt-6 mb-3 text-slate-200`},`Cancelling a Task`,-1),f(u,{language:`bash`,code:q})]),a(`section`,A,[$[16]||=a(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[a(`span`,{class:`text-violet-400`},`#`),i(` Managing Providers at Runtime `)],-1),f(u,{language:`bash`,code:J})]),a(`section`,j,[$[18]||=a(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[a(`span`,{class:`text-violet-400`},`#`),i(` Using the UI `)],-1),a(`div`,M,[$[17]||=a(`p`,{class:`text-slate-400 text-sm mb-3`},[i(` Open `),a(`code`,{class:`text-violet-300`},`http://localhost:63987/app/`),i(` in your browser (the daemon embeds the full UI when built with Node 22; otherwise `),a(`code`,{class:`text-violet-300`},`/ui`),i(` is a light dashboard): `)],-1),a(`ul`,N,[(p(),s(d,null,n(Z,e=>a(`li`,{key:e,class:`flex items-start gap-2 text-sm text-slate-400`},[f(l,{name:`check-circle`,class:`text-emerald-500 text-xs mt-0.5 flex-shrink-0`}),i(` `+t(e),1)])),64))])])]),a(`section`,te,[$[19]||=a(`h2`,{class:`text-2xl font-black mb-4 flex items-center gap-2`},[a(`span`,{class:`text-violet-400`},`#`),i(` Running Tests `)],-1),f(u,{language:`bash`,code:Y})]),a(`section`,P,[$[20]||=a(`h2`,{class:`text-2xl font-black mb-6 flex items-center gap-2`},[a(`span`,{class:`text-violet-400`},`#`),i(` Next Steps `)],-1),a(`div`,F,[(p(),s(d,null,n(Q,n=>f(r(ee),{key:n.to,to:n.to,class:`rounded-xl border border-white/8 bg-[#0d0d14] hover:border-violet-500/30 p-5 transition-all group`},{default:e(()=>[a(`div`,I,t(n.icon),1),a(`div`,L,t(n.title),1),a(`div`,R,t(n.desc),1)]),_:2},1032,[`to`])),64))])])])])]))}});export{X as default};