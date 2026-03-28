---
id: TASK-346
title: ModelCapabilityProfile domain type + built-in known profiles registry
role: backend
planId: PLAN-051
status: done
dependencies: []
createdAt: 2026-03-28T00:00:00Z
---

## Context

Local models like `qwen3.5-35b-a3b` have known context limits. nexusOrchestrator needs a domain type to represent model capability profiles so that tools can size responses appropriately and users/models can register their own profiles.

## Files to Read

- `internal/core/domain/provider.go` — existing domain types for reference
- `internal/core/domain/task.go` — TaskStatus pattern
- `internal/core/ports/ports.go` — port interface patterns

## Implementation Steps

1. Create `internal/core/domain/model_capability.go`:

```go
package domain

import "time"

// ModelCapabilityProfile describes the known capabilities of a language model.
type ModelCapabilityProfile struct {
    // ModelID is the canonical model identifier (e.g. "qwen3.5-35b-a3b").
    ModelID string `json:"modelId"`
    // ContextWindow is the maximum number of tokens the model can process in one request.
    ContextWindow int `json:"contextWindow"`
    // RecommendedMaxOutput is the suggested max tokens to request in a reply.
    RecommendedMaxOutput int `json:"recommendedMaxOutput,omitempty"`
    // Notes is a free-text description of strengths / weaknesses.
    Notes string `json:"notes,omitempty"`
    // BuiltIn is true for profiles shipped with nexusOrchestrator (not user-defined).
    BuiltIn bool `json:"builtIn,omitempty"`
    // CreatedAt is when the profile was first stored.
    CreatedAt time.Time `json:"createdAt,omitempty"`
    // UpdatedAt is the last modification time.
    UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

// BuiltInModelProfiles contains known context sizes for popular local models.
// Keys are lowercase model IDs (partial match is acceptable).
var BuiltInModelProfiles = []ModelCapabilityProfile{
    {ModelID: "qwen3.5-35b-a3b",  ContextWindow: 32768,  RecommendedMaxOutput: 4096,  Notes: "MoE 35B, good reasoning, ~32K context. Prefer compact tool responses.", BuiltIn: true},
    {ModelID: "qwen3-coder-next",  ContextWindow: 131072, RecommendedMaxOutput: 8192,  Notes: "Large context coder variant.", BuiltIn: true},
    {ModelID: "llama3.2",          ContextWindow: 131072, RecommendedMaxOutput: 4096,  Notes: "Meta Llama 3.2 instruction-tuned.", BuiltIn: true},
    {ModelID: "llama3.1",          ContextWindow: 131072, RecommendedMaxOutput: 4096,  Notes: "Meta Llama 3.1.", BuiltIn: true},
    {ModelID: "codestral",         ContextWindow: 32768,  RecommendedMaxOutput: 8192,  Notes: "Mistral code model.", BuiltIn: true},
    {ModelID: "mistral",           ContextWindow: 32768,  RecommendedMaxOutput: 4096,  Notes: "Mistral 7B/8x7B.", BuiltIn: true},
    {ModelID: "deepseek-coder",    ContextWindow: 16384,  RecommendedMaxOutput: 4096,  Notes: "DeepSeek Coder models.", BuiltIn: true},
    {ModelID: "phi-4",             ContextWindow: 16384,  RecommendedMaxOutput: 2048,  Notes: "Microsoft Phi-4 small but capable.", BuiltIn: true},
    {ModelID: "gemma",             ContextWindow: 8192,   RecommendedMaxOutput: 2048,  Notes: "Google Gemma family.", BuiltIn: true},
}

// LookupBuiltInProfile searches BuiltInModelProfiles for a profile whose
// ModelID is contained in (or equal to) the given model string.
// Returns nil if no match is found.
func LookupBuiltInProfile(modelID string) *ModelCapabilityProfile {
    lower := strings.ToLower(modelID)
    for i, p := range BuiltInModelProfiles {
        if strings.Contains(lower, strings.ToLower(p.ModelID)) ||
            strings.Contains(strings.ToLower(p.ModelID), lower) {
            return &BuiltInModelProfiles[i]
        }
    }
    return nil
}
```

2. Add `"strings"` import to the file.

3. Add `ModelCapabilityRepository` interface to `internal/core/ports/ports.go`:

```go
// ModelCapabilityRepository persists user-defined model capability profiles.
type ModelCapabilityRepository interface {
    Save(p domain.ModelCapabilityProfile) error
    GetByModelID(modelID string) (domain.ModelCapabilityProfile, error)
    GetAll() ([]domain.ModelCapabilityProfile, error)
    Delete(modelID string) error
}
```

## Acceptance Criteria

- `domain.ModelCapabilityProfile` compiles cleanly
- `domain.LookupBuiltInProfile("qwen3.5-35b-a3b")` returns the qwen profile
- `domain.LookupBuiltInProfile("qwen3-coder-next")` returns that profile
- `domain.LookupBuiltInProfile("gpt2")` returns nil
- `ports.ModelCapabilityRepository` interface is defined
- `go vet ./internal/core/...` passes
