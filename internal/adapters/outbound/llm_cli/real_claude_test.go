//go:build realclaude

package llm_cli

import (
	"context"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

// TestRealClaude runs the installed, logged-in `claude` once (a few tokens of your subscription):
//
//	go test -tags realclaude -run TestRealClaude ./internal/adapters/outbound/llm_cli/
func TestRealClaude(t *testing.T) {
	a, err := NewAdapter("Claude (subscription)", "claude -p --output-format stream-json --model haiku", "haiku")
	if err != nil {
		t.Fatal(err)
	}
	var chunks int
	out, err := a.ChatStream(context.Background(), []domain.Message{{Role: domain.RoleUser, Content: "Reply with exactly: ok"}}, func(string) { chunks++ })
	if err != nil {
		t.Fatalf("real claude: %v", err)
	}
	if !strings.Contains(strings.ToLower(out), "ok") || chunks == 0 {
		t.Errorf("out=%q chunks=%d", out, chunks)
	}
	u, ok := a.LastUsage()
	t.Logf("usage: %+v ok=%v", u, ok)
	q, qok := a.LastQuota()
	t.Logf("quota: %+v ok=%v", q, qok)
	if !ok || u.OutputTokens == 0 {
		t.Error("no usage reported")
	}
}
