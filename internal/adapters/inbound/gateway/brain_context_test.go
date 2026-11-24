package gateway

import (
	"context"
	"errors"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// brainStub answers the two calls BrainContext makes.
type brainStub struct {
	ports.BrainService
	onboarding             string
	onboardErr, focusedErr error
	sections               []domain.ContextSection
	asked                  domain.ContextQuery
}

func (b *brainStub) GetOnboardingContext(context.Context, string, int) (string, error) {
	return b.onboarding, b.onboardErr
}

func (b *brainStub) GetFocusedContext(_ context.Context, q domain.ContextQuery) (domain.ContextResponse, error) {
	b.asked = q
	return domain.ContextResponse{Sections: b.sections}, b.focusedErr
}

func TestBrainContext_JoinsOnboardingAndTheKnowledgeThatAnswersTheQuestion(t *testing.T) {
	b := &brainStub{onboarding: "# Onboarding\nGo project", sections: []domain.ContextSection{{Topic: "Routes", Content: "see routes.go"}}}
	got := NewBrainContext(b).Context(context.Background(), "/p", "where are routes?")
	for _, want := range []string{"Project memory", "# Onboarding", "### Routes", "see routes.go"} {
		if !strings.Contains(got, want) {
			t.Errorf("context lacks %q:\n%s", want, got)
		}
	}
	if b.asked.ProjectPath != "/p" || b.asked.Question != "where are routes?" || b.asked.MaxTokens <= 0 {
		t.Errorf("focused query = %+v", b.asked)
	}
}

func TestBrainContext_SurvivesAFailingBrainAndSkipsTheQuestionWhenThereIsNone(t *testing.T) {
	if got := NewBrainContext(&brainStub{onboardErr: errors.New("x"), focusedErr: errors.New("y")}).Context(context.Background(), "/p", "q"); got != "" {
		t.Errorf("a brain that fails gives no context, got %q", got)
	}
	b := &brainStub{onboarding: "O"}
	got := NewBrainContext(b).Context(context.Background(), "/p", "")
	if !strings.Contains(got, "O") || b.asked.ProjectPath != "" {
		t.Errorf("without a question only the onboarding is used: %q, asked %+v", got, b.asked)
	}
	if got := NewBrainContext(nil).Context(context.Background(), "/p", "q"); got != "" {
		t.Errorf("without a brain: %q", got)
	}
}
