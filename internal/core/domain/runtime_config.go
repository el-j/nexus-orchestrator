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

	UpdatedAt time.Time `json:"updatedAt"`
}

// RuntimeConfigUpdate is a partial update request for RuntimeConfig.
type RuntimeConfigUpdate struct {
	QueueCap *int `json:"queueCap,omitempty"`

	APIToken *string `json:"apiToken,omitempty"`
	MCPToken *string `json:"mcpToken,omitempty"`

	RotateAPIToken bool `json:"rotateApiToken,omitempty"`
	RotateMCPToken bool `json:"rotateMcpToken,omitempty"`
}
