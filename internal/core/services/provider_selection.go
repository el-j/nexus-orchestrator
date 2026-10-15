package services

import (
	"context"
	"errors"
	"fmt"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"strings"
)

// isFrontierRole returns true for complex reasoning, architectural, and planning roles.
func isFrontierRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "architect", "techlead", "planner", "frontier", "deep_debug", "complex", "backend":
		return true
	default:
		return false
	}
}

// isFastLocalRole returns true for routine, formatting, and boilerplate tasks.
func isFastLocalRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "linter", "formatter", "test", "tester", "boilerplate", "fast", "local":
		return true
	default:
		return false
	}
}

// isFrontierProvider tests if a provider matches known frontier cloud LLMs.
func isFrontierProvider(p ports.LLMClient) bool {
	name := strings.ToLower(p.ProviderName())
	return strings.Contains(name, "anthropic") ||
		strings.Contains(name, "gemini") ||
		strings.Contains(name, "openai") ||
		strings.Contains(name, "copilot") ||
		strings.Contains(name, "claude")
}

// isLocalProvider tests if a provider matches known local/on-premise LLMs.
func isLocalProvider(p ports.LLMClient) bool {
	name := strings.ToLower(p.ProviderName())
	return strings.Contains(name, "ollama") ||
		strings.Contains(name, "lmstudio") ||
		strings.Contains(name, "lm studio") ||
		strings.Contains(name, "localai") ||
		strings.Contains(name, "vllm") ||
		strings.Contains(name, "local")
}

// resolveProviderWithFallback returns an ordered chain of available LLM providers
// based on task requirements, role-driven routing hints, and health status.
func (o *OrchestratorService) resolveProviderWithFallback(ctx context.Context, task domain.Task) ([]ports.LLMClient, error) {
	if o.discovery == nil {
		return nil, errors.New("discovery service not configured")
	}

	// Case 1: Explicit ProviderName specified
	if task.ProviderName != "" {
		client, ok := o.discovery.GetClientByName(task.ProviderName)
		if !ok {
			return nil, fmt.Errorf("provider %q not found or not active", task.ProviderName)
		}
		chain := []ports.LLMClient{client}
		for _, c := range o.discovery.GetAllClients() {
			if !strings.EqualFold(c.ProviderName(), client.ProviderName()) && c.Ping(ctx) {
				chain = append(chain, c)
			}
		}
		return chain, nil
	}

	all := o.discovery.GetAllClients()
	if len(all) == 0 {
		return nil, errors.New("discovery: no active provider available")
	}

	// Check if a specific model was requested
	var preferred ports.LLMClient
	if task.ModelID != "" {
		c, err := o.discovery.FindForModel(task.ModelID, task.ProviderHint)
		if err != nil {
			return nil, err
		}
		preferred = c
	}

	// Filter down to alive providers that can serve the task
	var alive []ports.LLMClient
	for _, c := range all {
		if c.Ping(ctx) {
			if task.ModelID != "" {
				if strings.EqualFold(c.ActiveModel(), task.ModelID) {
					alive = append(alive, c)
				} else if models, err := c.GetAvailableModels(ctx); err == nil {
					for _, m := range models {
						if strings.EqualFold(m, task.ModelID) {
							alive = append(alive, c)
							break
						}
					}
				}
			} else {
				alive = append(alive, c)
			}
		}
	}
	if len(alive) == 0 {
		if task.ModelID != "" {
			return nil, fmt.Errorf("discovery: model %q not available on any registered provider", task.ModelID)
		}
		return nil, errors.New("discovery: no active provider available")
	}

	// Classify alive providers
	var frontierClients, localClients, otherClients []ports.LLMClient
	for _, c := range alive {
		switch {
		case isFrontierProvider(c):
			frontierClients = append(frontierClients, c)
		case isLocalProvider(c):
			localClients = append(localClients, c)
		default:
			otherClients = append(otherClients, c)
		}
	}

	var ordered []ports.LLMClient
	switch {
	case isFastLocalRole(task.Role):
		// Fast local chain: Local -> Other -> Frontier
		ordered = append(ordered, localClients...)
		ordered = append(ordered, otherClients...)
		ordered = append(ordered, frontierClients...)
	case isFrontierRole(task.Role):
		// Frontier chain: Frontier -> Other -> Local
		ordered = append(ordered, frontierClients...)
		ordered = append(ordered, otherClients...)
		ordered = append(ordered, localClients...)
	default:
		// Default / Balanced: preserve alive order, or Frontier first if alive
		if len(frontierClients) > 0 {
			ordered = append(ordered, frontierClients...)
			ordered = append(ordered, otherClients...)
			ordered = append(ordered, localClients...)
		} else {
			ordered = alive
		}
	}

	// Deduplicate, ensuring preferred client is at index 0 if present
	var result []ports.LLMClient
	seen := make(map[string]bool)
	if preferred != nil {
		result = append(result, preferred)
		seen[strings.ToLower(preferred.ProviderName())] = true
	}
	for _, c := range ordered {
		name := strings.ToLower(c.ProviderName())
		if !seen[name] {
			result = append(result, c)
			seen[name] = true
		}
	}

	if len(result) == 0 {
		return nil, errors.New("discovery: no active provider available")
	}
	return result, nil
}
