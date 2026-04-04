---
id: TASK-352
title: MCP tools — register_model_capabilities + get_model_capabilities
role: mcp
planId: PLAN-051
status: todo
dependencies: [TASK-347]
createdAt: 2026-03-28T00:00:00Z
---

## Context

Models (or users) should be able to self-register their capabilities with the orchestrator. Then other tools can adapt their response sizes. `register_model_capabilities` saves a profile; `get_model_capabilities` retrieves it (falling back to built-in profiles from TASK-346 if no user profile exists).

## Files to Read

- `internal/adapters/inbound/mcp/tools.go` — tool patterns
- `internal/adapters/inbound/mcp/server.go` — Server struct to see how to add the capability repo
- `internal/core/ports/ports.go` — `ModelCapabilityRepository` interface (TASK-346/347)
- `internal/core/domain/model_capability.go` — `ModelCapabilityProfile` (TASK-346)

## Implementation Steps

### 1. Add `ModelCapabilityRepository` to the MCP `Server` struct

In `server.go`, extend `Server`:

```go
type Server struct {
    orchestrator ports.Orchestrator
    modelCaps    ports.ModelCapabilityRepository // NEW
}

func NewServer(o ports.Orchestrator, modelCaps ports.ModelCapabilityRepository) *Server {
    return &Server{orchestrator: o, modelCaps: modelCaps}
}
```

Update all call sites of `NewServer` in `main.go` and `cmd/nexus-daemon/main.go` to pass the new repo.

### 2. Add `toolRegisterModelCapabilities`

```go
func (s *Server) toolRegisterModelCapabilities(args json.RawMessage) (callToolResult, error) {
    var p struct {
        ModelID              string `json:"model_id"`
        ContextWindow        int    `json:"context_window"`
        RecommendedMaxOutput int    `json:"recommended_max_output"`
        Notes                string `json:"notes"`
    }
    if err := json.Unmarshal(args, &p); err != nil || p.ModelID == "" {
        return callToolResult{}, &mcpError{code: codeInvalidParams, msg: "model_id required"}
    }
    profile := domain.ModelCapabilityProfile{
        ModelID:              p.ModelID,
        ContextWindow:        p.ContextWindow,
        RecommendedMaxOutput: p.RecommendedMaxOutput,
        Notes:                p.Notes,
    }
    if err := s.modelCaps.Save(profile); err != nil {
        return callToolResult{}, err
    }
    return textResult(fmt.Sprintf("Registered capabilities for model %q (context_window: %d)", p.ModelID, p.ContextWindow)), nil
}
```

### 3. Add `toolGetModelCapabilities`

```go
func (s *Server) toolGetModelCapabilities(args json.RawMessage) (callToolResult, error) {
    var p struct {
        ModelID string `json:"model_id"`
    }
    _ = json.Unmarshal(args, &p)

    if p.ModelID != "" {
        // Try user-defined first, then built-in
        prof, err := s.modelCaps.GetByModelID(p.ModelID)
        if err != nil {
            // Fall back to built-in lookup
            if builtIn := domain.LookupBuiltInProfile(p.ModelID); builtIn != nil {
                data, _ := json.MarshalIndent(builtIn, "", "  ")
                return textResult(string(data)), nil
            }
            return textResult(fmt.Sprintf("No profile found for %q. Use register_model_capabilities to add one.", p.ModelID)), nil
        }
        data, _ := json.MarshalIndent(prof, "", "  ")
        return textResult(string(data)), nil
    }

    // Return all: user-defined + built-ins
    userProfs, _ := s.modelCaps.GetAll()
    result := struct {
        UserDefined []domain.ModelCapabilityProfile `json:"userDefined"`
        BuiltIn     []domain.ModelCapabilityProfile `json:"builtIn"`
    }{
        UserDefined: userProfs,
        BuiltIn:     domain.BuiltInModelProfiles,
    }
    data, _ := json.MarshalIndent(result, "", "  ")
    return textResult(string(data)), nil
}
```

### 4. Wire both into `handleToolCall` and `toolList`

Tool list entries:

```go
toolDef{
    Name: "register_model_capabilities",
    Description: "Register your model's context window and capabilities. Call this once per session so the orchestrator can adapt response sizes to your context budget.",
    InputSchema: toolSchema{
        Type: "object",
        Required: []string{"model_id", "context_window"},
        Properties: map[string]toolProp{
            "model_id":               {Type: "string", Description: "Model identifier, e.g. 'qwen3.5-35b-a3b'"},
            "context_window":         {Type: "integer", Description: "Maximum input tokens for this model"},
            "recommended_max_output": {Type: "integer", Description: "Suggested max tokens per response"},
            "notes":                  {Type: "string", Description: "Free-text notes about the model"},
        },
    },
},
toolDef{
    Name: "get_model_capabilities",
    Description: "Get capability profile(s). Pass model_id to look up a specific model (checks user-registered first, then built-in profiles). Omit model_id to list all.",
    InputSchema: toolSchema{
        Type: "object",
        Properties: map[string]toolProp{
            "model_id": {Type: "string", Description: "Model ID to look up. Omit to list all profiles."},
        },
    },
},
```

### 5. Update MCP server tests

Add stubs in `mcp/server_test.go` mock for the new `toolList` entries.

## Acceptance Criteria

- `register_model_capabilities` saves a profile; subsequent `get_model_capabilities` returns it
- `get_model_capabilities {"model_id": "qwen3.5-35b-a3b"}` returns the built-in profile even without registration
- `get_model_capabilities` with no args returns both user-defined and built-in lists
- `go vet ./...` clean; `go test ./...` passes
