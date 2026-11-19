package gateway

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"

	"github.com/google/uuid"
)

// Config holds configuration options for the Gateway.
type Config struct {
	APIKey      string
	DefaultRole string
	ProjectPath string
}

// Gateway routes chat completions and prompt requests across LLM providers
// using model ladders, cooldown recovery, and failovers.
type Gateway struct {
	cfg       Config
	discovery *services.DiscoveryService
}

// NewGateway creates a new Gateway instance.
func NewGateway(cfg Config, discovery *services.DiscoveryService) *Gateway {
	if cfg.DefaultRole == "" {
		cfg.DefaultRole = "coder"
	}
	return &Gateway{
		cfg:       cfg,
		discovery: discovery,
	}
}

// ResolveChain returns the ordered candidate chain and applicable ladder for modelOrRole.
func (g *Gateway) ResolveChain(ctx context.Context, modelOrRole, projectPath string) ([]ports.LLMClient, []string, error) {
	if g.discovery == nil {
		return nil, nil, errors.New("gateway: discovery service not initialized")
	}
	all := g.discovery.GetAllClients()
	if len(all) == 0 {
		return nil, nil, errors.New("gateway: no LLM providers registered")
	}
	if modelOrRole == "" {
		modelOrRole = g.cfg.DefaultRole
	}
	target := strings.TrimSpace(modelOrRole)
	preferred := findPreferredClient(ctx, all, target)

	// Prefer candidates that respond to ping; if none respond, consider all candidates for preview.
	var alive []ports.LLMClient
	for _, c := range all {
		if c.Ping(ctx) {
			alive = append(alive, c)
		}
	}
	candidates := alive
	if len(candidates) == 0 {
		candidates = all
	}

	policy, ok, err := services.LoadProjectPolicy(projectPath)
	if err != nil || !ok {
		policy = domain.ProjectPolicy{}
	}
	ladder := policy.LadderFor(target)

	ordered := make([]ports.LLMClient, len(candidates))
	copy(ordered, candidates)

	if len(ladder) > 0 {
		pos := func(c ports.LLMClient) int {
			for i, r := range ladder {
				if domain.RungMatches(r, c.ProviderName(), c.ActiveModel()) {
					return i
				}
			}
			return len(ladder)
		}
		slices.SortStableFunc(ordered, func(a, b ports.LLMClient) int {
			return pos(a) - pos(b)
		})
	} else {
		ordered = orderProvidersByRole(ordered, target)
	}

	chain := partitionCooling(ordered, g.discovery, time.Now())
	if preferred != nil {
		var deduped []ports.LLMClient
		deduped = append(deduped, preferred)
		for _, c := range chain {
			if !sameProvider(c, preferred) {
				deduped = append(deduped, c)
			}
		}
		chain = deduped
	}

	return chain, ladder, nil
}

// findPreferredClient finds a client matching target name or model.
func findPreferredClient(ctx context.Context, all []ports.LLMClient, target string) ports.LLMClient {
	for _, c := range all {
		if strings.EqualFold(c.ProviderName(), target) {
			return c
		}
	}
	for _, c := range all {
		if strings.EqualFold(c.ActiveModel(), target) {
			return c
		}
		models, err := c.GetAvailableModels(ctx)
		if err == nil {
			for _, m := range models {
				if strings.EqualFold(m, target) {
					return c
				}
			}
		}
	}
	return nil
}

// partitionCooling partitions cooling providers to the tail of the candidate list.
func partitionCooling(ordered []ports.LLMClient, discovery *services.DiscoveryService, now time.Time) []ports.LLMClient {
	var ready, cooling []ports.LLMClient
	for _, c := range ordered {
		if _, isCooling := discovery.Cooldown(c, now); isCooling {
			cooling = append(cooling, c)
		} else {
			ready = append(ready, c)
		}
	}
	return append(ready, cooling...)
}

// ApplyPolicyBudget trims inbound messages according to project policy or client-specified budget.
func ApplyPolicyBudget(msgs []domain.Message, projectPath string, clientBudget int) []domain.Message {
	budget := clientBudget
	if budget <= 0 && projectPath != "" {
		if policy, ok, _ := services.LoadProjectPolicy(projectPath); ok && policy.HistoryBudget > 0 {
			budget = policy.HistoryBudget
		}
	}
	if budget > 0 {
		return services.TrimHistoryTokens(msgs, budget)
	}
	return msgs
}

// ExecuteChat sends messages to the best available provider with automatic failover.
func (g *Gateway) ExecuteChat(ctx context.Context, req ChatCompletionRequest, projectPath string) (*ChatCompletionResponse, error) {
	chain, _, err := g.ResolveChain(ctx, req.Model, projectPath)
	if err != nil {
		return nil, err
	}

	domainMsgs := ApplyPolicyBudget(toDomainMessages(req.Messages), projectPath, req.HistoryBudget)
	var lastErr error
	now := time.Now()

	for _, client := range chain {
		if _, cooling := g.discovery.Cooldown(client, now); cooling && len(chain) > 1 {
			continue
		}

		reply, err := client.Chat(ctx, domainMsgs)
		if err != nil {
			lastErr = err
			g.discovery.MarkCooldown(client, time.Now().Add(30*time.Second), domain.ErrKindRateLimited, err.Error())
			continue
		}

		g.discovery.ClearCooldown(client)
		resp := &ChatCompletionResponse{
			ID:      "chatcmpl-" + uuid.NewString(),
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   client.ActiveModel(),
			Choices: []ChatChoice{
				{
					Index: 0,
					Message: ChatMessage{
						Role:    "assistant",
						Content: reply,
					},
					FinishReason: "stop",
				},
			},
		}

		if reporter, ok := client.(ports.UsageReporter); ok {
			if u, ok := reporter.LastUsage(); ok {
				resp.Usage = &UsageInfo{
					PromptTokens:     u.InputTokens,
					CompletionTokens: u.OutputTokens,
					TotalTokens:      u.InputTokens + u.OutputTokens,
				}
			}
		}
		return resp, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("gateway: all providers failed, last error: %w", lastErr)
	}
	return nil, errors.New("gateway: no candidate provider available")
}

// StreamChat executes streaming chat completions across the candidate chain with failover on initial connection failure.
func (g *Gateway) StreamChat(ctx context.Context, req ChatCompletionRequest, projectPath string, onChunk func(piece string) error) (*ChatCompletionResponse, error) {
	chain, _, err := g.ResolveChain(ctx, req.Model, projectPath)
	if err != nil {
		return nil, err
	}

	domainMsgs := ApplyPolicyBudget(toDomainMessages(req.Messages), projectPath, req.HistoryBudget)
	var lastErr error
	now := time.Now()

	for _, client := range chain {
		if _, cooling := g.discovery.Cooldown(client, now); cooling && len(chain) > 1 {
			continue
		}

		var fullReply strings.Builder
		emitted := false

		chunkHandler := func(piece string) {
			if piece == "" {
				return
			}
			emitted = true
			fullReply.WriteString(piece)
			_ = onChunk(piece)
		}

		var streamErr error
		if streamer, ok := client.(ports.ChatStreamer); ok {
			_, streamErr = streamer.ChatStream(ctx, domainMsgs, chunkHandler)
		} else {
			reply, err := client.Chat(ctx, domainMsgs)
			if err == nil {
				chunkHandler(reply)
			}
			streamErr = err
		}

		if streamErr != nil {
			if !emitted {
				lastErr = streamErr
				g.discovery.MarkCooldown(client, time.Now().Add(30*time.Second), domain.ErrKindRateLimited, streamErr.Error())
				continue
			}
			return nil, streamErr
		}

		g.discovery.ClearCooldown(client)
		return &ChatCompletionResponse{
			ID:      "chatcmpl-" + uuid.NewString(),
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   client.ActiveModel(),
			Choices: []ChatChoice{
				{
					Index:        0,
					Message:      ChatMessage{Role: "assistant", Content: fullReply.String()},
					FinishReason: "stop",
				},
			},
		}, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("gateway: all providers failed, last error: %w", lastErr)
	}
	return nil, errors.New("gateway: no candidate provider available")
}

// ListModels returns all discovered models and standard virtual role models.
func (g *Gateway) ListModels(ctx context.Context) []ModelItem {
	var items []ModelItem
	seen := make(map[string]bool)

	// Virtual role models
	roles := []string{"coder", "architect", "reviewer", "planner", "linter", "default"}
	for _, r := range roles {
		items = append(items, ModelItem{
			ID:      r,
			Object:  "model",
			Created: 1700000000,
			OwnedBy: "nexus-role",
		})
		seen[strings.ToLower(r)] = true
	}

	if g.discovery == nil {
		return items
	}

	for _, c := range g.discovery.GetAllClients() {
		active := c.ActiveModel()
		if active != "" && !seen[strings.ToLower(active)] {
			items = append(items, ModelItem{
				ID:      active,
				Object:  "model",
				Created: 1700000000,
				OwnedBy: c.ProviderName(),
			})
			seen[strings.ToLower(active)] = true
		}
		models, err := c.GetAvailableModels(ctx)
		if err == nil {
			for _, m := range models {
				if m != "" && !seen[strings.ToLower(m)] {
					items = append(items, ModelItem{
						ID:      m,
						Object:  "model",
						Created: 1700000000,
						OwnedBy: c.ProviderName(),
					})
					seen[strings.ToLower(m)] = true
				}
			}
		}
	}

	return items
}

// toDomainMessages converts inbound ChatMessages to internal domain.Messages.
func toDomainMessages(msgs []ChatMessage) []domain.Message {
	out := make([]domain.Message, len(msgs))
	now := time.Now()
	for i, m := range msgs {
		role := domain.MessageRole(strings.ToLower(m.Role))
		if role != domain.RoleSystem && role != domain.RoleAssistant {
			role = domain.RoleUser
		}
		out[i] = domain.Message{
			Role:      role,
			Content:   m.Content,
			CreatedAt: now,
		}
	}
	return out
}

// orderProvidersByRole sorts providers giving preference to frontier or local providers depending on role.
func orderProvidersByRole(alive []ports.LLMClient, role string) []ports.LLMClient {
	r := strings.ToLower(role)
	var frontier, local, other []ports.LLMClient
	for _, c := range alive {
		name := strings.ToLower(c.ProviderName())
		switch {
		case strings.Contains(name, "anthropic") || strings.Contains(name, "gemini") || strings.Contains(name, "openai") || strings.Contains(name, "copilot"):
			frontier = append(frontier, c)
		case strings.Contains(name, "ollama") || strings.Contains(name, "lmstudio") || strings.Contains(name, "local"):
			local = append(local, c)
		default:
			other = append(other, c)
		}
	}

	if r == "architect" || r == "planner" || r == "reviewer" {
		return append(append(frontier, other...), local...)
	}
	return append(append(local, other...), frontier...)
}

// sameProvider compares two providers by name and active model.
func sameProvider(a, b ports.LLMClient) bool {
	return strings.EqualFold(a.ProviderName(), b.ProviderName()) && a.ActiveModel() == b.ActiveModel()
}
