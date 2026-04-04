// Package bootstrap provides shared provider construction logic used by
// all nexus entry points (Wails GUI, daemon, CLI).
package bootstrap

import (
	"fmt"
	"os"
	"strings"

	"nexus-orchestrator/internal/adapters/outbound/llm_anthropic"
	"nexus-orchestrator/internal/adapters/outbound/llm_lmstudio"
	"nexus-orchestrator/internal/adapters/outbound/llm_ollama"
	"nexus-orchestrator/internal/adapters/outbound/llm_openaicompat"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// BuildProviders constructs all default LLM provider clients from environment variables.
// Local providers are always included; cloud providers require env-var API keys.
func BuildProviders() []ports.LLMClient {
	lmStudioURL := os.Getenv("NEXUS_LMSTUDIO_URL")
	if lmStudioURL == "" {
		lmStudioURL = llm_lmstudio.DefaultBaseURL
	}
	ollamaURL := os.Getenv("NEXUS_OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = llm_ollama.DefaultBaseURL
	}
	providers := []ports.LLMClient{
		llm_lmstudio.NewLMStudioAdapter(lmStudioURL),
		llm_ollama.NewOllamaAdapter(ollamaURL, "codellama"),
	}
	if key := os.Getenv("NEXUS_OPENAI_API_KEY"); key != "" {
		model := os.Getenv("NEXUS_OPENAI_MODEL")
		if model == "" {
			model = "gpt-4o-mini"
		}
		providers = append(providers, llm_openaicompat.NewAdapter("OpenAI", "https://api.openai.com/v1", key, model))
	}
	if token := os.Getenv("NEXUS_GITHUBCOPILOT_TOKEN"); token != "" {
		model := os.Getenv("NEXUS_GITHUBCOPILOT_MODEL")
		if model == "" {
			model = "gpt-4o"
		}
		providers = append(providers, llm_openaicompat.NewAdapter("GitHub Copilot", "https://api.githubcopilot.com", token, model))
	}
	if key := os.Getenv("NEXUS_ANTHROPIC_API_KEY"); key != "" {
		model := os.Getenv("NEXUS_ANTHROPIC_MODEL")
		if model == "" {
			model = "claude-3-5-sonnet-20241022"
		}
		providers = append(providers, llm_anthropic.NewAdapter(key, model))
	}
	antigravityURL := os.Getenv("NEXUS_ANTIGRAVITY_URL")
	if antigravityURL == "" {
		antigravityURL = "http://127.0.0.1:4315/v1"
	}
	providers = append(providers, llm_openaicompat.NewAdapter("Antigravity", antigravityURL, "", ""))
	return providers
}

// BuildProviderFromConfig constructs a single LLMClient from a ProviderConfig record.
// Injected into OrchestratorService to keep the services package free of adapter imports.
func BuildProviderFromConfig(cfg domain.ProviderConfig) (ports.LLMClient, error) {
	switch cfg.Kind {
	case domain.ProviderKindLMStudio:
		if cfg.BaseURL == "" {
			cfg.BaseURL = llm_lmstudio.DefaultBaseURL
		}
		return llm_lmstudio.NewLMStudioAdapter(cfg.BaseURL), nil
	case domain.ProviderKindOllama:
		if cfg.BaseURL == "" {
			cfg.BaseURL = llm_ollama.DefaultBaseURL
		}
		return llm_ollama.NewOllamaAdapter(cfg.BaseURL, cfg.Model), nil
	case domain.ProviderKindOpenAICompat:
		return llm_openaicompat.NewAdapter(cfg.Name, cfg.BaseURL, cfg.APIKey, cfg.Model), nil
	case domain.ProviderKindAnthropic:
		return llm_anthropic.NewAdapter(cfg.APIKey, cfg.Model), nil
	case domain.ProviderKindDesktopApp, domain.ProviderKindLocalAI, domain.ProviderKindVLLM, domain.ProviderKindTextGenUI:
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "http://127.0.0.1:4315/v1"
		} else if !strings.HasSuffix(baseURL, "/v1") {
			baseURL = strings.TrimRight(baseURL, "/") + "/v1"
		}
		return llm_openaicompat.NewAdapter(cfg.Name, baseURL, cfg.APIKey, cfg.Model), nil
	default:
		return nil, fmt.Errorf("unknown provider kind: %q", cfg.Kind)
	}
}
