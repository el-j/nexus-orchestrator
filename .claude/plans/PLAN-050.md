# PLAN-050: Real-Time AI Activity Observatory

**Created:** 2026-03-27
**Goal:** Transform nexusOrchestrator from a passive agent detector into an active real-time AI activity hub that shows what every AI agent is actually doing, across all providers and protocols, without requiring agents to self-register.

## Problem Statement

nexusOrchestrator today successfully **detects** 7 AI agents and 4 providers on the system, but displays **zero live activity**. In reality, multiple AI tools are actively working: Continue+LM Studio chatting in VS Code, Copilot active in 2 projects, Claude Code running in multiple workspaces with subagents, Antigravity running. The dashboard is a detection/inventory tool, not an observatory.

### Root Cause Analysis

| Gap                                 | Description                                                                                                                                                                                    |
| ----------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Detection ≠ Observation**         | `ScanAgents()` finds agents exist (process, config, extension) but doesn't read their activity streams                                                                                         |
| **No passive reading**              | Claude JSONL session files contain complete conversation history (messages, tool calls, token usage, timestamps) — currently only `agentId`, `model`, `cwd` are parsed; 90% of data is ignored |
| **Continue sessions ignored**       | `~/.continue/sessions/` has full JSON chat history with titles, messages, workspace dirs — not read at all                                                                                     |
| **Copilot unobservable**            | No filesystem logs; only detectable via VS Code LM API (extension-side only)                                                                                                                   |
| **No activity stream**              | Frontend only shows session status ("active"/"idle") with no detail about WHAT the agent is doing                                                                                              |
| **Discovered ≠ Sessions**           | Backend discovers agents but doesn't auto-create AISession records — two disconnected data models                                                                                              |
| **"generating" flag always empty**  | `DiscoveredProvider.generating` is never set to true — no mechanism exists                                                                                                                     |
| **VS Code extension scope limited** | SessionMonitor only detects Copilot; AgentDetector only detects existence, not activity                                                                                                        |
| **No unified timeline**             | No cross-agent activity feed showing chronological AI actions                                                                                                                                  |

### Available Passive Data Sources (confirmed by filesystem inspection)

| Source                                          | Data Quality | Freshness      | What's Available                                                                                                                                                                  |
| ----------------------------------------------- | ------------ | -------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Claude JSONL** (`~/.claude/projects/`)        | Excellent    | Real-time      | Full conversations: user/assistant messages, tool_use events, thinking blocks, token usage, timestamps, git branch, model, cwd, session/agent IDs, sidechain/parent relationships |
| **Claude history.jsonl** (`~/.claude/`)         | Very Good    | Real-time      | Global activity log: all CLI commands, project context, timestamps across all workspaces                                                                                          |
| **Continue sessions** (`~/.continue/sessions/`) | Excellent    | File-based     | Full chat history: session titles, messages with roles, context items, workspace dirs, message counts                                                                             |
| **Copilot**                                     | Limited      | Extension-only | Extension presence detectable; no filesystem conversation logs; activity only via VS Code LM API                                                                                  |
| **Antigravity**                                 | Network-only | Probe-based    | HTTP API at :4315 reachable; no local session logs; model listing available                                                                                                       |
| **LM Studio**                                   | Network-only | Probe-based    | HTTP API at :1234; active model detection via existing adapter                                                                                                                    |

## Architecture: Activity Reader Layer

New layer between detection and display:

```
sys_scanner (detection)     →  "agent exists"        (current)
activity_reader (NEW)       →  "agent is doing X"    (to build)
AISession + AIActivity      →  domain model           (to extend)
HTTP API + SSE              →  real-time to frontend   (to extend)
Frontend views              →  visualize activity      (to rebuild)
```

### New Domain Types (Wave 1)

```go
// AIActivity — a single observable action by an AI agent
type AIActivity struct {
    ID            string
    SessionID     string      // links to AISession
    AgentName     string
    ActivityType  string      // "message", "tool_use", "thinking", "file_edit", "generation"
    Summary       string      // human-readable: "Editing server.go", "Running tests", "Thinking..."
    ProjectPath   string
    Model         string
    TokensIn      int
    TokensOut     int
    Timestamp     time.Time
    Metadata      map[string]string  // flexible key-value for type-specific data
}

// ActivityType constants
const (
    ActivityTypeMessage    = "message"
    ActivityTypeToolUse    = "tool_use"
    ActivityTypeThinking   = "thinking"
    ActivityTypeFileEdit   = "file_edit"
    ActivityTypeGeneration = "generation"
)
```

### New Port: ActivityReader

```go
type ActivityReader interface {
    // ReadActivities returns recent activities from all observable sources
    // since the given timestamp. Each reader implementation handles one source.
    ReadActivities(ctx context.Context, since time.Time) ([]domain.AIActivity, error)
    // SourceName returns the reader's identity (e.g., "claude-jsonl", "continue-sessions")
    SourceName() string
}
```

## Tasks — 4 Waves

### Wave 1: Domain Model & Port Contracts (foundation)

- **TASK-328**: Add `AIActivity` domain type and `ActivityReader` port interface
- **TASK-329**: Add `AIActivityRepository` port + SQLite implementation (activities table, last 24h retention)
- **TASK-330**: Extend `AISession` domain with `CurrentActivity`, `MessageCount`, `TokensUsed`, `LastMessage` fields

### Wave 2: Activity Readers (parallel — one per source)

- **TASK-331**: Claude JSONL activity reader (`activity_claude`) — parse `~/.claude/projects/**/*.jsonl` for messages, tool_use, thinking, token usage; track file position for incremental reads
- **TASK-332**: Claude global history reader — parse `~/.claude/history.jsonl` for cross-project CLI commands
- **TASK-333**: Continue sessions reader (`activity_continue`) — parse `~/.continue/sessions/*.json` for chat history, titles, workspace dirs
- **TASK-334**: Network probe activity reader — poll LM Studio `/v1/models`, Ollama `/api/ps`, Antigravity `/v1/models` for active model state; set `generating` flag when models are loaded

### Wave 3: Service Integration & API (connects readers to frontend)

- **TASK-335**: `ActivityService` in core services — periodic poll (5s) of all registered `ActivityReader`s, auto-create/update `AISession` records from discovered activity, broadcast `ai_activity` SSE events
- **TASK-336**: HTTP API endpoints — `GET /api/activities` (list recent, filterable by agent/project/type), `GET /api/activities/timeline` (chronological cross-agent feed), SSE event type `ai_activity_new`
- **TASK-337**: Wire activity service into daemon + Wails entry points, register all readers
- **TASK-338**: Bridge discovered agents → sessions: `ActivityService` auto-creates `AISession` from any `DiscoveredAgent` that has `IsRunning=true` + produces activities

### Wave 4: Frontend — Real-Time Activity Dashboard (parallel views)

- **TASK-339**: Rebuild `LiveActivityView.vue` — unified real-time timeline showing all AI activities chronologically, with agent badges, activity type icons, project tags, token counters, and live-updating relative timestamps
- **TASK-340**: Activity cards component — `AIActivityCard.vue` showing: agent name+model, activity summary, project path, timestamp, token count; different visual treatment per activity type (message=chat bubble, tool_use=terminal icon, thinking=brain icon, file_edit=file icon)
- **TASK-341**: Agent detail panel — click an agent in sidebar to see: current status, active project, conversation depth, last 10 activities, token usage chart, session uptime
- **TASK-342**: Project activity view — `ProjectActivityView.vue` grouped by project path, showing all agents working in each project with their recent activities; answers "what's happening in my project right now?"
- **TASK-343**: Dashboard provider cards — update provider cards on Task Queue to show `generating` state, active model name, and current request count when available

### Wave 5: VS Code Extension Enhancement

- **TASK-344**: Update WorkspaceOrchView to show live activities from daemon (not just orchestrator.json plans) — if daemon reachable, show "Live Activity" tree group with recent activities for the workspace's project path
- **TASK-345**: Activity status bar — show compact activity summary in VS Code status bar: "🤖 3 agents active · 2 generating" with click to open activity panel

## Non-Goals (explicitly excluded)

- Intercepting/modifying agent traffic (read-only observation)
- Requiring agents to install nexus plugins (passive observation only)
- Storing full conversation content (only summaries and metadata)
- Real-time streaming of LLM token generation (too invasive; summarize after completion)

## Privacy & Security Considerations

- Activity readers only extract metadata (summaries, types, counts) — NOT full message content
- User message content is NOT stored in AIActivity; only tool names and file paths
- All data stays local (SQLite); no external transmission
- Activity retention: 24 hours default, configurable via `NEXUS_ACTIVITY_RETENTION`

## Success Criteria

1. Open the dashboard → immediately see what every AI agent is doing across all projects
2. Claude Code subagent chains visible as parent→child relationships
3. Continue chat sessions show with title, message count, and workspace
4. LM Studio/Ollama show when models are actively loaded and serving
5. Unified timeline scrolls through all AI activity chronologically
6. No agent needs to know about nexusOrchestrator — all observation is passive

## Status: COMPLETE

**Completed:** 2026-03-27 — All 18 tasks (TASK-328..TASK-345) implemented and validated. Build clean, all tests pass with `-race`.
