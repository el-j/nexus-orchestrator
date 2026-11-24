package services_test

import (
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

func TestEstimateTaskComplexity(t *testing.T) {
	tests := []struct {
		name string
		task domain.Task
		want services.TaskComplexity
	}{
		{
			name: "architect role is high complexity",
			task: domain.Task{Role: "architect", Instruction: "do something"},
			want: services.ComplexityHigh,
		},
		{
			name: "techlead role is high complexity",
			task: domain.Task{Role: "techlead", Instruction: "check things"},
			want: services.ComplexityHigh,
		},
		{
			name: "linter role is low complexity",
			task: domain.Task{Role: "linter", Instruction: "run linter on files"},
			want: services.ComplexityLow,
		},
		{
			name: "design keyword is high complexity",
			task: domain.Task{Instruction: "design the distributed consensus protocol"},
			want: services.ComplexityHigh,
		},
		{
			name: "security audit keyword is high complexity",
			task: domain.Task{Instruction: "perform a security audit on auth handlers"},
			want: services.ComplexityHigh,
		},
		{
			name: "typo keyword is low complexity",
			task: domain.Task{Instruction: "fix typo in doc comments"},
			want: services.ComplexityLow,
		},
		{
			name: "clean up whitespace is low complexity",
			task: domain.Task{Instruction: "clean up whitespace and indentation"},
			want: services.ComplexityLow,
		},
		{
			name: "more than 10 context files is high complexity",
			task: domain.Task{
				Instruction:  "refactor helper functions",
				ContextFiles: make([]string, 12),
			},
			want: services.ComplexityHigh,
		},
		{
			name: "standard feature task is medium complexity",
			task: domain.Task{Instruction: "implement get user profile endpoint"},
			want: services.ComplexityMedium,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := services.EstimateTaskComplexity(tc.task)
			if got != tc.want {
				t.Errorf("EstimateTaskComplexity() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestEstimateTaskAffinity(t *testing.T) {
	// Explicit tokens provided
	task := domain.Task{Role: "architect", Instruction: "design subsystem"}
	prof := services.EstimateTaskAffinity(task, 450)
	if prof.EstimatedPromptTokens != 450 {
		t.Errorf("expected 450 prompt tokens, got %d", prof.EstimatedPromptTokens)
	}
	if prof.RequiredRole != "architect" {
		t.Errorf("expected architect role, got %s", prof.RequiredRole)
	}
	if prof.Complexity != services.ComplexityHigh {
		t.Errorf("expected high complexity, got %v", prof.Complexity)
	}

	// Implicit token calculation with empty role
	taskNoRole := domain.Task{
		Instruction:  "fix typo in readme",
		ContextFiles: []string{"README.md"},
	}
	profNoRole := services.EstimateTaskAffinity(taskNoRole, 0)
	if profNoRole.EstimatedPromptTokens <= 0 {
		t.Errorf("expected computed tokens > 0, got %d", profNoRole.EstimatedPromptTokens)
	}
	if profNoRole.RequiredRole != "coder" {
		t.Errorf("expected default coder role, got %s", profNoRole.RequiredRole)
	}
	if profNoRole.Complexity != services.ComplexityLow {
		t.Errorf("expected low complexity, got %v", profNoRole.Complexity)
	}
}

func TestScoreProviderAffinity(t *testing.T) {
	frontierClient := &mockLLMClient{
		name:        "anthropic",
		activeModel: "claude-3-5-sonnet",
	}
	localClient := &mockLLMClient{
		name:        "ollama",
		activeModel: "qwen3-coder-next",
	}
	smallWindowClient := &mockLLMClient{
		name:        "ollama",
		activeModel: "gemma", // 8192 context window
	}

	// High complexity task
	highProfile := services.TaskAffinityProfile{
		EstimatedPromptTokens: 2000,
		RequiredRole:          "architect",
		Complexity:            services.ComplexityHigh,
	}

	scoreFrontier := services.ScoreProviderAffinity(frontierClient, highProfile)
	if !scoreFrontier.ContextFit {
		t.Errorf("expected frontier client to fit")
	}
	if scoreFrontier.CostScore != 1.0 {
		t.Errorf("expected frontier cost 1.0, got %f", scoreFrontier.CostScore)
	}
	if scoreFrontier.CapabilityScore <= 0.8 {
		t.Errorf("expected high capability score for frontier on high complexity, got %f", scoreFrontier.CapabilityScore)
	}

	// Low complexity task (lint/typo)
	lowProfile := services.TaskAffinityProfile{
		EstimatedPromptTokens: 300,
		RequiredRole:          "linter",
		Complexity:            services.ComplexityLow,
	}
	scoreLocalLow := services.ScoreProviderAffinity(localClient, lowProfile)
	scoreFrontierLow := services.ScoreProviderAffinity(frontierClient, lowProfile)

	if scoreLocalLow.TotalAffinity <= scoreFrontierLow.TotalAffinity {
		t.Errorf("expected local model to have higher affinity for low complexity task (saving cost), local=%f frontier=%f",
			scoreLocalLow.TotalAffinity, scoreFrontierLow.TotalAffinity)
	}

	// Medium complexity task
	medProfile := services.TaskAffinityProfile{
		EstimatedPromptTokens: 800,
		RequiredRole:          "coder",
		Complexity:            services.ComplexityMedium,
	}
	standardClient := &mockLLMClient{
		name:        "custom",
		activeModel: "gpt-custom",
	}
	scoreMedStandard := services.ScoreProviderAffinity(standardClient, medProfile)
	if scoreMedStandard.CapabilityScore != 0.90 {
		t.Errorf("expected standard client to score 0.90 for medium complexity, got %f", scoreMedStandard.CapabilityScore)
	}
	scoreMedFrontier := services.ScoreProviderAffinity(frontierClient, medProfile)
	if scoreMedFrontier.CapabilityScore != 0.90 {
		t.Errorf("expected frontier client to score 0.90 for medium complexity, got %f", scoreMedFrontier.CapabilityScore)
	}

	// Standard tier on high and low complexity
	scoreHighStandard := services.ScoreProviderAffinity(standardClient, highProfile)
	if scoreHighStandard.CapabilityScore != 0.65 {
		t.Errorf("expected 0.65, got %f", scoreHighStandard.CapabilityScore)
	}
	scoreLowStandard := services.ScoreProviderAffinity(standardClient, lowProfile)
	if scoreLowStandard.CapabilityScore != 0.80 {
		t.Errorf("expected 0.80, got %f", scoreLowStandard.CapabilityScore)
	}

	// Context headroom overflow -> pre-flight rejection
	hugeProfile := services.TaskAffinityProfile{
		EstimatedPromptTokens: 10000, // exceeds gemma 8192 window
		RequiredRole:          "coder",
		Complexity:            services.ComplexityMedium,
	}
	scoreSmall := services.ScoreProviderAffinity(smallWindowClient, hugeProfile)
	if scoreSmall.ContextFit {
		t.Errorf("expected small window client to not fit")
	}
	if scoreSmall.TotalAffinity != -1.0 {
		t.Errorf("expected -1.0 affinity for unfit client, got %f", scoreSmall.TotalAffinity)
	}
	if scoreSmall.RejectionReason != "prompt exceeds model context window" {
		t.Errorf("unexpected rejection reason: %s", scoreSmall.RejectionReason)
	}
}

func TestOrderChainByAffinity(t *testing.T) {
	// Chain <= 1 returns unchanged
	single := []ports.LLMClient{&mockLLMClient{name: "one"}}
	if got := services.OrderChainByAffinity(single, domain.Task{}, 0); len(got) != 1 {
		t.Errorf("expected single client returned unchanged")
	}

	frontier := &mockLLMClient{name: "anthropic", activeModel: "claude-3-5-sonnet"}
	local := &mockLLMClient{name: "ollama", activeModel: "qwen3-coder-next"}
	small := &mockLLMClient{name: "ollama", activeModel: "gemma"} // 8192 context window

	chain := []ports.LLMClient{frontier, small, local}

	// 1. Task prompt fits gemma but task is low complexity
	lowTask := domain.Task{Instruction: "fix typo in comments", Role: "linter"}
	orderedLow := services.OrderChainByAffinity(chain, lowTask, 200)
	// Local coder should be preferred over expensive frontier for trivial task
	if orderedLow[0].ActiveModel() != "qwen3-coder-next" && orderedLow[0].ActiveModel() != "gemma" {
		t.Errorf("expected local model to be first for low complexity task, got %s", orderedLow[0].ActiveModel())
	}

	// 2. Task prompt exceeds gemma (10000 tokens) -> small window filtered out
	largeTask := domain.Task{Instruction: "design the distributed system", Role: "architect"}
	orderedLarge := services.OrderChainByAffinity(chain, largeTask, 10000)
	if len(orderedLarge) != 2 {
		t.Fatalf("expected gemma to be filtered out due to context overflow, got %d clients", len(orderedLarge))
	}
	if orderedLarge[0].ActiveModel() != "claude-3-5-sonnet" {
		t.Errorf("expected frontier architect to be preferred for large design task, got %s", orderedLarge[0].ActiveModel())
	}

	// Reverse input order tests b > a branch in SortStableFunc
	revChain := []ports.LLMClient{local, frontier}
	orderedRev := services.OrderChainByAffinity(revChain, largeTask, 5000)
	if orderedRev[0].ActiveModel() != "claude-3-5-sonnet" {
		t.Errorf("expected claude-3-5-sonnet first, got %s", orderedRev[0].ActiveModel())
	}

	// Identical affinity triggers equal comparison (return 0)
	equalChain := []ports.LLMClient{
		&mockLLMClient{name: "local1", activeModel: "codestral"},
		&mockLLMClient{name: "local2", activeModel: "codestral"},
	}
	orderedEqual := services.OrderChainByAffinity(equalChain, lowTask, 100)
	if len(orderedEqual) != 2 {
		t.Errorf("expected 2 clients in equal chain")
	}

	// 3. All clients unfit -> fallback returns candidates rather than empty slice
	small2 := &mockLLMClient{name: "ollama2", activeModel: "gemma"}
	orderedOverflow := services.OrderChainByAffinity([]ports.LLMClient{small, small2}, lowTask, 20000)
	if len(orderedOverflow) != 2 {
		t.Errorf("expected fallback to return candidates when all are unfit, got %d", len(orderedOverflow))
	}
}

func TestEvaluateSessionAffinity(t *testing.T) {
	task := domain.Task{
		ProjectPath: "/projects/repo",
		Role:        "architect",
		Tags:        []string{"security", "auth"},
	}

	// 1. Cross-project session rejected
	crossSess := domain.AISession{
		ID:          "s-cross",
		ProjectPath: "/projects/other",
		Status:      domain.SessionStatusActive,
	}
	s1 := services.EvaluateSessionAffinity(task, crossSess)
	if s1.ProjectMatch || s1.TotalAffinity != -1.0 {
		t.Errorf("expected cross-project session rejected with -1.0 affinity")
	}

	// 2. Disconnected terminal session rejected
	deadSess := domain.AISession{
		ID:          "s-dead",
		ProjectPath: "/projects/repo",
		Status:      domain.SessionStatusDisconnected,
	}
	s2 := services.EvaluateSessionAffinity(task, deadSess)
	if s2.TotalAffinity != -1.0 {
		t.Errorf("expected disconnected session rejected with -1.0 affinity")
	}

	// 3. Active matching session with capabilities
	matchingSess := domain.AISession{
		ID:                "s-match",
		ProjectPath:       "/projects/repo", // exact project path matching
		AgentName:         "Antigravity",
		Status:            domain.SessionStatusActive,
		AgentCapabilities: []string{"architect", "security", "auth"},
		LastActivity:      time.Now(),
	}
	s3 := services.EvaluateSessionAffinity(task, matchingSess)
	if !s3.RoleMatch || !s3.CapabilityMatch || s3.TotalAffinity != 1.0 {
		t.Errorf("expected high clamped affinity for matching session, got: %+v", s3)
	}

	// Session with empty project path matches any project
	anyProjSess := domain.AISession{
		ID:          "s-any",
		ProjectPath: "",
		Status:      domain.SessionStatusActive,
	}
	sAny := services.EvaluateSessionAffinity(task, anyProjSess)
	if !sAny.ProjectMatch {
		t.Errorf("expected empty project path to match")
	}

	// 4. FindBestSessionForTask
	sessions := []domain.AISession{crossSess, deadSess, matchingSess}
	best, ok := services.FindBestSessionForTask(task, sessions)
	if !ok || best.ID != "s-match" {
		t.Errorf("expected best session s-match, got %s (ok=%v)", best.ID, ok)
	}

	// 5. FindBestSessionForTask when no session matches
	noMatch, ok := services.FindBestSessionForTask(task, []domain.AISession{crossSess, deadSess})
	if ok || noMatch.ID != "" {
		t.Errorf("expected no match, got %v", noMatch)
	}
}
