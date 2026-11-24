package services

import (
	"path/filepath"
	"slices"
	"strings"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// TaskComplexity classifies the reasoning and capability requirements of a task.
type TaskComplexity string

const (
	// ComplexityLow denotes simple tasks (typos, linters, comments, small docs).
	ComplexityLow TaskComplexity = "low"
	// ComplexityMedium denotes standard feature development and bug fixes.
	ComplexityMedium TaskComplexity = "medium"
	// ComplexityHigh denotes architectural design, multi-file refactoring, or complex logic.
	ComplexityHigh TaskComplexity = "high"
)

// TaskAffinityProfile captures the estimated requirements used for routing decisions.
type TaskAffinityProfile struct {
	EstimatedPromptTokens int
	RequiredRole          string
	Complexity            TaskComplexity
}

// ProviderAffinityScore represents the evaluated routing score for a candidate LLM client.
type ProviderAffinityScore struct {
	Client          ports.LLMClient
	ModelID         string
	ProviderName    string
	Tier            domain.ModelTier
	ContextHeadroom int
	ContextFit      bool
	CostScore       float64
	CapabilityScore float64
	TotalAffinity   float64
	RejectionReason string
}

// SessionAffinityScore represents affinity between a task and a registered external AI session.
type SessionAffinityScore struct {
	SessionID       string
	AgentName       string
	RoleMatch       bool
	CapabilityMatch bool
	ProjectMatch    bool
	TotalAffinity   float64
}

// highComplexityKeywords identify tasks demanding frontier model reasoning.
var highComplexityKeywords = []string{
	"architect", "design", "redesign", "security audit", "breakdown",
	"milestone", "refactor subsystem", "distributed", "concurrency", "formal spec",
}

// lowComplexityKeywords identify tasks suitable for fast, free local models.
var lowComplexityKeywords = []string{
	"typo", "fix typo", "comment", "docstring", "lint", "format",
	"clean up whitespace", "rename variable", "license",
}

// EstimateTaskComplexity determines the reasoning tier required for a task.
func EstimateTaskComplexity(task domain.Task) TaskComplexity {
	role := strings.ToLower(task.Role)
	if role == "architect" || role == "techlead" {
		return ComplexityHigh
	}
	if role == "linter" {
		return ComplexityLow
	}

	instruction := strings.ToLower(task.Instruction)
	for _, kw := range highComplexityKeywords {
		if strings.Contains(instruction, kw) {
			return ComplexityHigh
		}
	}
	for _, kw := range lowComplexityKeywords {
		if strings.Contains(instruction, kw) {
			return ComplexityLow
		}
	}

	if len(task.ContextFiles) > 10 {
		return ComplexityHigh
	}
	return ComplexityMedium
}

// EstimateTaskAffinity builds a TaskAffinityProfile from a task and optional known prompt token count.
func EstimateTaskAffinity(task domain.Task, promptTokens int) TaskAffinityProfile {
	tokens := promptTokens
	if tokens <= 0 {
		tokens = estimateStringTokens(task.Instruction)
		for _, f := range task.ContextFiles {
			tokens += estimateStringTokens(f) + 50
		}
	}

	role := task.Role
	if role == "" {
		role = "coder"
	}

	return TaskAffinityProfile{
		EstimatedPromptTokens: tokens,
		RequiredRole:          role,
		Complexity:            EstimateTaskComplexity(task),
	}
}

// ScoreProviderAffinity evaluates a candidate client against a task's affinity profile.
func ScoreProviderAffinity(c ports.LLMClient, profile TaskAffinityProfile) ProviderAffinityScore {
	modelID := c.ActiveModel()
	providerName := c.ProviderName()
	tier := ResolveModelTier(modelID, "")

	contextWindow := 32768
	if prof := domain.LookupBuiltInProfile(modelID); prof != nil && prof.ContextWindow > 0 {
		contextWindow = prof.ContextWindow
	}

	headroom := contextWindow - profile.EstimatedPromptTokens
	contextFit := headroom > 0

	var costScore float64
	switch tier {
	case domain.ModelTierFrontier:
		costScore = 1.0
	case domain.ModelTierStandard:
		costScore = 0.5
	default:
		costScore = 0.0
	}

	var capabilityScore float64
	switch profile.Complexity {
	case ComplexityHigh:
		switch tier {
		case domain.ModelTierFrontier:
			capabilityScore = 0.95
		case domain.ModelTierStandard:
			capabilityScore = 0.65
		default:
			capabilityScore = 0.40
		}
	case ComplexityLow:
		switch tier {
		case domain.ModelTierLocal:
			capabilityScore = 0.95
		case domain.ModelTierStandard:
			capabilityScore = 0.80
		default:
			capabilityScore = 0.20
		}
	default: // ComplexityMedium
		switch tier {
		case domain.ModelTierStandard:
			capabilityScore = 0.90
		case domain.ModelTierLocal:
			capabilityScore = 0.80
		default:
			capabilityScore = 0.90
		}
	}

	if profile.Complexity != ComplexityLow && isLightweightModel(strings.ToLower(modelID)) {
		capabilityScore -= 0.35 // a fast model is for routine work, not the first choice for real work
	}

	roleLower := strings.ToLower(profile.RequiredRole)
	if (roleLower == "architect" || roleLower == "techlead") && tier == domain.ModelTierFrontier {
		capabilityScore += 0.1
	} else if roleLower == "linter" && tier == domain.ModelTierLocal {
		capabilityScore += 0.1
	}
	if capabilityScore > 1.0 {
		capabilityScore = 1.0
	}

	var rejectionReason string
	totalAffinity := 0.0
	if !contextFit {
		rejectionReason = "prompt exceeds model context window"
		totalAffinity = -1.0
	} else {
		normHeadroom := float64(headroom) / float64(contextWindow)
		var wCap, wCost, wHead float64
		switch profile.Complexity {
		case ComplexityHigh:
			wCap, wCost, wHead = 0.75, 0.10, 0.15
		case ComplexityLow:
			wCap, wCost, wHead = 0.25, 0.60, 0.15
		default:
			wCap, wCost, wHead = 0.50, 0.30, 0.20
		}
		totalAffinity = (capabilityScore * wCap) + ((1.0 - costScore) * wCost) + (normHeadroom * wHead)
	}

	return ProviderAffinityScore{
		Client:          c,
		ModelID:         modelID,
		ProviderName:    providerName,
		Tier:            tier,
		ContextHeadroom: headroom,
		ContextFit:      contextFit,
		CostScore:       costScore,
		CapabilityScore: capabilityScore,
		TotalAffinity:   totalAffinity,
		RejectionReason: rejectionReason,
	}
}

// OrderChainByAffinity orders candidate providers by capability, cost efficiency, and context headroom.
// Clients with insufficient context window are filtered out unless all candidates are unfit.
func OrderChainByAffinity(chain []ports.LLMClient, task domain.Task, promptTokens int) []ports.LLMClient {
	if len(chain) <= 1 {
		return chain
	}
	profile := EstimateTaskAffinity(task, promptTokens)

	type scored struct {
		client ports.LLMClient
		score  ProviderAffinityScore
	}
	var scoredList []scored
	var fitList []scored

	for _, c := range chain {
		s := ScoreProviderAffinity(c, profile)
		entry := scored{client: c, score: s}
		scoredList = append(scoredList, entry)
		if s.ContextFit {
			fitList = append(fitList, entry)
		}
	}

	candidates := fitList
	if len(candidates) == 0 {
		candidates = scoredList
	}

	slices.SortStableFunc(candidates, func(a, b scored) int {
		if a.score.TotalAffinity > b.score.TotalAffinity {
			return -1
		}
		if a.score.TotalAffinity < b.score.TotalAffinity {
			return 1
		}
		return 0
	})

	out := make([]ports.LLMClient, len(candidates))
	for i, entry := range candidates {
		out[i] = entry.client
	}
	return out
}

// EvaluateSessionAffinity evaluates the matching score between an incoming task and an external AI session.
func EvaluateSessionAffinity(task domain.Task, session domain.AISession) SessionAffinityScore {
	cleanTaskProj := filepath.Clean(task.ProjectPath)
	cleanSessProj := filepath.Clean(session.ProjectPath)
	projectMatch := session.ProjectPath == "" || cleanTaskProj == cleanSessProj

	if !projectMatch || session.Status.IsTerminal() {
		return SessionAffinityScore{
			SessionID:     session.ID,
			AgentName:     session.AgentName,
			ProjectMatch:  projectMatch,
			TotalAffinity: -1.0,
		}
	}

	roleLower := strings.ToLower(task.Role)
	roleMatch := false
	capMatch := false

	for _, cap := range session.AgentCapabilities {
		capLower := strings.ToLower(cap)
		if roleLower != "" && strings.Contains(capLower, roleLower) {
			roleMatch = true
		}
		for _, tag := range task.Tags {
			if strings.Contains(capLower, strings.ToLower(tag)) {
				capMatch = true
			}
		}
	}

	affinity := 0.5
	if session.Status == domain.SessionStatusActive {
		affinity += 0.2
	}
	if roleMatch {
		affinity += 0.2
	}
	if capMatch {
		affinity += 0.1
	}
	if affinity > 1.0 || affinity >= 0.99 {
		affinity = 1.0
	}

	return SessionAffinityScore{
		SessionID:       session.ID,
		AgentName:       session.AgentName,
		RoleMatch:       roleMatch,
		CapabilityMatch: capMatch,
		ProjectMatch:    true,
		TotalAffinity:   affinity,
	}
}

// FindBestSessionForTask returns the registered AI session with the highest affinity for task.
func FindBestSessionForTask(task domain.Task, sessions []domain.AISession) (domain.AISession, bool) {
	var best domain.AISession
	bestScore := -1.0

	for _, s := range sessions {
		score := EvaluateSessionAffinity(task, s)
		if score.TotalAffinity > bestScore && score.TotalAffinity >= 0.5 {
			bestScore = score.TotalAffinity
			best = s
		}
	}
	if bestScore >= 0.5 {
		return best, true
	}
	return domain.AISession{}, false
}
