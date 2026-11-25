package domain

import "time"

// RuntimeConfig is mutable, operator-controlled configuration intended to be
// managed at runtime (e.g. via the Settings UI) and persisted in the database.
//
// Important: tokens are secrets. Callers should mask them in responses and only
// return plaintext when explicitly rotating or setting them.
type RuntimeConfig struct {
	QueueCap int `json:"queueCap"`

	// APIToken gates the HTTP API when non-empty.
	APIToken string `json:"apiToken,omitempty"`
	// MCPToken gates the MCP server when non-empty.
	MCPToken string `json:"mcpToken,omitempty"`

	// DefaultLadder is the global fallback model ladder per role (e.g. "coder", "architect", "default")
	// applied whenever a project's .nexus/policy.json does not define a ladder for that role.
	DefaultLadder map[string][]string `json:"defaultLadder,omitempty"`

	UpdatedAt time.Time `json:"updatedAt"`
}

// RuntimeConfigUpdate is a partial update request for RuntimeConfig.
type RuntimeConfigUpdate struct {
	QueueCap *int `json:"queueCap,omitempty"`

	APIToken *string `json:"apiToken,omitempty"`
	MCPToken *string `json:"mcpToken,omitempty"`

	RotateAPIToken bool `json:"rotateApiToken,omitempty"`
	RotateMCPToken bool `json:"rotateMcpToken,omitempty"`

	DefaultLadder *map[string][]string `json:"defaultLadder,omitempty"`
}
