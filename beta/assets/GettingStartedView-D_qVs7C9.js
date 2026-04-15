import{d as b,c as n,a as t,F as d,r as c,b as m,e as o,g as a,_ as i,k as h,t as l,w as v,u as p,R as f,j as r}from"./app-CGlCt2Sl.js";import{s as k}from"./scroll-DpH_xQsu.js";const _={class:"max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16"},E={class:"grid grid-cols-1 lg:grid-cols-4 gap-12"},C={class:"lg:col-span-1"},S={class:"sticky top-24 space-y-1"},T={class:"space-y-1"},D=["href","onClick"],P={class:"lg:col-span-3 space-y-16"},A={id:"installation",class:"reveal"},y={id:"starting-daemon",class:"reveal"},O={id:"first-task",class:"reveal"},N={id:"task-management",class:"reveal"},I={id:"providers",class:"reveal"},w={id:"dashboard",class:"reveal"},L={class:"rounded-xl border border-white/8 bg-[#0d0d14] p-6"},U={class:"space-y-2 list-none p-0"},M={id:"testing",class:"reveal"},R={id:"next-steps",class:"reveal"},G={class:"grid grid-cols-1 sm:grid-cols-3 gap-4"},B={class:"text-xl mb-2"},j={class:"font-bold text-white text-sm mb-1 group-hover:text-violet-300 transition-colors"},X={class:"text-xs text-slate-500"},H=`# Clone the repository
git clone https://github.com/el-j/nexus-orchestrator.git
cd nexus-orchestrator

# Build all binaries
CGO_ENABLED=1 go build ./...

# Or build specific binaries
CGO_ENABLED=1 go build -o nexus-daemon ./cmd/nexus-daemon/...
CGO_ENABLED=1 go build -o nexus-cli ./cmd/nexus-cli/...`,q=`# Start with default settings
./nexus-daemon
# HTTP API:   http://127.0.0.1:63987
# MCP server: http://127.0.0.1:63988/mcp
# Dashboard:  http://127.0.0.1:63987/ui`,F=`# Use environment variables for custom settings
NEXUS_DB_PATH=/path/to/nexus.db \\
NEXUS_LISTEN_ADDR=:8080 \\
NEXUS_MCP_ADDR=:8081 \\
./nexus-daemon`,K=`# OpenAI
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
  }' | jq .`,W=`{
  "id": "a1b2c3d4-...",
  "projectPath": "/path/to/project",
  "targetFile": "hello.go",
  "instruction": "Write a Go function...",
  "status": "QUEUED",
  "createdAt": "2025-01-01T00:00:00Z"
}`,Y=`# Get task by ID
curl -s http://localhost:63987/api/tasks/TASK_ID | jq .

# List all pending tasks
curl -s http://localhost:63987/api/tasks | jq .`,$=`curl -X DELETE http://localhost:63987/api/tasks/TASK_ID
# Returns 204 No Content on success`,Q=`# List all providers
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
curl -X DELETE http://localhost:63987/api/providers/My%20OpenAI`,Z=`# Full test suite with race detection
CGO_ENABLED=1 go test -race ./...

# Service tests only
CGO_ENABLED=1 go test ./internal/core/services/...

# Lint
go vet ./...`,ot=b({__name:"GettingStartedView",setup(z){const g=[{id:"prerequisites",label:"Prerequisites"},{id:"installation",label:"Installation"},{id:"starting-daemon",label:"Starting the Daemon"},{id:"first-task",label:"Your First Task"},{id:"task-management",label:"Task Management"},{id:"providers",label:"Managing Providers"},{id:"dashboard",label:"Using the Dashboard"},{id:"testing",label:"Running Tests"},{id:"next-steps",label:"Next Steps"}],x=["Shows all tasks with real-time status updates via SSE","Allows submitting new tasks directly","Displays provider status and model information","Auto-refreshes every 2 seconds"],u=[{icon:"📡",title:"API Reference",desc:"Full HTTP and MCP endpoint docs",to:"/api-reference"},{icon:"🔌",title:"MCP Integration",desc:"Connect with Claude Desktop",to:"/mcp-integration"},{icon:"🏗️",title:"Architecture",desc:"Understand the hexagonal design",to:"/architecture"}];return(J,e)=>(r(),n("div",_,[t("div",E,[t("aside",C,[t("div",S,[e[0]||(e[0]=t("h4",{class:"text-xs font-semibold uppercase tracking-wider text-slate-500 mb-4"}," On this page ",-1)),t("nav",T,[(r(),n(d,null,c(g,s=>t("a",{key:s.id,href:`#${s.id}`,onClick:h(tt=>p(k)(s.id),["prevent"]),class:"block text-sm text-slate-500 hover:text-violet-400 transition-colors py-0.5 cursor-pointer"},l(s.label),9,D)),64))])])]),t("main",P,[e[16]||(e[16]=m('<div class="reveal"><div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-violet-500/30 bg-violet-500/5 text-sm text-violet-300 mb-4"><i class="pi pi-book text-xs"></i> Step-by-step guide </div><h1 class="text-4xl font-black mb-4"> Getting <span class="gradient-text">Started</span></h1><p class="text-lg text-slate-400"> Be up and running with nexus-orchestrator in under 5 minutes. </p></div><section id="prerequisites" class="reveal"><h2 class="text-2xl font-black mb-6 flex items-center gap-2"><span class="text-violet-400">#</span> Prerequisites </h2><div class="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6"><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4"><div class="text-lg mb-2">🔷</div><div class="font-bold text-white text-sm mb-1">Go 1.26+</div><div class="text-xs text-slate-500"> With CGO_ENABLED=1 and a C compiler (gcc/clang) </div></div><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4"><div class="text-lg mb-2">🗄️</div><div class="font-bold text-white text-sm mb-1">C Compiler</div><div class="text-xs text-slate-500">Required by go-sqlite3 (gcc or clang)</div></div><div class="rounded-xl border border-white/8 bg-[#0d0d14] p-4"><div class="text-lg mb-2">🤖</div><div class="font-bold text-white text-sm mb-1">LLM Provider</div><div class="text-xs text-slate-500">LM Studio, Ollama, or cloud API keys</div></div></div><div class="rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 text-sm text-amber-300"><strong>Provider options:</strong><ul class="mt-2 space-y-1 list-disc list-inside text-amber-400/80"><li><a href="https://lmstudio.ai/" target="_blank" rel="noopener" class="underline">LM Studio</a> running on 127.0.0.1:1234 </li><li><a href="https://ollama.ai/" target="_blank" rel="noopener" class="underline">Ollama</a> running on 127.0.0.1:11434 </li><li>Cloud API keys for OpenAI, Anthropic, or GitHub Copilot</li></ul></div></section>',2)),t("section",A,[e[1]||(e[1]=t("h2",{class:"text-2xl font-black mb-6 flex items-center gap-2"},[t("span",{class:"text-violet-400"},"#"),o(" Installation ")],-1)),a(i,{language:"bash",code:H})]),t("section",y,[e[2]||(e[2]=t("h2",{class:"text-2xl font-black mb-4 flex items-center gap-2"},[t("span",{class:"text-violet-400"},"#"),o(" Starting the Daemon ")],-1)),a(i,{language:"bash",code:q}),e[3]||(e[3]=t("h3",{class:"text-lg font-bold mt-8 mb-3 text-slate-200"},"Custom Configuration",-1)),a(i,{language:"bash",code:F}),e[4]||(e[4]=t("h3",{class:"text-lg font-bold mt-8 mb-3 text-slate-200"},"Cloud Provider Configuration",-1)),a(i,{language:"bash",code:K})]),t("section",O,[e[5]||(e[5]=t("h2",{class:"text-2xl font-black mb-4 flex items-center gap-2"},[t("span",{class:"text-violet-400"},"#"),o(" Submitting Your First Task ")],-1)),a(i,{language:"bash",code:V}),e[6]||(e[6]=t("p",{class:"text-slate-500 text-sm mt-3 mb-2"},"Response:",-1)),a(i,{language:"json",code:W})]),t("section",N,[e[7]||(e[7]=t("h2",{class:"text-2xl font-black mb-4 flex items-center gap-2"},[t("span",{class:"text-violet-400"},"#"),o(" Task Management ")],-1)),e[8]||(e[8]=t("h3",{class:"text-lg font-bold mb-3 text-slate-200"},"Checking Task Status",-1)),a(i,{language:"bash",code:Y}),e[9]||(e[9]=t("h3",{class:"text-lg font-bold mt-6 mb-3 text-slate-200"},"Cancelling a Task",-1)),a(i,{language:"bash",code:$})]),t("section",I,[e[10]||(e[10]=t("h2",{class:"text-2xl font-black mb-4 flex items-center gap-2"},[t("span",{class:"text-violet-400"},"#"),o(" Managing Providers at Runtime ")],-1)),a(i,{language:"bash",code:Q})]),t("section",w,[e[13]||(e[13]=t("h2",{class:"text-2xl font-black mb-4 flex items-center gap-2"},[t("span",{class:"text-violet-400"},"#"),o(" Using the Dashboard ")],-1)),t("div",L,[e[12]||(e[12]=t("p",{class:"text-slate-400 text-sm mb-3"},[o(" Open "),t("code",{class:"text-violet-300"},"http://localhost:63987/ui"),o(" in your browser for a live dashboard that: ")],-1)),t("ul",U,[(r(),n(d,null,c(x,s=>t("li",{key:s,class:"flex items-start gap-2 text-sm text-slate-400"},[e[11]||(e[11]=t("i",{class:"pi pi-check-circle text-emerald-500 text-xs mt-0.5 flex-shrink-0"},null,-1)),o(" "+l(s),1)])),64))])])]),t("section",M,[e[14]||(e[14]=t("h2",{class:"text-2xl font-black mb-4 flex items-center gap-2"},[t("span",{class:"text-violet-400"},"#"),o(" Running Tests ")],-1)),a(i,{language:"bash",code:Z})]),t("section",R,[e[15]||(e[15]=t("h2",{class:"text-2xl font-black mb-6 flex items-center gap-2"},[t("span",{class:"text-violet-400"},"#"),o(" Next Steps ")],-1)),t("div",G,[(r(),n(d,null,c(u,s=>a(p(f),{key:s.to,to:s.to,class:"rounded-xl border border-white/8 bg-[#0d0d14] hover:border-violet-500/30 p-5 transition-all group"},{default:v(()=>[t("div",B,l(s.icon),1),t("div",j,l(s.title),1),t("div",X,l(s.desc),1)]),_:2},1032,["to"])),64))])])])])]))}});export{ot as default};
