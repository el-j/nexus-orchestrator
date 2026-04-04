package domain

import "time"

// AISessionSource identifies how an AI agent session was registered.
type AISessionSource string

const (
	// SessionSourceMCP is registered via the MCP register_session tool.
	SessionSourceMCP AISessionSource = "mcp"
	// SessionSourceVSCode is pushed by the nexus VS Code extension.
	SessionSourceVSCode AISessionSource = "vscode"
	// SessionSourceHTTP is posted to POST /api/ai-sessions.
	SessionSourceHTTP             AISessionSource = "http"
	SessionSourceVSCodeDiscovered AISessionSource = "vscode-discovered"
)

// AISessionStatus represents the lifecycle state of an AI agent session.
type AISessionStatus string

const (
	SessionStatusActive       AISessionStatus = "active"
	SessionStatusIdle         AISessionStatus = "idle"
	SessionStatusDisconnected AISessionStatus = "disconnected"
)

// IsTerminal returns true only when the session has reached a terminal state
// (i.e. it will no longer receive or process tasks).
func (s AISessionStatus) IsTerminal() bool {
	return s == SessionStatusDisconnected
}

// AISession tracks a registered external AI agent session (GitHub Copilot,
// Claude Code, Cursor, etc.) that nexusOrchestrator can route tasks to.
type AISession struct {
	ID                  string          `json:"id"`
	Source              AISessionSource `json:"source"`
	ExternalID          string          `json:"externalId,omitempty"`
	AgentName           string          `json:"agentName"`
	ProjectPath         string          `json:"projectPath,omitempty"`
	Status              AISessionStatus `json:"status"`
	LastActivity        time.Time       `json:"lastActivity"`
	RoutedTaskIDs       []string        `json:"routedTaskIds,omitempty"`
	CreatedAt           time.Time       `json:"createdAt"`
	UpdatedAt           time.Time       `json:"updatedAt"`
	DelegatedToNexus    bool            `json:"delegatedToNexus"`
	DelegationTimestamp *time.Time      `json:"delegationTimestamp,omitempty"`
	AgentCapabilities   []string        `json:"agentCapabilities,omitempty"`
	DetectionMethod     string          `json:"detectionMethod,omitempty"`
	PID                 int             `json:"pid,omitempty"`
	ModelID             string          `json:"modelId,omitempty"`
	// CurrentActivity is a short summary of what the agent is doing right now.
	// Updated by the activity service from passive observation.
	CurrentActivity string `json:"currentActivity,omitempty"`
	// MessageCount is the total number of messages observed for this session.
	MessageCount int `json:"messageCount,omitempty"`
	// TokensUsed is the total tokens consumed across all observed activities.
	TokensUsed int64 `json:"tokensUsed,omitempty"`
	// LastMessage is a truncated summary of the most recent message (max 100 chars).
	LastMessage string `json:"lastMessage,omitempty"`
}

// AISessionEvent is emitted by OrchestratorService when an AI agent session
// is registered, deregistered, or expires from inactivity.
type AISessionEvent struct {
	Type        string          `json:"type"`
	AISessionID string          `json:"aiSessionId"`
	Status      AISessionStatus `json:"status"`
	Timestamp   time.Time       `json:"timestamp"`
}
