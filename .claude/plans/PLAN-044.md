# PLAN-044 — Universal AI Takeover

**Status:** DESIGN  
**Author:** UXArchitect  
**Date:** 2026-03-14  
**Scope:** Go daemon + VS Code extension + frontend dashboard  

---

## Vision

Every AI agent running on the developer's machine becomes a **first-class citizen** in nexusOrchestrator's session registry. The VS Code extension acts as the universal bridge: it detects Claude Desktop, Claude CLI, Antigravity, Cline, Continue, CodeGPT, Cursor, and any other AI tool; reports each as an `AISession` to the daemon; and gives the operator a single "SEND TO NEXUS" command that delegates all pending work to the orchestrator queue.

The full loop: **detect → register → monitor → (optional) delegate**.

---

## Task Breakdown

| Task | Title | Layer | Depends On |
|------|-------|-------|------------|
| TASK-276 | Domain: `DiscoveredAgent` type + `AISession` field extensions | Go · domain | — |
| TASK-277 | Ports: `AgentScanner` interface + `DelegateToNexus` + `GetDiscoveredAgents` on `Orchestrator` | Go · ports | TASK-276 |
| TASK-278 | sys_scanner: Agent-specific probes (Claude cfg, VS Code ext dir, MCP port sweep, process flags) | Go · outbound | TASK-276 |
| TASK-279 | SQLite: additive migrations for new `ai_sessions` columns + `discovered_agents` table | Go · outbound | TASK-276 |
| TASK-280 | OrchestratorService: `DelegateToNexus()` + `GetDiscoveredAgents()` implementations | Go · services | TASK-277, TASK-279 |
| TASK-281 | HTTP API: `/api/ai-sessions/discovered`, `/api/ai-sessions/{id}/delegate`, SSE for session tasks | Go · inbound | TASK-280 |
| TASK-282 | VS Code: `AgentDetector` class — 30 s polling loop + per-agent probe strategies | TS · extension | — |
| TASK-283 | VS Code: `AISessionsTreeProvider` — live tree view of all registered sessions | TS · extension | TASK-285 |
| TASK-284 | VS Code: `nexus.delegateToNexus` command — three delegation paths | TS · extension | TASK-282, TASK-285 |
| TASK-285 | VS Code: `NexusClient` additions — delegate + discovered agents endpoints | TS · extension | TASK-281 |
| TASK-286 | Frontend: "AI Agents" dashboard section — list + colour-coded status + "Delegate All" | Vue · frontend | TASK-281 |
| TASK-287 | Frontend: Per-agent task timeline view | Vue · frontend | TASK-286 |
| TASK-288 | Extension wiring: integrate `AgentDetector` into `extension.ts`, update `package.json` | TS · extension | TASK-282, TASK-283, TASK-284 |
| TASK-289 | Tests: Go unit tests for new scanner probes + delegation service logic | Go · tests | TASK-278, TASK-280 |
| TASK-290 | Tests: TS unit tests for `AgentDetector` + delegation command | TS · tests | TASK-282, TASK-284 |

---

## Per-Task Implementation Specs

### TASK-276 — Domain: `DiscoveredAgent` + `AISession` extensions

Add a new `DiscoveredAgent` struct to `internal/core/domain/ai_session.go` (or a new file `discovered_agent.go`). This type represents an AI *agent* (a tool that processes instructions for the developer) as distinct from a `DiscoveredProvider` (an LLM *server* that processes tokens). Fields: `ID string`, `Name string`, `Kind AgentKind` (enum: `claude-cli`, `claude-desktop`, `antigravity`, `cline`, `continue`, `codegpt`, `cursor`, `copilot`, `aichat`, `generic`), `DetectionMethod string` (e.g., `"process"`, `"fs-config"`, `"vscode-extension"`, `"port-mcp"`), `ProcessName string`, `CLIPath string`, `ConfigPath string` (path to detected config file), `MCPEndpoint string` (if MCP-capable), `IsRunning bool`, `LastSeen time.Time`. Extend `AISession` with three new fields: `DelegatedToNexus bool` (JSON: `delegatedToNexus`), `DelegationTimestamp *time.Time` (JSON: `delegationTimestamp,omitempty`), `AgentCapabilities []string` (JSON: `agentCapabilities,omitempty`; values: `"file-write"`, `"code-execute"`, `"mcp-client"`, `"terminal"`, `"chat"`). Also add source constant `SessionSourceVSCodeDiscovered AISessionSource = "vscode-discovered"` for sessions registered by the new `AgentDetector`. Add `AgentKind` typed string and its constants to the same file. No framework imports anywhere in this package.

### TASK-277 — Ports: `AgentScanner` interface + Orchestrator extensions

Add a new `AgentScanner` outbound port interface to `internal/core/ports/ports.go`:
```go
type AgentScanner interface {
    ScanAgents(ctx context.Context) ([]domain.DiscoveredAgent, error)
}
```
`AgentScanner` is separate from `SystemScanner` to honour the single-responsibility principle and allow independent implementations or mocking. Add two new methods to the `Orchestrator` inbound port:
- `GetDiscoveredAgents(ctx context.Context) ([]domain.DiscoveredAgent, error)` — returns results of last agent scan, triggering one on-demand if cache is stale (>30 s).
- `DelegateToNexus(ctx context.Context, sessionID string) (delegationInstruction string, error)` — marks the session as delegated, writes the delegation instruction, and returns the instruction text for the caller to display or inject.

### TASK-278 — sys_scanner: Agent-specific probes

Extend `internal/adapters/outbound/sys_scanner/scanner.go` with agent-focused probes, implemented in a new file `agent_scanner.go` in the same package. The `Scanner` struct shall implement `ports.AgentScanner` via a `ScanAgents(ctx)` method that fans out these five probe types concurrently (same semaphore/WaitGroup pattern as `Scan`):

1. **Claude config probe** (`probeClaudeConfig`): `os.ReadFile(os.Getenv("HOME") + "/.claude/settings.json")`. If the file exists and is non-empty JSON, emit a `DiscoveredAgent{Kind: "claude-cli", DetectionMethod: "fs-config", ConfigPath: …, IsRunning: false}`. Separately, `ReadDir("~/Library/Application Support/Claude/")` on macOS (or `~/.config/Claude` on Linux); if the directory exists emit `Kind: "claude-desktop"`. Never fail on missing files — treat `os.IsNotExist` as "not installed".

2. **VS Code extension directory probe** (`probeVSCodeExtensions`): `ReadDir("~/.vscode/extensions/")`. Match directory names against a static table of known AI extension ID prefixes:
   ```
   "saoudrizwan.claude-dev"   → "cline"
   "continue.continue"        → "continue"
   "codeium.codeium"          → "codeium"
   "codegpt.codegpt"          → "codegpt"
   "anysphere.cursor"         → "cursor"
   "github.copilot"           → "copilot"
   ```
   For each match, emit a `DiscoveredAgent{Kind: <kind>, DetectionMethod: "vscode-extension", IsRunning: false}`.

3. **MCP port sweep** (`probeMCPPorts`): Dial and send a minimal JSON-RPC `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"nexus-probe","version":"1"}}}` to ports 3000, 3001, 3100, 5100, 6006, 7007, 8008, 9009. If the response contains `"result"` and a `serverInfo.name` field, emit `DiscoveredAgent{Kind: "generic", DetectionMethod: "port-mcp", MCPEndpoint: "http://127.0.0.1:<port>/mcp", Name: serverInfo.name}`. Use a 500 ms per-port timeout.

4. **Process flag probe** (`probeProcessFlags`): Run `pgrep -lf "\-\-mcp"` and `pgrep -lf "\-\-mcp-server"`. For each line matched, parse out the process name and emit a `DiscoveredAgent{Kind: "generic", DetectionMethod: "process-flag", ProcessName: …, IsRunning: true}`. Skip if `pgrep` is unavailable (Windows).

5. **Existing process patterns** for agents vs providers: the `processPatterns` table currently includes Claude, Antigravity, ChatGPT, Copilot. Copy these same patterns into `ScanAgents` with appropriate `AgentKind` mapping so they appear in both `ScanAgents` output (as `DiscoveredAgent`) and `Scan` output (as `DiscoveredProvider`) without code duplication — use a shared internal `runPgrep(ctx, pattern)` helper.

Results are deduplicated by `Kind` — if multiple probes find the same agent type, merge into one record (prefer `IsRunning: true` over `false`).

### TASK-279 — SQLite: additive migrations

Following the existing pattern in `internal/adapters/outbound/repo_sqlite/repo.go`, add to the `additiveColumnMigrations` loop:

**On `ai_sessions` table:**
```sql
ALTER TABLE ai_sessions ADD COLUMN delegated_to_nexus  INTEGER NOT NULL DEFAULT 0;
ALTER TABLE ai_sessions ADD COLUMN delegation_timestamp DATETIME;
ALTER TABLE ai_sessions ADD COLUMN agent_capabilities   TEXT NOT NULL DEFAULT '[]';
ALTER TABLE ai_sessions ADD COLUMN detection_method     TEXT NOT NULL DEFAULT '';
```

**New `discovered_agents` table** (created in the `CREATE TABLE IF NOT EXISTS` block):
```sql
CREATE TABLE IF NOT EXISTS discovered_agents (
    id               TEXT PRIMARY KEY,
    kind             TEXT NOT NULL,
    name             TEXT NOT NULL,
    detection_method TEXT NOT NULL DEFAULT '',
    process_name     TEXT NOT NULL DEFAULT '',
    cli_path         TEXT NOT NULL DEFAULT '',
    config_path      TEXT NOT NULL DEFAULT '',
    mcp_endpoint     TEXT NOT NULL DEFAULT '',
    is_running       INTEGER NOT NULL DEFAULT 0,
    last_seen        DATETIME NOT NULL
);
```
Add `CREATE INDEX IF NOT EXISTS idx_discovered_agents_kind ON discovered_agents(kind)`. Update `scanAISession` / `SaveAISession` in `ai_session_repo.go` to read/write the four new columns. Provide a `DiscoveredAgentRepo` struct (file: `discovered_agent_repo.go`) implementing `UpsertDiscoveredAgent(ctx, domain.DiscoveredAgent) error` and `ListDiscoveredAgents(ctx) ([]domain.DiscoveredAgent, error)`. `UpsertDiscoveredAgent` uses `INSERT OR REPLACE` keyed on `id = kind` (agents are singletons per kind on a host) plus `last_seen = CURRENT_TIMESTAMP`.

### TASK-280 — OrchestratorService: DelegateToNexus + GetDiscoveredAgents

In `internal/core/services/orchestrator.go`, add two new fields: `agentScanner ports.AgentScanner` and `discoveredAgentRepo` (a local interface accepting `UpsertDiscoveredAgent` + `ListDiscoveredAgents` to avoid importing the sqite package). Wire them via `SetAgentScanner(s ports.AgentScanner)` and `SetDiscoveredAgentRepo(r ...)` setters consistent with the existing `SetAISessionRepo` pattern.

`GetDiscoveredAgents(ctx)`: acquires a read-lock on a `lastAgentScan time.Time` field; if stale (>30 s) or zero, calls `agentScanner.ScanAgents(ctx)`, upserts each result into `discoveredAgentRepo`, updates `lastAgentScan`, then returns the list. Falls back to reading from `discoveredAgentRepo` if scanner is nil (headless / test mode).

`DelegateToNexus(ctx, sessionID)`: Look up the `AISession` via `aiSessionRepo.GetAISessionByID`. Return `domain.ErrNotFound` if missing. Compose the delegation instruction string using the template described in Layer 3 (Delegation Instruction Format). Save updated session with `DelegatedToNexus = true`, `DelegationTimestamp = time.Now()`. Return the instruction text. This method is pure business logic — it does NOT write files or open terminals; that is the VS Code extension's responsibility (Layer 2/3).

### TASK-281 — HTTP API: discovered agents + delegate + SSE task feed

Three additions to `internal/adapters/inbound/httpapi/server.go`:

**`GET /api/ai-sessions/discovered`** — calls `orch.GetDiscoveredAgents(ctx)`, returns `200 OK` with JSON array of `domain.DiscoveredAgent`. Registered as a **literal segment** before the `{id}` wildcard in the router (i.e., before `r.Get("/api/ai-sessions/{id}/tasks", ...)`). Triggers on-demand scan; never returns a 5xx for scan errors — log and return `[]` with a warning header.

**`POST /api/ai-sessions/{id}/delegate`** — calls `orch.DelegateToNexus(ctx, id)`. On success: `200 OK` with `{"instruction": "<text>", "sessionId": "<id>"}`. On `domain.ErrNotFound`: `404`. On other errors: `500`.

**`GET /api/ai-sessions/{id}/tasks` (upgrade to SSE-capable)** — the existing handler returns a JSON snapshot. Change it to honour `Accept: text/event-stream`: if the client requests SSE, subscribe to the `Hub`, filter events by `sessionID == id`, and stream `TaskEvent` messages as `data: <json>\n\n`. If `Accept` is `application/json` (default), keep the existing synchronous behaviour. This is an additive change — existing callers are unaffected.

Register routes in `Handler()`:
```go
r.Get("/api/ai-sessions/discovered", s.handleGetDiscoveredAgents)
r.Post("/api/ai-sessions/{id}/delegate", s.handleDelegateToNexus)
```
`discovered` must be registered *before* the `{id}` group to avoid chi routing the literal word as an ID.

### TASK-282 — VS Code: `AgentDetector` class

New file: `vscode-extension/src/agentDetector.ts`.

`AgentDetector` runs a 30-second `setInterval` polling loop (started by `start()`, stopped by `stop()`/`dispose()`). Each tick calls `detectAll()` which runs the following strategies in parallel (`Promise.allSettled`):

**Strategy 1 — VS Code extension API** (`detectVSCodeExtensions`):
```typescript
const knownAIExtensions: Record<string, string> = {
  'saoudrizwan.claude-dev': 'Cline',
  'continue.continue': 'Continue',
  'codeium.codeium': 'Codeium',
  'codegpt.codegpt-4': 'CodeGPT',
  'anysphere.cursor-always-local': 'Cursor AI',
  'github.copilot': 'GitHub Copilot',
  'github.copilot-chat': 'GitHub Copilot Chat',
  'anthropic.claude': 'Claude (VS Code)',
};
for (const [extId, name] of Object.entries(knownAIExtensions)) {
  const ext = vscode.extensions.getExtension(extId);
  if (ext) yield { agentName: name, source: 'vscode-discovered', externalId: `ext:${extId}`, capabilities: ['chat'] };
}
```

**Strategy 2 — Filesystem config sniffing** (`detectFromFilesystem`): Uses `vscode.workspace.fs` (or Node `fs.promises` with `os.homedir()`) to read:
- `~/.claude/settings.json` → if parseable JSON with `apiKey`, emit `{ agentName: "Claude CLI", … }`
- `~/Library/Application Support/Claude/` (macOS) or `~/.config/claude/` → if directory exists, emit `{ agentName: "Claude Desktop", … }`
- `~/AppData/Roaming/Anthropic/Claude/` (Windows)
- MCP registrations: read `~/.config/claude/claude_desktop_config.json` (or macOS equivalent `~/Library/Application Support/Claude/claude_desktop_config.json`) and parse the `mcpServers` map — for each entry emit a `{ agentName: "MCP: <serverName>", source: "vscode-discovered", capabilities: ["mcp-client"] }`.

**Strategy 3 — Terminal session sniffing** (`detectFromTerminals`): Iterate `vscode.window.terminals`. For each terminal whose `name` or `creationOptions.shellPath` matches patterns `/(claude|cline|continue|cursor|copilot)/i`, emit a detected agent with `capabilities: ["terminal", "code-execute"]`.

**Strategy 4 — Active LM participant detection** (`detectActiveLMParticipants`): Call `vscode.lm.selectChatModels({})` (no vendor filter). For each model, if `model.vendor !== "copilot"`, emit an agent named `${model.vendor}/${model.family}`.

After collecting all detected agents, `AgentDetector` maintains an internal `Map<externalId, AISession>`. For new entries it calls `client.registerSession(...)` with `source: "vscode-discovered"`. For entries still present it calls `client.heartbeatSession(sessionId)`. For entries that disappeared (were in map last tick, not in current tick), it calls `client.deregisterSession(sessionId)` and removes from map. Emits a `vscode.EventEmitter<void>` `onDidChange` for tree view refresh.

### TASK-283 — VS Code: `AISessionsTreeProvider`

New file: `vscode-extension/src/aiSessionsTreeProvider.ts`. Implements `vscode.TreeDataProvider<AISessionItem>`. Registered as a new tree view `nexus.aiSessions` in `package.json` (contributes to the `nexus` view container, after `nexus.workspaceAgents`). Polls `GET /api/ai-sessions` every 15 s. Each `AISessionItem` displays:
- Label: `agentName`
- Description: `status` badge + `projectPath` (if set, shortened to last two path segments)
- Icon: `$(robot)` for source=vscode-discovered, `$(gear)` for source=mcp, `$(person)` for source=vscode
- Context value: `"aiSession"` — used to attach inline action buttons via `package.json` menus
- Colour: green for `active` (+ `delegatedToNexus`), yellow for `active` (not delegated), orange for `idle`, grey for `disconnected`

Tree node `AISessionItem` carries the full `AISession` object for command dispatch. Supports a `nexus.refreshAISessions` command. Inline menu item `nexus.delegateToNexus` shown on each item via `"when": "viewItem == aiSession"` in `package.json`.

### TASK-284 — VS Code: `nexus.delegateToNexus` command

New file: `vscode-extension/src/commands/delegateToNexus.ts`. The command receives a `AISessionItem` tree node (or prompts the user to select a session via `vscode.window.showQuickPick` if invoked from the command palette). It calls `client.delegateSession(session.id)` (TASK-285) to get the `instruction` string from the daemon, then dispatches to one of three paths based on `session.source` and `session.agentName`:

**Path A — CLI / Terminal agents** (`Claude CLI`, `Cline`, `Continue`, or any `terminal`-capable session):
1. Write `<projectPath>/.nexus-delegate.md` with the delegation instruction via `vscode.workspace.fs.writeFile`.
2. Open a new VS Code terminal: `vscode.window.createTerminal({ name: "Nexus Delegate" })`.
3. Send `cat .nexus-delegate.md` to the terminal so the user sees the instruction, plus a comment: `# Paste above into your CLI session`.
4. Show info message: `"Delegation instruction written to .nexus-delegate.md"`.

**Path B — MCP-connected agents** (source=`mcp` or `agentName` matches `Claude Desktop`, `Antigravity`):
1. Call `client.submitTask({ instruction: delegationInstruction, projectPath, targetFile: ".nexus-delegate.md", command: "auto" })` to create a QUEUED nexus task that captures the work.
2. Open a new Copilot Chat (if `vscode.lm` is available) with the delegation text pre-filled via `vscode.commands.executeCommand("workbench.action.chat.open", { query: delegationInstruction })`.
3. Notify: `"Task submitted to Nexus. MCP-connected agent will pick it up from the queue."`.

**Path C — GitHub Copilot in VS Code**:
1. Open Copilot Chat via `vscode.commands.executeCommand("workbench.action.chat.open", { query: delegationInstruction })`.
2. No file write needed — the chat window is the delivery mechanism.

After dispatch, refresh the `AISessionsTreeProvider` so the delegated session turns green.

**Delegation instruction format** (canonical, generated by `DelegateToNexus` in Go):
```
You are now operating under nexusOrchestrator coordination.

YOUR NEW WORKFLOW:
1. Before starting any task, check the nexus queue: GET http://127.0.0.1:63987/api/tasks
2. Claim the next QUEUED task: POST http://127.0.0.1:63987/api/tasks/{id}/claim
   Body: {"sessionId": "<YOUR_SESSION_ID>"}
3. Execute the task's instruction against the task's projectPath/targetFile.
4. Report completion: PUT http://127.0.0.1:63987/api/tasks/{id}/status
   Body: {"sessionId": "<YOUR_SESSION_ID>", "status": "COMPLETED", "logs": "<summary>"}
5. Repeat from step 1.

If the nexus queue is empty, wait 30 seconds and check again.
Do NOT take autonomous actions outside the nexus task queue while this delegation is active.

Nexus session ID: <SESSION_ID>
Project: <PROJECT_PATH>
Timestamp: <ISO8601>
```

### TASK-285 — VS Code: `NexusClient` additions

In `vscode-extension/src/nexusClient.ts`, add:

```typescript
// New types
export interface DiscoveredAgent {
  id: string;
  kind: string;
  name: string;
  detectionMethod: string;
  isRunning: boolean;
  mcpEndpoint?: string;
  configPath?: string;
  lastSeen: string;
}

export interface DelegateResponse {
  instruction: string;
  sessionId: string;
}

// New methods on NexusClient
async getDiscoveredAgents(): Promise<DiscoveredAgent[]> {
  return this.get<DiscoveredAgent[]>('/api/ai-sessions/discovered');
}

async delegateSession(sessionId: string): Promise<DelegateResponse> {
  return this.post<DelegateResponse>(`/api/ai-sessions/${encodeURIComponent(sessionId)}/delegate`, {});
}
```

Also extend `AISession` interface to include:
```typescript
delegatedToNexus?: boolean;
delegationTimestamp?: string;
agentCapabilities?: string[];
```

### TASK-286 — Frontend: "AI Agents" dashboard section

Add a new route `/agents` to the Vue frontend router. New view file: `frontend/src/views/AIAgentsView.vue`. The view:
- Polls `GET /api/ai-sessions` every 10 s (use the existing `useInterval` composable pattern).
- Also fetches `GET /api/ai-sessions/discovered` every 30 s to show unregistered agents.
- Renders two sections: **Registered Sessions** (top card per session) and **Discovered (Unregistered)** agents.

**Registered session card** colour scheme:
- ● Green (`#4ade80`): `status === "active"` AND `delegatedToNexus === true`
- ● Yellow (`#facc15`): `status === "active"` AND `delegatedToNexus === false`
- ● Orange (`#fb923c`): `status === "idle"`
- ● Gray (`#6b7280`): `status === "disconnected"`

Each card shows: agent name, source badge, project path (last 2 segments), last activity relative timestamp, capability chips (`file-write`, `mcp-client`, etc.), and a **"Delegate →"** button. The **"Delegate All"** button in the section header fires `POST /api/ai-sessions/{id}/delegate` for every non-delegated active session in sequence (no bulk endpoint needed yet).

Add a nav link to `AIAgentsView` in the sidebar alongside Tasks and Providers.

### TASK-287 — Frontend: Per-agent task timeline

Within `AIAgentsView.vue`, clicking a session card expands an inline timeline panel. The panel fetches `GET /api/ai-sessions/{id}/tasks` and renders tasks sorted by `updatedAt` descending, grouped by date. Each timeline entry shows: task status chip, target file, instruction excerpt (first 80 chars), and relative time. If the client supports SSE (`EventSource`), upgrade to streaming via `Accept: text/event-stream` so new tasks appear in real time without polling. Maximum 50 tasks shown; "Load more" paginates via `?before=<timestamp>` (add this optional query param to the handler in TASK-281 — low-priority, can be deferred).

### TASK-288 — Extension wiring

In `vscode-extension/src/extension.ts`:
1. Instantiate `AgentDetector` after `SessionMonitor`.
2. Register views: `nexus.aiSessions` → `AISessionsTreeProvider`.
3. Register commands: `nexus.delegateToNexus`, `nexus.refreshAISessions`.
4. Wire `AgentDetector.onDidChange` → `aiSessionsTreeProvider.refresh()`.

In `vscode-extension/package.json`:
- Add view `nexus.aiSessions` to the `nexus` view container.
- Add command `nexus.delegateToNexus` with title `"Send to Nexus Orchestrator"`, icon `$(arrow-up)`.
- Add command `nexus.refreshAISessions` with title `"Refresh AI Agents"`, icon `$(refresh)`.
- Add to `menus["view/item/context"]`: `{ "command": "nexus.delegateToNexus", "when": "viewItem == aiSession", "group": "inline" }`.
- Add to `menus["view/title"]` for `nexus.aiSessions`: refresh command.

Wire `AgentDetector` into `context.subscriptions` so it is disposed on extension deactivation.

### TASK-289 — Go tests

In `internal/adapters/outbound/sys_scanner/agent_scanner_test.go`:
- Test `probeVSCodeExtensions` with a temp dir containing fake extension directories (`saoudrizwan.claude-dev-1.2.3`, `continue.continue-0.9.0`).
- Test `probeMCPPorts` with an `httptest.Server` responding with a valid MCP `initialize` response on a port added to the sweep list via a test hook.
- Test `probeProcessFlags` is skipped on Windows (`runtime.GOOS == "windows"`).

In `internal/core/services/orchestrator_delegate_test.go`:
- Test `DelegateToNexus` on a session that exists → returns instruction string, session has `DelegatedToNexus = true`.
- Test `DelegateToNexus` on a nonexistent session → returns `domain.ErrNotFound`.
- Test `GetDiscoveredAgents` with a mock `AgentScanner` returning 2 agents → upserts both, returns 2.
- Test `GetDiscoveredAgents` with nil scanner → falls back to repo, returns whatever was stored.

Follow `CGO_ENABLED=1 go test -race ./...` — all tests must pass under `-race`.

### TASK-290 — TS tests

In `vscode-extension/src/agentDetector.test.ts` (vitest):
- Mock `vscode.extensions.getExtension` to return a non-null value for `saoudrizwan.claude-dev` → assert `client.registerSession` called with `agentName: "Cline"`.
- Mock the filesystem read for `~/.claude/settings.json` (use `vi.spyOn(fs.promises, 'readFile')`) to return `{"apiKey":"sk-ant-test"}` → assert Claude CLI session registered.
- Simulate agent disappearance (present in tick 1, absent in tick 2) → assert `client.deregisterSession` called.

In `vscode-extension/src/commands/delegateToNexus.test.ts`:
- Mock `client.delegateSession` returning a test instruction.
- For source=`"vscode"` / agentName=`"GitHub Copilot"` → assert `vscode.commands.executeCommand("workbench.action.chat.open", ...)` called.
- For source=`"vscode-discovered"` / capabilities including `"terminal"` → assert `vscode.workspace.fs.writeFile` called with `.nexus-delegate.md`.

---

## New Domain Type Definitions

### Go — `internal/core/domain/discovered_agent.go` (new file)

```go
package domain

import "time"

// AgentKind classifies a detected AI agent/tool.
type AgentKind string

const (
    AgentKindClaudeCLI     AgentKind = "claude-cli"
    AgentKindClaudeDesktop AgentKind = "claude-desktop"
    AgentKindAntigravity   AgentKind = "antigravity"
    AgentKindCline         AgentKind = "cline"
    AgentKindContinue      AgentKind = "continue"
    AgentKindCodeGPT       AgentKind = "codegpt"
    AgentKindCursor        AgentKind = "cursor"
    AgentKindCopilot       AgentKind = "copilot"
    AgentKindAichat        AgentKind = "aichat"
    AgentKindGeneric       AgentKind = "generic"
)

// DiscoveredAgent represents a running AI agent/tool on the local machine.
// It is distinct from DiscoveredProvider which represents an LLM server.
type DiscoveredAgent struct {
    ID              string    `json:"id"`              // Kind is used as ID (singleton per host)
    Kind            AgentKind `json:"kind"`
    Name            string    `json:"name"`
    DetectionMethod string    `json:"detectionMethod"` // "fs-config"|"vscode-extension"|"port-mcp"|"process"|"process-flag"
    ProcessName     string    `json:"processName,omitempty"`
    CLIPath         string    `json:"cliPath,omitempty"`
    ConfigPath      string    `json:"configPath,omitempty"`
    MCPEndpoint     string    `json:"mcpEndpoint,omitempty"`
    IsRunning       bool      `json:"isRunning"`
    LastSeen        time.Time `json:"lastSeen"`
}
```

### Go — additions to `internal/core/domain/ai_session.go`

```go
// New source constant
SessionSourceVSCodeDiscovered AISessionSource = "vscode-discovered"

// New fields on AISession
DelegatedToNexus     bool       `json:"delegatedToNexus"`
DelegationTimestamp  *time.Time `json:"delegationTimestamp,omitempty"`
AgentCapabilities    []string   `json:"agentCapabilities,omitempty"`
DetectionMethod      string     `json:"detectionMethod,omitempty"`
```

### TypeScript — `vscode-extension/src/agentDetector.ts` (key interfaces)

```typescript
export interface DetectedAgent {
  agentName: string;
  source: 'vscode-discovered';
  externalId: string;          // e.g. "ext:saoudrizwan.claude-dev"
  projectPath?: string;
  capabilities: string[];      // e.g. ["file-write", "code-execute", "mcp-client", "chat"]
  detectionMethod: string;     // "vscode-extension" | "fs-config" | "terminal" | "lm-participant"
}
```

---

## New HTTP API Endpoints

### `GET /api/ai-sessions/discovered`

Triggers (or returns cached) `AgentScanner.ScanAgents()` result.

**Response 200:**
```json
[
  {
    "id": "claude-cli",
    "kind": "claude-cli",
    "name": "Claude CLI",
    "detectionMethod": "fs-config",
    "configPath": "/home/user/.claude/settings.json",
    "isRunning": false,
    "lastSeen": "2026-03-14T10:00:00Z"
  },
  {
    "id": "cline",
    "kind": "cline",
    "name": "Cline",
    "detectionMethod": "vscode-extension",
    "isRunning": false,
    "lastSeen": "2026-03-14T10:00:00Z"
  }
]
```

### `POST /api/ai-sessions/{id}/delegate`

**Request body:** `{}` (no body required; sessionID is in path)

**Response 200:**
```json
{
  "instruction": "You are now operating under nexusOrchestrator coordination...",
  "sessionId": "session-uuid-here"
}
```

**Response 404:**
```json
{ "error": "task not found" }
```

### `GET /api/ai-sessions/{id}/tasks` (extended)

Existing: returns `[]Task` as JSON.  
New: if `Accept: text/event-stream`, streams Server-Sent Events:
```
data: {"type":"task.queued","taskId":"t-001","status":"QUEUED"}\n\n
data: {"type":"task.completed","taskId":"t-001","status":"COMPLETED"}\n\n
```
No breaking change — JSON clients unaffected.

---

## New VS Code Commands

| Command ID | Title | Trigger | Description |
|---|---|---|---|
| `nexus.delegateToNexus` | Send to Nexus Orchestrator | Tree view item inline button + command palette | Delegate selected AI session to nexus queue |
| `nexus.refreshAISessions` | Refresh AI Agents | Tree view title button | Manually trigger `AgentDetector.detectAll()` |
| `nexus.delegateAllSessions` | Delegate All Active Agents | Command palette only | Fires delegation for every non-delegated active session |

---

## SQLite Schema Changes

All changes are additive (ALTER TABLE) and follow the existing safeguarded migration loop. No destructive migrations.

### Changes to `ai_sessions` table

```sql
ALTER TABLE ai_sessions ADD COLUMN delegated_to_nexus  INTEGER  NOT NULL DEFAULT 0;
ALTER TABLE ai_sessions ADD COLUMN delegation_timestamp DATETIME;          -- nullable
ALTER TABLE ai_sessions ADD COLUMN agent_capabilities   TEXT     NOT NULL DEFAULT '[]';
ALTER TABLE ai_sessions ADD COLUMN detection_method     TEXT     NOT NULL DEFAULT '';
```

Update `SaveAISession` and `scanAISession` in `ai_session_repo.go` to include these four columns.

### New `discovered_agents` table

```sql
CREATE TABLE IF NOT EXISTS discovered_agents (
    id               TEXT PRIMARY KEY,   -- equals Kind (singleton per host)
    kind             TEXT NOT NULL,
    name             TEXT NOT NULL,
    detection_method TEXT NOT NULL DEFAULT '',
    process_name     TEXT NOT NULL DEFAULT '',
    cli_path         TEXT NOT NULL DEFAULT '',
    config_path      TEXT NOT NULL DEFAULT '',
    mcp_endpoint     TEXT NOT NULL DEFAULT '',
    is_running       INTEGER NOT NULL DEFAULT 0,
    last_seen        DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_discovered_agents_kind ON discovered_agents(kind);
```

---

## Execution Wave Plan

```
Wave 1 (parallel — pure type work, no runtime dependencies)
├── TASK-276  Domain types
└── (research only)

Wave 2 (parallel — ports + infrastructure)
├── TASK-277  Port interfaces          (requires TASK-276)
├── TASK-278  sys_scanner probes       (requires TASK-276)
├── TASK-279  SQLite migrations        (requires TASK-276)
└── TASK-285  NexusClient additions    (TypeScript, independent)

Wave 3 (sequential on Wave 2)
├── TASK-280  OrchestratorService      (requires TASK-277, TASK-279)
├── TASK-282  AgentDetector TS         (requires TASK-285)
└── (other TS classes can start)

Wave 4 (parallel — surface layer)
├── TASK-281  HTTP API endpoints       (requires TASK-280)
├── TASK-283  AISessionsTreeProvider   (requires TASK-285)
└── TASK-284  delegateToNexus command  (requires TASK-282, TASK-285)

Wave 5 (parallel — wiring + UI)
├── TASK-286  Frontend AI Agents view  (requires TASK-281)
├── TASK-287  Frontend timeline        (requires TASK-286)
└── TASK-288  Extension wiring         (requires TASK-282, TASK-283, TASK-284)

Wave 6 (parallel — tests, after implementations complete)
├── TASK-289  Go tests
└── TASK-290  TS tests
```

---

## Risks & Open Questions

### R1 — macOS API access for process list
`pgrep -lf` works on macOS/Linux but is unavailable on Windows. `ScanAgents` must gate process probes behind `runtime.GOOS != "windows"`. As a Windows fallback, use `tasklist /FO CSV /NH` parsed to match process name patterns.

### R2 — VS Code filesystem read permissions
`vscode.workspace.fs` can only read workspace-relative URIs. Reads of `~/.claude/settings.json` and other home-directory paths require Node's `fs.promises` module directly. This works inside the VS Code extension host (Node runtime) but requires graceful fallback when paths don't exist — all reads must be wrapped in try/catch.

### R3 — Terminal sniffing reliability
`vscode.window.terminals` exposes terminal names but not live process output. Detection of "Claude CLI running in terminal X" is heuristic (name matching). False positives are possible. Mark such sessions with `detectionMethod: "terminal"` and lower confidence — they appear in the tree view but are greyish until a heartbeat confirms them.

### R4 — MCP port sweep side effects
The MCP probe sends a JSON-RPC `initialize` to arbitrary localhost ports. This could produce unexpected behaviour on any service that happens to accept a TCP connection and reads from it. Mitigate with a 500 ms timeout and by only probing the short curated port list (not a full sweep). Add a `nexus.enableMCPPortSweep` VS Code setting defaulting to `true` with a clear description of what it does.

### R5 — Delegation instruction delivery for Claude Desktop (no stdin access)
Claude Desktop has no programmatic API for injecting instructions. Path B (MCP tools/call) only works if Claude Desktop has nexus registered as an MCP server (`mcpServers` in its config). If it does, we can send a tool call. If not, delegation falls back to writing `.nexus-delegate.md` and showing the user a notification to paste it. Consider adding a "Copy to clipboard" action on the delegation instruction notification.

### R6 — `DelegationTimestamp` nil pointer in Go JSON serialisation
The `*time.Time` field must be serialised as `null` when nil (not omitted by default since `omitempty` on pointer types omits `null`). Use `omitempty` tag consistently. In SQLite `SaveAISession`, write `NULL` when nil and parse nullable `DATETIME` back to `*time.Time` using `sql.NullString` + `time.Parse`.

### R7 — Concurrent scan + tree refresh
`AgentDetector` runs every 30 s; if the VS Code window is unloaded mid-tick, `registerSession` calls will fail with network errors. Wrap every call in try/catch and log to the activity channel (do not surface as VS Code error notifications). Add an `isDisposed` guard checked before each API call.

### R8 — Session dedup with `externalId`
The `externalId` scheme for vscode-discovered sessions must be globally unique per agent instance. Proposed format: `discover:<machineId>:<agentKind>:<workspacePath>`. The `<workspacePath>` component scopes the session to the VS Code instance's current workspace, so two VS Code windows with different workspaces get separate sessions for the same agent kind — which is correct.

---

## Definition of Done (PLAN-044)

- [ ] `GET /api/ai-sessions/discovered` returns at least Claude CLI / Cline when installed
- [ ] `POST /api/ai-sessions/{id}/delegate` marks session, returns canonical instruction
- [ ] VS Code extension detects Copilot, Cline, Continue via extension API and registers sessions
- [ ] VS Code extension detects Claude CLI / Desktop via filesystem sniff
- [ ] `AgentDetector` deregisters a session within 60 s of agent disappearance
- [ ] "AI Agents" tree view in VS Code shows all registered sessions with colour coding
- [ ] `nexus.delegateToNexus` writes `.nexus-delegate.md` for CLI agents; opens Copilot Chat for Copilot
- [ ] "AI Agents" dashboard section renders in Wails GUI + browser at `:63987/ui`
- [ ] `CGO_ENABLED=1 go test -race ./...` passes
- [ ] All TS unit tests pass with `npm test` in `vscode-extension/`
