package gateway

import (
	"context"
	"fmt"
	"strings"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// Token budgets of the two parts of the project memory a request gets.
const (
	onboardingTokens = 600
	focusedTokens    = 800
)

// BrainContext is a ContextProvider that reads a project's memory from the knowledge service: the
// onboarding summary (stack, active plan, rules) and the entries that answer the question.
type BrainContext struct {
	brain ports.BrainService
}

// NewBrainContext returns a ContextProvider over brain; a nil brain gives no context.
func NewBrainContext(brain ports.BrainService) *BrainContext { return &BrainContext{brain: brain} }

// Context implements ContextProvider. A failing brain is a missing memory, not a failed request: what
// could be read is returned, and "" when nothing could.
func (b *BrainContext) Context(ctx context.Context, projectPath, question string) string {
	if b.brain == nil {
		return ""
	}
	var parts []string
	if onboarding, err := b.brain.GetOnboardingContext(ctx, projectPath, onboardingTokens); err == nil && strings.TrimSpace(onboarding) != "" {
		parts = append(parts, onboarding)
	}
	if strings.TrimSpace(question) != "" {
		resp, err := b.brain.GetFocusedContext(ctx, domain.ContextQuery{ProjectPath: projectPath, Question: question, MaxTokens: focusedTokens})
		if err == nil {
			for _, s := range resp.Sections {
				parts = append(parts, fmt.Sprintf("### %s\n%s", s.Topic, s.Content))
			}
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "Project memory (from Nexus; use it, do not repeat it back):\n\n" + strings.Join(parts, "\n\n")
}
