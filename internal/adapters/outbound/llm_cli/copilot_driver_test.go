//go:build !windows

package llm_cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

func TestCopilotDriver_AppendsLeanFlagsModelAndClassifiesLimitErrors(t *testing.T) {
	record := filepath.Join(t.TempDir(), "record")
	cli := fakeCLI(t, "copilot", `echo "$*" > `+record+`; cat`)
	a, err := NewAdapter("GitHub Copilot CLI", cli+" -p", "gpt-4o")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if a.ContextLimit(ctx) <= 0 {
		t.Error("copilot has a known context limit")
	}
	models, err := a.GetAvailableModels(ctx)
	if err != nil || len(models) == 0 {
		t.Errorf("models = %v, %v", models, err)
	}
	var chunks []string
	out, err := a.ChatStream(ctx, []domain.Message{{Role: domain.RoleUser, Content: "hello"}}, func(s string) { chunks = append(chunks, s) })
	if err != nil || !strings.Contains(out, "hello") || len(chunks) == 0 {
		t.Fatalf("out=%q chunks=%v err=%v", out, chunks, err)
	}
	rec, _ := os.ReadFile(record)
	for _, want := range []string{"-p", "-s", "--no-ask-user", "--model gpt-4o"} {
		if !strings.Contains(string(rec), want) {
			t.Errorf("arguments lack %q: %s", want, rec)
		}
	}

	// Does not duplicate flags already in the command line.
	record2 := filepath.Join(t.TempDir(), "record2")
	cli2 := fakeCLI(t, "copilot", `echo "$*" > `+record2+`; echo ok`)
	b, err := NewAdapter("Copilot", cli2+" -p -s --no-ask-user --model claude-sonnet-4.5", "gpt-4o")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.GenerateCode(ctx, "x"); err != nil {
		t.Fatal(err)
	}
	rec2, _ := os.ReadFile(record2)
	fields := strings.Fields(string(rec2))
	for _, flag := range []string{"-s", "--no-ask-user", "--model"} {
		count := 0
		for _, f := range fields {
			if f == flag {
				count++
			}
		}
		if count != 1 {
			t.Errorf("%s must appear once: %s", flag, rec2)
		}
	}

	// Quota and rate-limit failures on stderr/stdout become typed ProviderErrors.
	quotaCLI := fakeCLI(t, "copilot", `echo "You have exceeded your monthly quota of premium requests" >&2; exit 1`)
	qAdapter, _ := NewAdapter("Copilot", quotaCLI+" -p", "")
	_, err = qAdapter.GenerateCode(ctx, "x")
	if pe, ok := domain.AsProviderError(err); !ok || pe.Kind != domain.ErrKindQuotaExhausted {
		t.Errorf("quota error must be ErrKindQuotaExhausted: %v", err)
	}

	rateCLI := fakeCLI(t, "copilot", `echo "429 Too Many Requests: rate limit reached" >&2; exit 1`)
	rAdapter, _ := NewAdapter("Copilot", rateCLI+" -p", "")
	_, err = rAdapter.GenerateCode(ctx, "x")
	if pe, ok := domain.AsProviderError(err); !ok || pe.Kind != domain.ErrKindRateLimited {
		t.Errorf("rate-limit error must be ErrKindRateLimited: %v", err)
	}

	authCLI := fakeCLI(t, "copilot", `echo "Not logged in: please authenticate with github" >&2; exit 1`)
	aAdapter, _ := NewAdapter("Copilot", authCLI+" -p", "")
	_, err = aAdapter.GenerateCode(ctx, "x")
	if pe, ok := domain.AsProviderError(err); !ok || pe.Kind != domain.ErrKindAuth {
		t.Errorf("auth error must be ErrKindAuth: %v", err)
	}
}
