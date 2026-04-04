package domain

import "time"

// ActivityType categorises a single AI agent action for display purposes.
type ActivityType string

const (
	ActivityTypeMessage    ActivityType = "message"
	ActivityTypeToolUse    ActivityType = "tool_use"
	ActivityTypeThinking   ActivityType = "thinking"
	ActivityTypeFileEdit   ActivityType = "file_edit"
	ActivityTypeGeneration ActivityType = "generation"
)

// AIActivity represents a single observable action by an AI agent,
// collected passively from filesystem logs or network probes.
type AIActivity struct {
	ID           string            `json:"id"`
	SessionID    string            `json:"sessionId,omitempty"`
	AgentName    string            `json:"agentName"`
	ActivityType ActivityType      `json:"activityType"`
	Summary      string            `json:"summary"`
	ProjectPath  string            `json:"projectPath,omitempty"`
	Model        string            `json:"model,omitempty"`
	TokensIn     int               `json:"tokensIn,omitempty"`
	TokensOut    int               `json:"tokensOut,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ActivityFilter contains optional filters for querying activities.
type ActivityFilter struct {
	AgentName   string
	ProjectPath string
	Type        ActivityType
	Since       time.Time
	Limit       int
}
