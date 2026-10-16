/**
 * architecture.ts: the tables and code samples of the Architecture page.
 */

/** The fields of a task, as the architecture page lists them. */
export const taskFields = [
  { field: 'ID', type: 'string', desc: 'UUID, generated on submission' },
  { field: 'ProjectPath', type: 'string', desc: 'Absolute path to the project' },
  { field: 'TargetFile', type: 'string', desc: 'Relative path for generated code output' },
  { field: 'Instruction', type: 'string', desc: 'Natural language prompt' },
  { field: 'ContextFiles', type: '[]string', desc: 'Files to include as context' },
  { field: 'ModelID', type: 'string', desc: 'Constrain to a specific model (optional)' },
  { field: 'ProviderHint', type: 'string', desc: 'Prefer a specific provider (optional)' },
  { field: 'Command', type: 'CommandType', desc: 'Task classification: plan, execute, or auto' },
  { field: 'Status', type: 'TaskStatus', desc: 'Lifecycle state' },
  { field: 'CreatedAt', type: 'time.Time', desc: 'Creation timestamp' },
  { field: 'UpdatedAt', type: 'time.Time', desc: 'Last update timestamp' },
  {
    field: 'Logs',
    type: 'string',
    desc: 'LLM output or error details; the engine writes "completed via <provider>"',
  },
  {
    field: 'RanBy',
    type: 'string',
    desc: 'Derived from Logs: the provider Nexus ran the task on, empty when an agent reported the status',
  },
  {
    field: 'Role / Priority / Tags',
    type: 'string / int / []string',
    desc: 'Routing tier hint, backlog ordering and labels',
  },
  {
    field: 'Agentic',
    type: 'bool',
    desc: 'Run as a bounded multi-turn tool loop over the project files',
  },
];

/** The task statuses with their meaning and badge colour. */
export const statusRows = [
  { status: 'QUEUED', desc: 'Task is waiting in the queue', color: 'bg-blue-500/20 text-blue-300' },
  {
    status: 'PROCESSING',
    desc: 'Task is being processed by an LLM',
    color: 'bg-amber-500/20 text-amber-300',
  },
  {
    status: 'COMPLETED',
    desc: 'LLM generated output successfully',
    color: 'bg-emerald-500/20 text-emerald-300',
  },
  { status: 'FAILED', desc: 'LLM call failed', color: 'bg-red-500/20 text-red-300' },
  {
    status: 'DRAFT / BACKLOG',
    desc: 'Staged ideas that do not run until promoted',
    color: 'bg-sky-500/20 text-sky-300',
  },
  {
    status: 'WAITING',
    desc: 'Every provider is cooling down after a limit; queued again at the reset time',
    color: 'bg-cyan-500/20 text-cyan-300',
  },
  {
    status: 'CANCELLED',
    desc: 'Cancelled before processing',
    color: 'bg-slate-500/20 text-slate-300',
  },
  {
    status: 'TOO_LARGE',
    desc: 'Prompt exceeds model context window',
    color: 'bg-orange-500/20 text-orange-300',
  },
  {
    status: 'NO_PROVIDER',
    desc: 'No provider available for requested model',
    color: 'bg-purple-500/20 text-purple-300',
  },
];

/** The task command types. */
export const commandRows = [
  { cmd: 'plan', desc: 'Planning/orchestration work (creating plans, task documents)' },
  { cmd: 'execute', desc: 'Code implementation (requires a prior completed plan)' },
  { cmd: 'auto', desc: 'Let the orchestrator decide (default)' },
];

/** The core ports with their Go signatures. */
export const ports = [
  {
    name: 'Orchestrator',
    direction: 'Inbound',
    desc: 'The primary interface that UI, CLI, and HTTP API call',
    code: `type Orchestrator interface {
    SubmitTask(task domain.Task) (string, error)
    GetTask(id string) (domain.Task, error)
    GetQueue() ([]domain.Task, error)
    GetProviders() ([]ProviderInfo, error)
    CancelTask(id string) error
    RegisterCloudProvider(cfg domain.ProviderConfig) error
    RemoveProvider(providerName string) error
    GetProviderModels(providerName string) ([]string, error)
}`,
  },
  {
    name: 'LLMClient',
    direction: 'Outbound',
    desc: 'Interface for any language model backend; every call takes a context so a cancel aborts it',
    code: `type LLMClient interface {
    Ping(ctx context.Context) bool
    ProviderName() string
    ActiveModel() string
    BaseURL() string
    GetAvailableModels(ctx context.Context) ([]string, error)
    ContextLimit(ctx context.Context) int
    GenerateCode(ctx context.Context, prompt string) (string, error)
    Chat(ctx context.Context, messages []domain.Message) (string, error)
}`,
  },
  {
    name: 'ChatStreamer',
    direction: 'Outbound (optional)',
    desc: 'A provider that can deliver its reply piece by piece; the engine publishes the pieces as task.chunk events, others send the whole reply as one chunk',
    code: `type ChatStreamer interface {
    ChatStream(ctx context.Context, messages []domain.Message,
        onChunk func(piece string)) (string, error)
}`,
  },
  {
    name: 'TaskRepository',
    direction: 'Outbound',
    desc: 'Persistence for tasks',
    code: `type TaskRepository interface {
    Save(ctx context.Context, t domain.Task) error
    GetByID(ctx context.Context, id string) (domain.Task, error)
    GetPending(ctx context.Context) ([]domain.Task, error)
    ClaimNextQueued(ctx context.Context) (domain.Task, error)
    GetAll(ctx context.Context) ([]domain.Task, error)
    UpdateStatus(ctx context.Context, id string, status domain.TaskStatus) error
    UpdateLogs(ctx context.Context, id, logs string) error
    // ... by project, by status, stale PROCESSING tasks for the watchdog
}`,
  },
  {
    name: 'SessionRepository',
    direction: 'Outbound',
    desc: 'Per-project conversation history',
    code: `type SessionRepository interface {
    Save(s domain.Session) error
    GetByProjectPath(projectPath string) (domain.Session, error)
    AppendMessage(projectPath string, msg domain.Message) error
}`,
  },
];

/** The inbound adapters: what drives the core. */
export const inboundAdapters = [
  {
    adapter: 'HTTP API',
    pkg: 'httpapi',
    proto: 'REST + SSE (task events, live chunks)',
    port: ':63987',
  },
  {
    adapter: 'Web UI',
    pkg: 'webui',
    proto: 'The Vue UI embedded in the daemon, at /app/',
    port: ':63987',
  },
  {
    adapter: 'MCP Server',
    pkg: 'mcp',
    proto: 'JSON-RPC 2.0 (43 tools, wait_for_task streams progress)',
    port: ':63988',
  },
  {
    adapter: 'CLI Client',
    pkg: 'cli',
    proto: 'HTTP → daemon; nexus connect edits agent configs',
    port: '—',
  },
  {
    adapter: 'Origin / Host guard',
    pkg: 'httpguard',
    proto: 'Rejects foreign browser origins and DNS rebinding',
    port: '—',
  },
  {
    adapter: 'Wails GUI',
    pkg: 'app.go',
    proto: 'Native window over the same HTTP API',
    port: ':63987',
  },
];

/** The outbound adapters: what the core drives. */
export const outboundAdapters = [
  {
    adapter: 'LM Studio',
    pkg: 'llm_lmstudio',
    backend: 'OpenAI-compatible API at :1234 (streams)',
  },
  { adapter: 'Ollama', pkg: 'llm_ollama', backend: 'Ollama API at :11434 (streams NDJSON)' },
  {
    adapter: 'OpenAI-compatible',
    pkg: 'llm_openaicompat',
    backend: 'OpenAI, GitHub Copilot, the VS Code model bridge, vLLM, LocalAI (streams SSE)',
  },
  { adapter: 'Anthropic', pkg: 'llm_anthropic', backend: 'Anthropic Messages API (streams)' },
  {
    adapter: 'Google Gemini',
    pkg: 'llm_gemini',
    backend: 'generateContent / streamGenerateContent',
  },
  {
    adapter: 'CLI subscription',
    pkg: 'llm_cli',
    backend: 'A logged-in AI CLI such as claude -p (Claude Pro/Max)',
  },
  {
    adapter: 'SSE reader',
    pkg: 'sse',
    backend: 'Shared server-sent-events parser of the streaming adapters',
  },
  { adapter: 'SQLite', pkg: 'repo_sqlite', backend: 'Local SQLite (FTS5 brain) via go-sqlite3' },
  {
    adapter: 'Filesystem',
    pkg: 'fs_writer, fs_watcher',
    backend: 'Disk write-back and project change watching',
  },
  {
    adapter: 'Activity readers',
    pkg: 'activity_*',
    backend: 'Claude Code transcripts, Continue, network activity',
  },
  { adapter: 'Scanners', pkg: 'sys_scanner', backend: 'Provider, agent and plan-file discovery' },
];

/** How the core handles concurrency (HTML snippets). */
export const concurrencyPoints = [
  'Shared state is protected with <code class="text-slate-300">sync.Mutex</code> in OrchestratorService',
  'The background worker goroutine processes tasks sequentially — only one LLM call is ever in flight',
  'Shutdown is coordinated via a <code class="text-slate-300">stopCh chan struct{}</code> channel',
  '<code class="text-slate-300">Stop()</code> is idempotent via <code class="text-slate-300">sync.Once</code>',
  'No goroutines inside core services — goroutine lifecycle is an infrastructure concern owned by inbound adapters',
  'SSE broadcasting uses a separate Hub with its own mutex to avoid lock nesting; a slow subscriber drops events instead of blocking the engine',
  'Every attempt publishes task.restart first, then task.chunk events (at most one per 50 ms), so a client needs one rule: restart clears, chunk appends',
];
