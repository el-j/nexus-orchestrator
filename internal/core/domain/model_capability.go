package domain

import (
	"strings"
	"time"
)

// ModelCapabilityProfile captures the known context-window and output limits
// for a specific local model, along with advisory notes for prompt construction.
type ModelCapabilityProfile struct {
	// ModelID is the canonical identifier used by the provider (e.g. "qwen3-coder-next").
	ModelID string `json:"modelId"`
	// ContextWindow is the maximum number of input tokens the model accepts.
	ContextWindow int `json:"contextWindow"`
	// RecommendedMaxOutput is the suggested maximum number of tokens for a single response.
	RecommendedMaxOutput int `json:"recommendedMaxOutput"`
	// Notes contains human-readable guidance for callers constructing prompts.
	Notes string `json:"notes,omitempty"`
	// BuiltIn is true for profiles shipped with nexus-orchestrator.
	BuiltIn bool `json:"builtIn"`
	// CreatedAt is the time the record was first persisted.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt is the time the record was last modified.
	UpdatedAt time.Time `json:"updatedAt"`
}

// BuiltInModelProfiles is the catalogue of well-known local model profiles
// shipped with nexus-orchestrator. All entries have BuiltIn set to true.
var BuiltInModelProfiles = []ModelCapabilityProfile{
	{
		ModelID:              "qwen3.5-35b-a3b",
		ContextWindow:        32768,
		RecommendedMaxOutput: 4096,
		Notes:                "MoE 35B, good reasoning, ~32K context. Prefer compact tool responses.",
		BuiltIn:              true,
	},
	{
		ModelID:              "qwen3-coder-next",
		ContextWindow:        131072,
		RecommendedMaxOutput: 8192,
		Notes:                "Large context coder variant.",
		BuiltIn:              true,
	},
	{
		ModelID:              "llama3.2",
		ContextWindow:        131072,
		RecommendedMaxOutput: 4096,
		Notes:                "Meta Llama 3.2 instruction-tuned.",
		BuiltIn:              true,
	},
	{
		ModelID:              "llama3.1",
		ContextWindow:        131072,
		RecommendedMaxOutput: 4096,
		Notes:                "Meta Llama 3.1.",
		BuiltIn:              true,
	},
	{
		ModelID:              "codestral",
		ContextWindow:        32768,
		RecommendedMaxOutput: 8192,
		Notes:                "Mistral code model.",
		BuiltIn:              true,
	},
	{
		ModelID:              "mistral",
		ContextWindow:        32768,
		RecommendedMaxOutput: 4096,
		Notes:                "Mistral 7B/8x7B.",
		BuiltIn:              true,
	},
	{
		ModelID:              "deepseek-coder",
		ContextWindow:        16384,
		RecommendedMaxOutput: 4096,
		Notes:                "DeepSeek Coder models.",
		BuiltIn:              true,
	},
	{
		ModelID:              "phi-4",
		ContextWindow:        16384,
		RecommendedMaxOutput: 2048,
		Notes:                "Microsoft Phi-4 small but capable.",
		BuiltIn:              true,
	},
	{
		ModelID:              "gemma",
		ContextWindow:        8192,
		RecommendedMaxOutput: 2048,
		Notes:                "Google Gemma family.",
		BuiltIn:              true,
	},
}

// LookupBuiltInProfile searches BuiltInModelProfiles for a profile that matches
// modelID using case-insensitive substring matching in both directions: the
// lookup key contains the profile's ModelID, or the profile's ModelID contains
// the lookup key.  Returns nil when no match is found.
func LookupBuiltInProfile(modelID string) *ModelCapabilityProfile {
	lower := strings.ToLower(modelID)
	for i := range BuiltInModelProfiles {
		profileLower := strings.ToLower(BuiltInModelProfiles[i].ModelID)
		if strings.Contains(lower, profileLower) || strings.Contains(profileLower, lower) {
			return &BuiltInModelProfiles[i]
		}
	}
	return nil
}
