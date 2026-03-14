package domain

import "time"

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
}
