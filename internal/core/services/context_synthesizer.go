package services

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// ContextSynthesizer dynamically formats and sizes knowledge context for specific model tiers and token budgets.
type ContextSynthesizer struct {
	repo ports.ProjectKnowledgeRepository
}

// NewContextSynthesizer creates a new ContextSynthesizer backed by the given repository.
func NewContextSynthesizer(repo ports.ProjectKnowledgeRepository) *ContextSynthesizer {
	return &ContextSynthesizer{repo: repo}
}

// ResolveModelTier determines the model capability/cost tier from a model identifier or requested tier.
func ResolveModelTier(modelID string, requestedTier domain.ModelTier) domain.ModelTier {
	if requestedTier.IsValid() {
		return requestedTier
	}
	lower := strings.ToLower(strings.TrimSpace(modelID))
	if lower == "" {
		return domain.ModelTierStandard
	}

	frontierKeywords := []string{
		"opus", "sonnet", "gpt-4", "o1-", "o3-",
		"gemini-1.5-pro", "gemini-2.0-flash-thinking", "claude-3-7", "claude-3-5", "claude-3-opus",
	}
	for _, kw := range frontierKeywords {
		if strings.Contains(lower, kw) {
			return domain.ModelTierFrontier
		}
	}

	if domain.LookupBuiltInProfile(lower) != nil {
		return domain.ModelTierLocal
	}

	localKeywords := []string{
		"ollama", "lmstudio", "localai", "vllm", "llama",
		"qwen", "codestral", "mistral", "phi", "gemma", "deepseek",
	}
	for _, kw := range localKeywords {
		if strings.Contains(lower, kw) {
			return domain.ModelTierLocal
		}
	}

	return domain.ModelTierStandard
}

// CalculateTokenBudget computes the dynamic context budget based on model tier, context window, and project policy.
func CalculateTokenBudget(modelID string, requestedBudget int, projectPath string) int {
	tier := ResolveModelTier(modelID, "")
	budget := requestedBudget
	if budget <= 0 {
		switch tier {
		case domain.ModelTierFrontier:
			budget = 350
		case domain.ModelTierLocal:
			budget = 1200
		default:
			budget = 600
		}
	}

	if modelID != "" {
		profile := domain.LookupBuiltInProfile(modelID)
		if profile != nil && profile.ContextWindow > 0 {
			maxHeadroom := profile.ContextWindow / 4
			if budget > maxHeadroom {
				budget = maxHeadroom
			}
		}
	}

	if projectPath != "" {
		policy, _, _ := LoadProjectPolicy(projectPath)
		if policy.ContextBudget > 0 && budget > policy.ContextBudget {
			budget = policy.ContextBudget
		}
	}

	if budget < 50 {
		budget = 50
	}
	return budget
}

var fillerPrefixes = []string{
	"in order to ", "it is important that ", "it is important to remember that ",
	"please ensure that ", "remember that ", "please note that ", "this file is responsible for ",
}

// TailorContent adjusts the density and format of knowledge content according to the target model tier.
func TailorContent(kind domain.KnowledgeKind, content string, tier domain.ModelTier) (string, int) {
	origTokens := estimateStringTokens(content)
	switch tier {
	case domain.ModelTierFrontier:
		lines := strings.Split(content, "\n")
		var outLines []string
		for _, rawLine := range lines {
			trimmed := strings.TrimSpace(rawLine)
			if trimmed == "" {
				continue
			}
			lower := strings.ToLower(trimmed)
			for _, filler := range fillerPrefixes {
				if strings.HasPrefix(lower, filler) {
					trimmed = trimmed[len(filler):]
					break
				}
			}
			if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "```") {
				outLines = append(outLines, trimmed)
				continue
			}
			switch kind {
			case domain.KnowledgeConvention:
				if !strings.HasPrefix(trimmed, "*") && !strings.HasPrefix(trimmed, "-") {
					trimmed = "* INVARIANT: " + trimmed
				}
			case domain.KnowledgeLearning:
				if !strings.HasPrefix(trimmed, "*") && !strings.HasPrefix(trimmed, "-") {
					trimmed = "* AVOID: " + trimmed
				}
			case domain.KnowledgeArchitecture:
				if !strings.HasPrefix(trimmed, "*") && !strings.HasPrefix(trimmed, "-") {
					trimmed = "* ARCH: " + trimmed
				}
			default:
				if !strings.HasPrefix(trimmed, "*") && !strings.HasPrefix(trimmed, "-") {
					trimmed = "* " + trimmed
				}
			}
			outLines = append(outLines, trimmed)
		}
		tailored := strings.Join(outLines, "\n")
		tailoredTokens := estimateStringTokens(tailored)
		saved := origTokens - tailoredTokens
		if saved < 0 {
			saved = 0
		}
		return tailored, saved

	case domain.ModelTierLocal:
		var prefix string
		switch kind {
		case domain.KnowledgeConvention:
			prefix = "[CONVENTION GUIDELINES & STEP-BY-STEP RULES]:\n"
		case domain.KnowledgeArchitecture:
			prefix = "[ARCHITECTURE SPECIFICATION & CONTRACTS]:\n"
		case domain.KnowledgeLearning:
			prefix = "[KNOWN PITFALLS & MITIGATION STRATEGIES]:\n"
		case domain.KnowledgeFileMap:
			prefix = "[PROJECT STRUCTURE & FILE DISPATCH MAP]:\n"
		default:
			prefix = "[REFERENCE CONTEXT]:\n"
		}
		return prefix + content, 0

	default:
		return content, 0
	}
}

// TailorSection applies model-tier transformations to a single context section.
func TailorSection(sec domain.ContextSection, tier domain.ModelTier) (domain.ContextSection, int) {
	tailored, saved := TailorContent(sec.Kind, sec.Content, tier)
	sec.Content = tailored
	sec.Tokens = estimateStringTokens(tailored)
	return sec, saved
}

// SynthesizeSections filters and tailors a slice of context sections within the token budget.
func SynthesizeSections(sections []domain.ContextSection, tier domain.ModelTier, budget int) ([]domain.ContextSection, int, int, bool) {
	var result []domain.ContextSection
	tokensUsed := 0
	tokensSaved := 0
	truncated := false

	for _, sec := range sections {
		tailored, saved := TailorSection(sec, tier)
		if tokensUsed+tailored.Tokens > budget {
			truncated = true
			break
		}
		result = append(result, tailored)
		tokensUsed += tailored.Tokens
		tokensSaved += saved
	}
	return result, tokensUsed, tokensSaved, truncated
}

// Synthesize assembles a tailored ContextResponse according to the query's model tier and budget.
func (s *ContextSynthesizer) Synthesize(ctx context.Context, q domain.ContextQuery) (domain.ContextResponse, error) {
	tier := ResolveModelTier(q.ModelID, q.ModelTier)
	budget := CalculateTokenBudget(q.ModelID, q.MaxTokens, q.ProjectPath)

	cleanPath := q.ProjectPath
	if cleanPath != "" && cleanPath != "*" && cleanPath != "all" {
		cleanPath = filepath.Clean(cleanPath)
	}

	var entries []domain.ProjectKnowledge
	var err error
	if q.Question != "" {
		ftsPath := cleanPath
		if ftsPath == "" {
			ftsPath = "*"
		}
		entries, err = s.repo.SearchFTS(ctx, ftsPath, q.Question, 20)
	} else {
		entries, err = s.repo.GetByProject(ctx, cleanPath)
	}
	if err != nil {
		return domain.ContextResponse{}, fmt.Errorf("context_synthesizer: fetch entries: %w", err)
	}

	focus := strings.ToLower(q.FocusArea)
	resp := domain.ContextResponse{
		ProjectPath: cleanPath,
		TokenBudget: budget,
		ModelTier:   tier,
		Sections:    []domain.ContextSection{},
	}

	byKind := make(map[domain.KnowledgeKind][]domain.ProjectKnowledge)
	for _, e := range entries {
		byKind[e.Kind] = append(byKind[e.Kind], e)
	}
	for k := range byKind {
		sort.Slice(byKind[k], func(i, j int) bool {
			return byKind[k][i].RelevanceScore > byKind[k][j].RelevanceScore
		})
	}

	var rawSections []domain.ContextSection
	for _, kind := range kindPriority {
		for _, e := range byKind[kind] {
			if focus != "" && focus != "all" {
				if kind == domain.KnowledgeConvention || kind == domain.KnowledgeFileMap {
					if !strings.Contains(strings.ToLower(e.Topic), focus) {
						continue
					}
				}
			}
			rawSections = append(rawSections, domain.ContextSection{
				ProjectPath: e.ProjectPath,
				Kind:        e.Kind,
				Topic:       e.Topic,
				Content:     e.Content,
				Tokens:      e.TokenCount,
			})
			if kind == domain.KnowledgeFileMap {
				resp.SuggestedFiles = append(resp.SuggestedFiles, e.Content)
			}
		}
	}

	tailoredSections, tokensUsed, tokensSaved, truncated := SynthesizeSections(rawSections, tier, budget)
	resp.Sections = tailoredSections
	resp.TokensUsed = tokensUsed
	resp.TokensSaved = tokensSaved
	resp.Truncated = truncated

	return resp, nil
}
