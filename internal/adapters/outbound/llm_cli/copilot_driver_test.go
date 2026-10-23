//go:build !windows

package llm_cli

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

// recordedArgs returns the arguments a fake CLI wrote to its record file.
func recordedArgs(t *testing.T, path string) []string {
	t.Helper()
	rec, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Fields(string(rec))
}

func TestCopilotDriver_PromptGoesOnStdinWithLeanFlagsAndNoBareDashP(t *testing.T) {
	record := filepath.Join(t.TempDir(), "record")
	cli := fakeCLI(t, "copilot", `echo "$*" > `+record+`; cat`)
	// An older config still says `-p`: the real CLI rejects it without a value, so it is dropped.
	a, err := NewAdapter("GitHub Copilot CLI", cli+" -p", "auto")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if a.ContextLimit(ctx) <= 0 {
		t.Error("copilot has a known context limit")
	}
	if models, err := a.GetAvailableModels(ctx); err != nil || !slices.Equal(models, []string{"auto"}) {
		t.Errorf("models = %v, %v", models, err)
	}
	var chunks []string
	out, err := a.ChatStream(ctx, []domain.Message{{Role: domain.RoleUser, Content: "hello"}}, func(s string) { chunks = append(chunks, s) })
	if err != nil || !strings.Contains(out, "hello") || len(chunks) == 0 {
		t.Fatalf("out=%q chunks=%v err=%v", out, chunks, err)
	}
	args := recordedArgs(t, record)
	for _, want := range []string{"-s", "--no-ask-user", "--no-color", "--model", "auto"} {
		if !slices.Contains(args, want) {
			t.Errorf("arguments lack %q: %v", want, args)
		}
	}
	if slices.Contains(args, "-p") || slices.Contains(args, "--prompt") {
		t.Errorf("a valueless -p fails in the real CLI, it must be dropped: %v", args)
	}
}

func TestCopilotDriver_DoesNotDuplicateFlagsAndOmitsTheModelWhenNoneIsConfigured(t *testing.T) {
	record := filepath.Join(t.TempDir(), "record")
	cli := fakeCLI(t, "copilot", `echo "$*" > `+record+`; echo ok`)
	b, err := NewAdapter("Copilot", cli+" -s --no-ask-user --no-color --model=claude-sonnet-4.5", "gpt-5")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.GenerateCode(context.Background(), "x"); err != nil {
		t.Fatal(err)
	}
	args := recordedArgs(t, record)
	for _, flag := range []string{"-s", "--no-ask-user", "--no-color"} {
		n := 0
		for _, a := range args {
			if a == flag {
				n++
			}
		}
		if n != 1 {
			t.Errorf("%s must appear once, got %d: %v", flag, n, args)
		}
	}
	if slices.Contains(args, "--model") {
		t.Errorf("the model in the command line wins over the configured one: %v", args)
	}

	none, _ := NewAdapter("Copilot", cli, "")
	if _, err := none.GenerateCode(context.Background(), "x"); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(recordedArgs(t, record), "--model") {
		t.Error("no configured model: no --model flag")
	}
}

func TestCopilotDriver_QuotaRateLimitAndAuthFailuresAreTypedAndOthersAreNot(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		stderr string
		kind   domain.ProviderErrorKind
	}{
		"quota":      {"You have exceeded your monthly quota of premium requests", domain.ErrKindQuotaExhausted},
		"rate limit": {"429 Too Many Requests: rate limit reached", domain.ErrKindRateLimited},
		"auth":       {"Not logged in: run copilot login", domain.ErrKindAuth},
	} {
		t.Run(name, func(t *testing.T) {
			cli := fakeCLI(t, "copilot", `echo "`+tc.stderr+`" >&2; exit 1`)
			a, _ := NewAdapter("Copilot", cli, "")
			_, err := a.GenerateCode(ctx, "x")
			if pe, ok := domain.AsProviderError(err); !ok || pe.Kind != tc.kind {
				t.Errorf("want %v, got %v", tc.kind, err)
			}
		})
	}

	// A number that merely contains 429, or an unrelated failure, is an ordinary error.
	cli := fakeCLI(t, "copilot", `echo "syntax error at line 4290" >&2; exit 1`)
	a, _ := NewAdapter("Copilot", cli, "")
	if _, err := a.GenerateCode(ctx, "x"); err == nil {
		t.Fatal("must fail")
	} else if _, typed := domain.AsProviderError(err); typed {
		t.Errorf("line 4290 is not a rate limit: %v", err)
	}

	// Only the Copilot driver classifies by message: another CLI that prints "quota" stays untyped.
	other := fakeCLI(t, "aichat", `echo "quota: disk quota exceeded" >&2; exit 1`)
	o, _ := NewAdapter("aichat", other, "")
	if _, err := o.GenerateCode(ctx, "x"); err == nil {
		t.Fatal("must fail")
	} else if _, typed := domain.AsProviderError(err); typed {
		t.Errorf("only Copilot failures are classified by their text: %v", err)
	}
}
