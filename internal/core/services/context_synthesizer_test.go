package services_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

type synthMockKnowledgeRepo struct {
	entries []domain.ProjectKnowledge
	err     error
}

func (m *synthMockKnowledgeRepo) SaveKnowledge(_ context.Context, _ domain.ProjectKnowledge) error {
	return nil
}
func (m *synthMockKnowledgeRepo) GetByID(_ context.Context, _ string) (domain.ProjectKnowledge, error) {
	return domain.ProjectKnowledge{}, domain.ErrNotFound
}
func (m *synthMockKnowledgeRepo) UpdateKnowledge(_ context.Context, _ domain.ProjectKnowledge) error {
	return nil
}
func (m *synthMockKnowledgeRepo) DeleteKnowledge(_ context.Context, _ string) error {
	return nil
}
func (m *synthMockKnowledgeRepo) GetByProject(_ context.Context, _ string) ([]domain.ProjectKnowledge, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.entries, nil
}
func (m *synthMockKnowledgeRepo) GetByProjectAndKind(_ context.Context, _ string, _ domain.KnowledgeKind) ([]domain.ProjectKnowledge, error) {
	return nil, nil
}
func (m *synthMockKnowledgeRepo) SearchFTS(_ context.Context, _, _ string, _ int) ([]domain.ProjectKnowledge, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.entries, nil
}
func (m *synthMockKnowledgeRepo) SearchVector(_ context.Context, _ string, _ []float32, _ int) ([]domain.ProjectKnowledge, error) {
	return nil, nil
}
func (m *synthMockKnowledgeRepo) GetStatus(_ context.Context, _ string) (domain.BrainStatus, error) {
	return domain.BrainStatus{}, nil
}
func (m *synthMockKnowledgeRepo) DeleteByProject(_ context.Context, _ string) error {
	return nil
}

func TestResolveModelTier(t *testing.T) {
	tests := []struct {
		name     string
		modelID  string
		reqTier  domain.ModelTier
		wantTier domain.ModelTier
	}{
		{name: "explicit frontier tier", modelID: "", reqTier: domain.ModelTierFrontier, wantTier: domain.ModelTierFrontier},
		{name: "explicit local tier", modelID: "any", reqTier: domain.ModelTierLocal, wantTier: domain.ModelTierLocal},
		{name: "empty model and tier defaults to standard", modelID: "", reqTier: "", wantTier: domain.ModelTierStandard},
		{name: "claude-3-5-sonnet matches frontier", modelID: "claude-3-5-sonnet-20241022", wantTier: domain.ModelTierFrontier},
		{name: "claude-3-opus matches frontier", modelID: "claude-3-opus-20240229", wantTier: domain.ModelTierFrontier},
		{name: "gpt-4o matches frontier", modelID: "gpt-4o-mini", wantTier: domain.ModelTierFrontier},
		{name: "o1-preview matches frontier", modelID: "o1-preview", wantTier: domain.ModelTierFrontier},
		{name: "o3-mini matches frontier", modelID: "o3-mini", wantTier: domain.ModelTierFrontier},
		{name: "gemini-1.5-pro matches frontier", modelID: "gemini-1.5-pro-latest", wantTier: domain.ModelTierFrontier},
		{name: "gemini-2.0-flash-thinking matches frontier", modelID: "gemini-2.0-flash-thinking-exp", wantTier: domain.ModelTierFrontier},
		{name: "claude-3-7 matches frontier", modelID: "claude-3-7-sonnet", wantTier: domain.ModelTierFrontier},
		{name: "ollama matches local", modelID: "ollama/qwen2.5-coder", wantTier: domain.ModelTierLocal},
		{name: "lmstudio matches local", modelID: "lmstudio/mistral", wantTier: domain.ModelTierLocal},
		{name: "localai matches local", modelID: "localai", wantTier: domain.ModelTierLocal},
		{name: "vllm matches local", modelID: "vllm/llama-3", wantTier: domain.ModelTierLocal},
		{name: "qwen matches local", modelID: "qwen2.5:7b", wantTier: domain.ModelTierLocal},
		{name: "codestral matches local", modelID: "codestral-22b", wantTier: domain.ModelTierLocal},
		{name: "phi-4 matches local", modelID: "phi-4", wantTier: domain.ModelTierLocal},
		{name: "gemma matches local", modelID: "gemma-2:9b", wantTier: domain.ModelTierLocal},
		{name: "deepseek matches local", modelID: "deepseek-coder:6.7b", wantTier: domain.ModelTierLocal},
		{name: "builtin profile lookup matches local", modelID: "qwen3.5-35b-a3b", wantTier: domain.ModelTierLocal},
		{name: "unknown model falls back to standard", modelID: "custom-enterprise-corp", wantTier: domain.ModelTierStandard},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := services.ResolveModelTier(tc.modelID, tc.reqTier)
			if got != tc.wantTier {
				t.Errorf("ResolveModelTier(%q, %q) = %q; want %q", tc.modelID, tc.reqTier, got, tc.wantTier)
			}
		})
	}
}

func TestCalculateTokenBudget(t *testing.T) {
	// 1. Requested budget explicit
	if b := services.CalculateTokenBudget("custom", 800, ""); b != 800 {
		t.Errorf("expected 800, got %d", b)
	}

	// 2. Tier defaults
	if b := services.CalculateTokenBudget("gpt-4o", 0, ""); b != 350 {
		t.Errorf("expected 350 for frontier, got %d", b)
	}
	if b := services.CalculateTokenBudget("ollama/llama3", 0, ""); b != 1200 {
		t.Errorf("expected 1200 for local, got %d", b)
	}
	if b := services.CalculateTokenBudget("custom", 0, ""); b != 600 {
		t.Errorf("expected 600 for standard, got %d", b)
	}

	// 3. Headroom cap based on ContextWindow (gemma built-in profile has 8192 context window -> 8192/4 = 2048)
	if b := services.CalculateTokenBudget("gemma", 5000, ""); b != 2048 {
		t.Errorf("expected 2048 cap for gemma, got %d", b)
	}

	// 4. Policy budget capping
	tmpDir := t.TempDir()
	policyDir := filepath.Join(tmpDir, ".nexus")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	policyJSON := `{"contextBudget": 180}`
	if err := os.WriteFile(filepath.Join(policyDir, "policy.json"), []byte(policyJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	if b := services.CalculateTokenBudget("gpt-4o", 0, tmpDir); b != 180 {
		t.Errorf("expected policy cap of 180, got %d", b)
	}

	// 5. Minimum floor
	policyTiny := `{"contextBudget": 20}`
	if err := os.WriteFile(filepath.Join(policyDir, "policy.json"), []byte(policyTiny), 0o600); err != nil {
		t.Fatal(err)
	}
	if b := services.CalculateTokenBudget("gpt-4o", 0, tmpDir); b != 50 {
		t.Errorf("expected floor of 50, got %d", b)
	}
}

func TestTailorContent_Frontier(t *testing.T) {
	input := `
# Conventions
In order to maintain safety:
Please ensure that all models validate input.
* Existing bullet stays bullet.
- Dash bullet stays dash.
`
	tailored, saved := services.TailorContent(domain.KnowledgeConvention, input, domain.ModelTierFrontier)
	if !strings.Contains(tailored, "# Conventions") {
		t.Errorf("expected header to be preserved, got: %s", tailored)
	}
	if !strings.Contains(tailored, "* INVARIANT: maintain safety:") {
		t.Errorf("expected invariant prefix without filler, got: %s", tailored)
	}
	if !strings.Contains(tailored, "* INVARIANT: all models validate input.") {
		t.Errorf("expected filler stripped, got: %s", tailored)
	}
	if !strings.Contains(tailored, "* Existing bullet stays bullet.") {
		t.Errorf("expected existing bullet preserved, got: %s", tailored)
	}
	if !strings.Contains(tailored, "- Dash bullet stays dash.") {
		t.Errorf("expected dash bullet preserved, got: %s", tailored)
	}
	_ = saved

	// Learning kind
	lTailored, _ := services.TailorContent(domain.KnowledgeLearning, "avoid unbounded loops in tests", domain.ModelTierFrontier)
	if !strings.Contains(lTailored, "* AVOID: avoid unbounded loops in tests") {
		t.Errorf("expected avoid prefix, got: %s", lTailored)
	}

	// Architecture kind
	aTailored, _ := services.TailorContent(domain.KnowledgeArchitecture, "hexagonal port layout", domain.ModelTierFrontier)
	if !strings.Contains(aTailored, "* ARCH: hexagonal port layout") {
		t.Errorf("expected arch prefix, got: %s", aTailored)
	}

	// Glossary kind
	gTailored, _ := services.TailorContent(domain.KnowledgeGlossary, "TDD: Test-driven development", domain.ModelTierFrontier)
	if !strings.Contains(gTailored, "* TDD: Test-driven development") {
		t.Errorf("expected bullet prefix, got: %s", gTailored)
	}

	// Code block in frontier
	codeBlock := "```go\ntype Foo interface{}\n```"
	cTailored, _ := services.TailorContent(domain.KnowledgeConvention, codeBlock, domain.ModelTierFrontier)
	if !strings.Contains(cTailored, "```go") {
		t.Errorf("expected code block fences to be preserved, got: %s", cTailored)
	}
}

func TestTailorContent_Local(t *testing.T) {
	content := "Run all unit tests before commit."

	conv, _ := services.TailorContent(domain.KnowledgeConvention, content, domain.ModelTierLocal)
	if !strings.HasPrefix(conv, "[CONVENTION GUIDELINES & STEP-BY-STEP RULES]:\n") {
		t.Errorf("unexpected local convention format: %s", conv)
	}

	arch, _ := services.TailorContent(domain.KnowledgeArchitecture, content, domain.ModelTierLocal)
	if !strings.HasPrefix(arch, "[ARCHITECTURE SPECIFICATION & CONTRACTS]:\n") {
		t.Errorf("unexpected local arch format: %s", arch)
	}

	learn, _ := services.TailorContent(domain.KnowledgeLearning, content, domain.ModelTierLocal)
	if !strings.HasPrefix(learn, "[KNOWN PITFALLS & MITIGATION STRATEGIES]:\n") {
		t.Errorf("unexpected local learning format: %s", learn)
	}

	fmap, _ := services.TailorContent(domain.KnowledgeFileMap, content, domain.ModelTierLocal)
	if !strings.HasPrefix(fmap, "[PROJECT STRUCTURE & FILE DISPATCH MAP]:\n") {
		t.Errorf("unexpected local file_map format: %s", fmap)
	}

	gloss, _ := services.TailorContent(domain.KnowledgeGlossary, content, domain.ModelTierLocal)
	if !strings.HasPrefix(gloss, "[REFERENCE CONTEXT]:\n") {
		t.Errorf("unexpected local glossary format: %s", gloss)
	}
}

func TestTailorContent_Standard(t *testing.T) {
	content := "Standard balanced content"
	got, saved := services.TailorContent(domain.KnowledgeConvention, content, domain.ModelTierStandard)
	if got != content || saved != 0 {
		t.Errorf("expected content unchanged and 0 saved, got %q, %d", got, saved)
	}
}

func TestSynthesizeSections_Truncation(t *testing.T) {
	sections := []domain.ContextSection{
		{Kind: domain.KnowledgeConvention, Content: "Rule 1", Tokens: 1},
		{Kind: domain.KnowledgeConvention, Content: "Rule 2", Tokens: 1},
		{Kind: domain.KnowledgeConvention, Content: "Rule 3", Tokens: 1},
	}

	// Budget fits all
	res, used, _, trunc := services.SynthesizeSections(sections, domain.ModelTierStandard, 10)
	if len(res) != 3 || used != 3 || trunc {
		t.Errorf("expected 3 sections, 3 tokens, not truncated; got %d, %d, %v", len(res), used, trunc)
	}

	// Budget truncates after 2
	res, used, _, trunc = services.SynthesizeSections(sections, domain.ModelTierStandard, 2)
	if len(res) != 2 || used != 2 || !trunc {
		t.Errorf("expected 2 sections, 2 tokens, truncated; got %d, %d, %v", len(res), used, trunc)
	}
}

func TestContextSynthesizer_Synthesize(t *testing.T) {
	now := time.Now()
	entries := []domain.ProjectKnowledge{
		{
			ID:             "1",
			ProjectPath:    "/repo",
			Kind:           domain.KnowledgeArchitecture,
			Topic:          "core",
			Content:        "Architecture overview of system components.",
			TokenCount:     10,
			RelevanceScore: 0.9,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		{
			ID:             "2",
			ProjectPath:    "/repo",
			Kind:           domain.KnowledgeConvention,
			Topic:          "auth",
			Content:        "In order to authenticate users, always verify JWT tokens.",
			TokenCount:     12,
			RelevanceScore: 0.8,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		{
			ID:             "3",
			ProjectPath:    "/repo",
			Kind:           domain.KnowledgeConvention,
			Topic:          "database",
			Content:        "Always use prepared statements.",
			TokenCount:     8,
			RelevanceScore: 0.7,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		{
			ID:             "4",
			ProjectPath:    "/repo",
			Kind:           domain.KnowledgeFileMap,
			Topic:          "auth",
			Content:        "internal/auth/handler.go",
			TokenCount:     4,
			RelevanceScore: 0.85,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}

	repo := &synthMockKnowledgeRepo{entries: entries}
	synth := services.NewContextSynthesizer(repo)

	// 1. Synthesize for Frontier with FocusArea
	q := domain.ContextQuery{
		ProjectPath: "/repo",
		FocusArea:   "auth",
		ModelID:     "claude-3-5-sonnet",
	}
	resp, err := synth.Synthesize(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ModelTier != domain.ModelTierFrontier {
		t.Errorf("expected ModelTierFrontier, got %s", resp.ModelTier)
	}
	if len(resp.SuggestedFiles) != 1 || resp.SuggestedFiles[0] != "internal/auth/handler.go" {
		t.Errorf("expected suggested files to include internal/auth/handler.go, got: %v", resp.SuggestedFiles)
	}
	// "database" convention should be filtered out by focusArea "auth"
	for _, sec := range resp.Sections {
		if sec.Topic == "database" {
			t.Errorf("expected database topic to be filtered out by focus area")
		}
	}

	// 2. Synthesize with Question (FTS path)
	qQuestion := domain.ContextQuery{
		ProjectPath: "",
		Question:    "how to auth",
		ModelTier:   domain.ModelTierLocal,
	}
	respQ, err := synth.Synthesize(context.Background(), qQuestion)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if respQ.ModelTier != domain.ModelTierLocal {
		t.Errorf("expected ModelTierLocal, got %s", respQ.ModelTier)
	}

	// 3. Repo error
	errRepo := &synthMockKnowledgeRepo{err: errors.New("db error")}
	errSynth := services.NewContextSynthesizer(errRepo)
	_, err = errSynth.Synthesize(context.Background(), q)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// 4. BrainService delegation integration
	brain := services.NewBrainService(repo, nil)
	brainResp, err := brain.GetContext(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected brain.GetContext error: %v", err)
	}
	if brainResp.ModelTier != domain.ModelTierFrontier {
		t.Errorf("expected brain.GetContext to delegate to synthesizer with frontier tier, got: %s", brainResp.ModelTier)
	}

	brainFocusedResp, err := brain.GetFocusedContext(context.Background(), domain.ContextQuery{
		ProjectPath: "/repo",
		Question:    "jwt verification",
		ModelID:     "gpt-4o",
	})
	if err != nil {
		t.Fatalf("unexpected brain.GetFocusedContext error: %v", err)
	}
	if brainFocusedResp.ModelTier != domain.ModelTierFrontier {
		t.Errorf("expected brain.GetFocusedContext to delegate, got: %s", brainFocusedResp.ModelTier)
	}
}
