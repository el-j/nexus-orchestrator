---
id: TASK-051
title: "Entry-point wiring: env-var provider config for daemon + desktop"
role: devops
planId: PLAN-005
status: todo
dependencies: [TASK-047, TASK-048, TASK-049, TASK-050]
createdAt: 2026-03-10T06:00:00.000Z
---

## Context
Wire the new `llm_openaicompat` and `llm_anthropic` adapters into both entry points
(`cmd/nexus-daemon/main.go` and `main.go`) using environment variables for configuration.
Providers are only registered when their required env vars are present.

## Files to Read
- `cmd/nexus-daemon/main.go`
- `main.go`

## Environment Variables

| Variable | Required for | Description |
|---|---|---|
| `NEXUS_OPENAI_API_KEY` | OpenAI | API key |
| `NEXUS_OPENAI_BASE_URL` | OpenAI | Base URL (default: `https://api.openai.com/v1`) |
| `NEXUS_OPENAI_MODEL` | OpenAI | Model (default: `gpt-4o`) |
| `NEXUS_GITHUBCOPILOT_TOKEN` | GitHub Copilot | Token (uses OpenAI-compat adapter) |
| `NEXUS_GITHUBCOPILOT_MODEL` | GitHub Copilot | Model (default: `gpt-4o`) |
| `NEXUS_ANTHROPIC_API_KEY` | Anthropic | API key |
| `NEXUS_ANTHROPIC_MODEL` | Anthropic | Model (default: `claude-sonnet-4-5`) |

## Implementation Steps

### 1. Create a `buildProviders()` helper

In **both** `cmd/nexus-daemon/main.go` and `main.go`, add a `buildProviders()` function:

```go
func buildProviders() []ports.LLMClient {
    var clients []ports.LLMClient

    // Always-on local providers
    clients = append(clients,
        llm_lmstudio.NewLMStudioAdapter("http://127.0.0.1:1234/v1"),
        llm_ollama.NewOllamaAdapter("http://127.0.0.1:11434", "codellama"),
    )

    // OpenAI (only when API key is set)
    if key := os.Getenv("NEXUS_OPENAI_API_KEY"); key != "" {
        baseURL := os.Getenv("NEXUS_OPENAI_BASE_URL")
        if baseURL == "" {
            baseURL = "https://api.openai.com/v1"
        }
        model := os.Getenv("NEXUS_OPENAI_MODEL")
        if model == "" {
            model = "gpt-4o"
        }
        clients = append(clients, llm_openaicompat.NewAdapter("OpenAI", baseURL, key, model))
    }

    // GitHub Copilot (only when token is set)
    if token := os.Getenv("NEXUS_GITHUBCOPILOT_TOKEN"); token != "" {
        model := os.Getenv("NEXUS_GITHUBCOPILOT_MODEL")
        if model == "" {
            model = "gpt-4o"
        }
        clients = append(clients, llm_openaicompat.NewAdapter(
            "GitHub Copilot",
            "https://api.githubcopilot.com",
            token,
            model,
        ))
    }

    // Anthropic Claude (only when API key is set)
    if key := os.Getenv("NEXUS_ANTHROPIC_API_KEY"); key != "" {
        model := os.Getenv("NEXUS_ANTHROPIC_MODEL")
        if model == "" {
            model = "claude-sonnet-4-5"
        }
        clients = append(clients, llm_anthropic.NewAdapter(key, model))
    }

    return clients
}
```

### 2. Update `discoverySvc := services.NewDiscoveryService(...)` in both files
Replace the hardcoded adapter lines with:
```go
discoverySvc := services.NewDiscoveryService(buildProviders()...)
```
Remove the now-unused `lmStudio` and `ollama` variable declarations.

### 3. Add imports
In both files add:
```go
"nexus-orchestrator/internal/adapters/outbound/llm_openaicompat"
"nexus-orchestrator/internal/adapters/outbound/llm_anthropic"
"nexus-orchestrator/internal/core/ports"
```
Remove direct `llm_lmstudio` and `llm_ollama` imports from the top-level wiring
(they are only used inside `buildProviders()` now, so keep them).

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `go build .` exits 0 (desktop app)
- [ ] When `NEXUS_OPENAI_API_KEY` is not set, no OpenAI adapter is registered (no panic)
- [ ] When `NEXUS_ANTHROPIC_API_KEY` is not set, no Anthropic adapter is registered
- [ ] `buildProviders()` always includes LM Studio and Ollama as baseline adapters
- [ ] No provider credentials appear in log output

## Anti-patterns to Avoid
- NEVER log API keys or tokens
- NEVER panic when optional providers are absent
- NEVER hardcode provider URLs outside `buildProviders()` / env-var defaults
