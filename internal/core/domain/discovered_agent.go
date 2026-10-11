package domain

import "time"

// AgentKind names a kind of AI agent the system scanner can discover on the machine.
type AgentKind string

// The known AgentKind values.
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

// DiscoveredAgent is an AI agent found on the machine by the system scanner,
// as opposed to one that registered itself.
type DiscoveredAgent struct {
	ID              string    `json:"id"`
	Kind            AgentKind `json:"kind"`
	Name            string    `json:"name"`
	DetectionMethod string    `json:"detectionMethod"`
	ProcessName     string    `json:"processName,omitempty"`
	CLIPath         string    `json:"cliPath,omitempty"`
	ConfigPath      string    `json:"configPath,omitempty"`
	MCPEndpoint     string    `json:"mcpEndpoint,omitempty"`
	IsRunning       bool      `json:"isRunning"`
	LastSeen        time.Time `json:"lastSeen"`
	PID             int       `json:"pid,omitempty"`
	ModelID         string    `json:"modelId,omitempty"`       // e.g. "claude-sonnet-4-6", "gpt-4o"
	WorkingDir      string    `json:"workingDir,omitempty"`    // project/cwd of the agent process
	ParentAgentID   string    `json:"parentAgentId,omitempty"` // ID of parent if this is a sub-agent
	SubAgentIDs     []string  `json:"subAgentIds,omitempty"`   // IDs of spawned sub-agents
}

// PlanFileKind identifies the format/tool of a discovered plan file.
type PlanFileKind string

// The PlanFileKind values: which convention a discovered plan file follows.
const (
	PlanFileKindNexus      PlanFileKind = "nexus"       // .claude/orchestrator.json
	PlanFileKindClaudeTask PlanFileKind = "claude-task" // .claude/tasks/*.md
	PlanFileKindMarkdown   PlanFileKind = "markdown"    // TASKS.md / TODO.md / PLAN.md / ROADMAP.md / AGENTS.md
	PlanFileKindCursor     PlanFileKind = "cursor"      // .cursorrules / .cursor/rules/*.mdc
	PlanFileKindMCPConfig  PlanFileKind = "mcp-config"  // mcp.json / .mcp.json / claude_desktop_config.json
	PlanFileKindCrewAI     PlanFileKind = "crewai"      // crew.py / agents.py containing crewai imports
	PlanFileKindClaude     PlanFileKind = "claude"      // CLAUDE.md / copilot-instructions.md / .github/agents/*.agent.md
)

// DiscoveredPlanFile represents a detected orchestration/plan/task file.
type DiscoveredPlanFile struct {
	ID           string       `json:"id"`   // stable: sha1 of path
	Path         string       `json:"path"` // absolute filesystem path
	Kind         PlanFileKind `json:"kind"`
	Format       string       `json:"format"`            // "json", "md", "yaml", "py"
	ProjectPath  string       `json:"projectPath"`       // nearest git root or parent dir
	Summary      string       `json:"summary,omitempty"` // first ~200 chars of content
	LastModified time.Time    `json:"lastModified"`
	IsActive     bool         `json:"isActive"` // modified within last 24h
}
