package domain

// ProviderKind identifies the adapter family for a cloud LLM provider.
type ProviderKind string

const (
	ProviderKindLMStudio     ProviderKind = "lmstudio"
	ProviderKindOllama       ProviderKind = "ollama"
	ProviderKindOpenAICompat ProviderKind = "openaicompat"
	ProviderKindAnthropic    ProviderKind = "anthropic"
)

// String returns the underlying string value of the ProviderKind.
func (k ProviderKind) String() string { return string(k) }

// ProviderConfig is a serialisable configuration record for registering or
// editing an LLM provider at runtime (e.g. through the UI or HTTP API).
type ProviderConfig struct {
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
}
