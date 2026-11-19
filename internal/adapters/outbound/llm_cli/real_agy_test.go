//go:build realagy

package llm_cli

import (
	"context"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

// TestRealAgy runs the installed, signed-in `agy` once (a few tokens of the Antigravity plan):
//
//	go test -tags realagy -run TestRealAgy ./internal/adapters/outbound/llm_cli/
//
// The configured model becomes --model; the plan counts its allowance per
// model, so a model whose allowance is used up fails with a typed quota error while others answer.
func TestRealAgy(t *testing.T) {
	a, err := NewAdapter("Antigravity CLI", "agy", "gemini-3.8-flash-low")
	if err != nil {
		t.Fatal(err)
	}
	out, err := a.Chat(context.Background(), []domain.Message{{Role: domain.RoleUser, Content: "Reply with exactly: pong"}})
	if pe, ok := domain.AsProviderError(err); ok {
		t.Skipf("agy answered with a typed failure (%s, reset %s): %v", pe.Kind, pe.ResetAt, err)
	}
	if err != nil {
		t.Fatalf("real agy: %v", err)
	}
	if !strings.Contains(strings.ToLower(out), "pong") {
		t.Errorf("out=%q", out)
	}
}
