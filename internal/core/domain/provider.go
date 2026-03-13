package domain

import "time"

// ProviderKind identifies the adapter family for a cloud LLM provider.
type ProviderKind string

const (
	ProviderKindLMStudio     ProviderKind = "lmstudio"
	ProviderKindOllama       ProviderKind = "ollama"
	ProviderKindOpenAICompat ProviderKind = "openaicompat"
	ProviderKindAnthropic    ProviderKind = "anthropic"
	ProviderKindLocalAI      ProviderKind = "localai"
	ProviderKindVLLM         ProviderKind = "vllm"
	ProviderKindTextGenUI    ProviderKind = "textgenui"
	ProviderKindCLI          ProviderKind = "cli"
	ProviderKindDesktopApp   ProviderKind = "desktopapp"
)

// String returns the underlying string value of the ProviderKind.
func (k ProviderKind) String() string { return string(k) }

// ProviderConfig is a serialisable configuration record for registering or
// editing an LLM provider at runtime (e.g. through the UI or HTTP API).
type ProviderConfig struct {
	// ID is the unique persistent identifier (UUID) for this provider record.
	ID string `json:"id,omitempty"`
	// Name is the unique human-readable identifier for the provider instance.
	Name string `json:"name"`
	// Kind determines which adapter is constructed.
	Kind ProviderKind `json:"kind"`
	// BaseURL is the API endpoint root (required for openaicompat / lmstudio / ollama).
	BaseURL string `json:"baseUrl,omitempty"`
	// APIKey is the bearer token (required for openaicompat / anthropic).
	APIKey string `json:"apiKey,omitempty"`
	// Model is the default / active model identifier for this provider.
	Model string `json:"model,omitempty"`
	// Enabled controls whether this provider is active.
	Enabled bool `json:"enabled"`
	// CreatedAt is the time the record was first persisted.
	CreatedAt time.Time `json:"createdAt,omitempty"`
	// UpdatedAt is the time the record was last modified.
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

// DiscoveryMethod describes how a provider was found on the local system.
type DiscoveryMethod string

const (
	DiscoveryMethodPort    DiscoveryMethod = "port"
	DiscoveryMethodCLI     DiscoveryMethod = "cli"
	DiscoveryMethodProcess DiscoveryMethod = "process"
)

// DiscoveryStatus describes the level of detection for a discovered provider.
type DiscoveryStatus string

const (
	DiscoveryStatusReachable DiscoveryStatus = "reachable"
	DiscoveryStatusInstalled DiscoveryStatus = "installed"
	DiscoveryStatusRunning   DiscoveryStatus = "running"
)

// DiscoveredProvider is a provider found by the system scanner that has NOT
// yet been promoted to an active (registered) provider.
type DiscoveredProvider struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Kind        ProviderKind    `json:"kind"`
	Method      DiscoveryMethod `json:"method"`
	Status      DiscoveryStatus `json:"status"`
	BaseURL     string          `json:"baseUrl,omitempty"`
	CLIPath     string          `json:"cliPath,omitempty"`
	ProcessName string          `json:"processName,omitempty"`
	Models      []string        `json:"models,omitempty"`
	// ActiveModels lists models currently loaded in memory (non-empty means in-use).
	// Currently populated for Ollama via /api/ps.
	ActiveModels []string `json:"activeModels,omitempty"`
	// Generating is true when the provider is actively producing a response right now.
	Generating bool      `json:"generating,omitempty"`
	LastSeen   time.Time `json:"lastSeen"`
}
