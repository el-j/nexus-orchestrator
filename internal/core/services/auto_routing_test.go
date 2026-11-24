package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"
)

// The model names of today's cloud providers are classified: strong families are frontier, small and
// fast ones standard, so that Auto routing has something to choose by.
func TestResolveModelTierKnowsCurrentCloudFamilies(t *testing.T) {
	cases := map[string]domain.ModelTier{
		"claude-fable-5.1":      domain.ModelTierFrontier,
		"gpt-6.2-sol":           domain.ModelTierFrontier,
		"gpt-5.3-codex":         domain.ModelTierFrontier,
		"grok-4.7":              domain.ModelTierFrontier,
		"kimi-k3":               domain.ModelTierFrontier,
		"gemini-4-neon":         domain.ModelTierFrontier,
		"claude-haiku-5.5":      domain.ModelTierStandard,
		"haiku":                 domain.ModelTierStandard,
		"gemini-3.8-flash-low":  domain.ModelTierStandard,
		"mai-code-2-flash":      domain.ModelTierStandard,
		"gpt-6-mini":            domain.ModelTierStandard,
		"gemini-4-neon-lite":    domain.ModelTierStandard,
		"claude-sonnet-5.5":     domain.ModelTierFrontier,
		"some-unknown-model-x1": domain.ModelTierStandard,
	}
	for model, want := range cases {
		if got := services.ResolveModelTier(model, ""); got != want {
			t.Errorf("ResolveModelTier(%q) = %q, want %q", model, got, want)
		}
	}
}

// A task that is not trivial goes to a capable model first, not to the cheapest one that happens to be
// listed first.
func TestAutoRoutingPrefersACapableModelOverAFastOneForRealWork(t *testing.T) {
	haiku := &mockLLMClient{alive: true, name: "Claude (subscription)", activeModel: "haiku"}
	flash := &mockLLMClient{alive: true, name: "vscode-copilot-gemini-3.8-flash", activeModel: "gemini-3.8-flash"}
	strong := &mockLLMClient{alive: true, name: "vscode-copilot-claude-sonnet-5.5", activeModel: "claude-sonnet-5.5"}
	task := domain.Task{Instruction: "write the CAD scripts for the front axle parts", ProjectPath: t.TempDir()}
	got := services.OrderChainByAffinity([]ports.LLMClient{haiku, flash, strong}, task, 0)
	if got[0] != ports.LLMClient(strong) {
		t.Errorf("first choice = %s/%s, want the capable model", got[0].ProviderName(), got[0].ActiveModel())
	}
	small := domain.Task{Instruction: "fix typo in the readme", ProjectPath: t.TempDir()}
	if first := services.OrderChainByAffinity([]ports.LLMClient{strong, haiku, flash}, small, 0)[0]; first == ports.LLMClient(strong) {
		t.Errorf("a typo fix should not need the strongest model")
	}
}

// Without a ladder a failed verification is corrected by the next candidate, not by the same model that
// just failed; with a single candidate it stays.
func TestAFailedVerificationAsksTheNextCandidateWhenThereIsNoLadder(t *testing.T) {
	first := &mockLLMClient{alive: true, name: "first", code: "v1"}
	second := &mockLLMClient{alive: true, name: "second", code: "v2"}
	repo := newMemRepo()
	runner := &mockCommandRunner{}
	runner.runFunc = func(context.Context, string, string) (string, error) {
		if runner.calls.Load() == 1 {
			return "red", errors.New("exit 1")
		}
		return "green", nil
	}
	orch := services.NewOrchestrator(services.NewDiscoveryService(first, second), repo, &recordingWriter{}, nil, services.WithCommandRunner(runner))
	defer orch.Stop()
	id, err := orch.SubmitTask(domain.Task{Instruction: "x", TargetFile: "a.go", ProjectPath: t.TempDir(), VerificationCommand: "check", MaxCorrectionTurns: 2})
	if err != nil {
		t.Fatal(err)
	}
	task := waitStatus(t, repo, id, domain.StatusCompleted, 10*time.Second)
	if want := "verification passed on correction turn 1/2 (check) — completed via second"; !strings.Contains(task.Logs, want) {
		t.Errorf("the correction did not go to the second candidate: %q", task.Logs)
	}
}
