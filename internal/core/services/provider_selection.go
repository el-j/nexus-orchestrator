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
		strings.Contains(name, "antigravity") ||
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

	alive := o.aliveFor(ctx, all, task)
	if len(alive) == 0 {
		if task.ModelID != "" {
			return nil, fmt.Errorf("discovery: model %q not available on any registered provider", task.ModelID)
		}
		return nil, errors.New("discovery: no active provider available")
	}

	return dedupeCandidates(preferred, orderByRole(alive, task.Role)), nil // never empty: alive is not, and each of its members is either preferred or the first of its key
}

// aliveFor keeps the providers that answer a ping and, when the task names a model, offer it.
func (o *OrchestratorService) aliveFor(ctx context.Context, all []ports.LLMClient, task domain.Task) []ports.LLMClient {
	var alive []ports.LLMClient
	for _, c := range all {
		if c.Ping(ctx) && offersModel(ctx, c, task.ModelID) {
			alive = append(alive, c)
		}
	}
	return alive
}

// offersModel reports whether the provider serves the model: it is the active one or in its model list.
// An empty model is served by every provider.
func offersModel(ctx context.Context, c ports.LLMClient, model string) bool {
	if model == "" || strings.EqualFold(c.ActiveModel(), model) {
		return true
	}
	models, err := c.GetAvailableModels(ctx)
	if err != nil {
		return false
	}
	for _, m := range models {
		if strings.EqualFold(m, model) {
			return true
		}
	}
	return false
}

// orderByRole sorts the providers into the chain the role calls for: routine roles try local ones first,
// demanding roles the frontier ones, anything else the frontier ones when there are any and otherwise the
// order they were found in.
func orderByRole(alive []ports.LLMClient, role string) []ports.LLMClient {
	var frontier, local, other []ports.LLMClient
	for _, c := range alive {
		switch {
		case isFrontierProvider(c):
			frontier = append(frontier, c)
		case isLocalProvider(c):
			local = append(local, c)
		default:
			other = append(other, c)
		}
	}
	chain := func(groups ...[]ports.LLMClient) []ports.LLMClient {
		var out []ports.LLMClient
		for _, g := range groups {
			out = append(out, g...)
		}
		return out
	}
	switch {
	case isFastLocalRole(role):
		return chain(local, other, frontier)
	case isFrontierRole(role), len(frontier) > 0:
		return chain(frontier, other, local)
	default:
		return alive
	}
}

// dedupeCandidates puts the preferred client first and drops repeats. A provider and model are one
// candidate: another model of the same provider is another candidate (the model ladder needs it).
func dedupeCandidates(preferred ports.LLMClient, ordered []ports.LLMClient) []ports.LLMClient {
	var result []ports.LLMClient
	seen := make(map[string]bool)
	if preferred != nil {
		result = append(result, preferred)
		seen[strings.ToLower(cooldownKey(preferred))] = true
	}
	for _, c := range ordered {
		key := strings.ToLower(cooldownKey(c))
		if !seen[key] {
			result = append(result, c)
			seen[key] = true
		}
	}
	return result
}
