//go:build !windows

package llm_cli

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// Output recorded from agy 1.4.12 (brew install --cask antigravity-cli) when the plan's allowance was
// used up: exit code 3, nothing on stdout, and on stderr a line and a JSON object.
const agyQuotaStderr = `error: Individual quota reached. Please upgrade your subscription to increase your limits. Resets in 43h54m17s.
AGY_ERROR: {"short_error":"RESOURCE_EXHAUSTED (code 429): Individual quota reached. Please upgrade your subscription to increase your limits. Resets in 43h54m17s.","status":"RESOURCE_EXHAUSTED","error_code":429,"code_kind":"http","retryable":true,"error_id":"1f42b3f8-1713-45bd-8beb-86e286cacbf8-1"}`

func TestAgyDriver_PromptGoesOnStdinWithoutDashPAndTheConfiguredModel(t *testing.T) {
	record := filepath.Join(t.TempDir(), "record")
	cli := fakeCLI(t, "agy", `echo "$*" > `+record+`; cat`)
	// A config with a valueless -p / --print would swallow the next flag; the prompt is on stdin.
	a, err := NewAdapter("Antigravity CLI", cli+" -p", "gemini-3.8-flash-low")
	if err != nil {
		t.Fatal(err)
	}
	out, err := a.Chat(context.Background(), []domain.Message{{Role: domain.RoleUser, Content: "hello"}})
	if err != nil || !strings.Contains(out, "hello") {
		t.Fatalf("out=%q err=%v", out, err)
	}
	args := recordedArgs(t, record)
	if slices.Contains(args, "-p") || slices.Contains(args, "--print") || slices.Contains(args, "--prompt") {
		t.Errorf("a valueless print flag must be dropped: %v", args)
	}
	if !slices.Contains(args, "--model") || !slices.Contains(args, "gemini-3.8-flash-low") {
		t.Errorf("the configured model is passed: %v", args)
	}

	// A model already on the command line wins; no model configured means no flag.
	b, _ := NewAdapter("Antigravity CLI", cli+" --model=claude-sonnet-5-5-low", "other")
	if _, err := b.GenerateCode(context.Background(), "x"); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(recordedArgs(t, record), "--model") {
		t.Error("the model in the command line wins")
	}
	c, _ := NewAdapter("Antigravity CLI", cli, "")
	if _, err := c.GenerateCode(context.Background(), "x"); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(recordedArgs(t, record), "--model") {
		t.Error("no configured model: no --model flag")
	}
}

func TestAgyDriver_ModelsComeFromTheCLI(t *testing.T) {
	cli := fakeCLI(t, "agy", `if [ "$1" = models ]; then
echo "Fetching available models..."
printf 'gemini-3.8-flash-low\tGemini 3.8 Flash (Low)\n'
printf 'claude-sonnet-5-5-low\tClaude Sonnet 5.5 (Low)\n'
exit 0
fi
cat`)
	a, _ := NewAdapter("Antigravity CLI", cli, "")
	models, err := a.GetAvailableModels(context.Background())
	want := []string{"gemini-3.8-flash-low", "claude-sonnet-5-5-low"}
	if err != nil || !slices.Equal(models, want) {
		t.Fatalf("models = %v, %v; want %v", models, err, want)
	}

	broken := fakeCLI(t, "agy", `exit 1`)
	b, _ := NewAdapter("Antigravity CLI", broken, "")
	if models, err := b.GetAvailableModels(context.Background()); err != nil || !slices.Equal(models, []string{"default"}) {
		t.Errorf("a failing models command falls back to default: %v, %v", models, err)
	}
}

func TestAgyDriver_FailuresAreTypedAndTheResetTimeIsRead(t *testing.T) {
	ctx := context.Background()
	t.Run("quota with the reset time from the message", func(t *testing.T) {
		cli := fakeCLI(t, "agy", "cat >/dev/null; cat >&2 <<'EOF'\n"+agyQuotaStderr+"\nEOF\nexit 3")
		a, _ := NewAdapter("Antigravity CLI", cli, "")
		_, err := a.GenerateCode(ctx, "x")
		pe, ok := domain.AsProviderError(err)
		if !ok || pe.Kind != domain.ErrKindQuotaExhausted || pe.Status != 429 {
			t.Fatalf("want a quota error, got %v", err)
		}
		want := time.Now().Add(43*time.Hour + 54*time.Minute + 17*time.Second)
		if d := pe.ResetAt.Sub(want); d < -time.Minute || d > time.Minute {
			t.Errorf("ResetAt = %v, want about %v", pe.ResetAt, want)
		}
		if until := pe.CooldownUntil(time.Now()); !until.Equal(pe.ResetAt) {
			t.Errorf("the cooldown follows the reset time: %v", until)
		}
	})
	for name, tc := range map[string]struct {
		stderr string
		kind   domain.ProviderErrorKind
	}{
		"auth":       {`error: not signed in` + "\n" + `AGY_ERROR: {"short_error":"UNAUTHENTICATED (code 401)","status":"UNAUTHENTICATED","error_code":401}`, domain.ErrKindAuth},
		"overloaded": {`AGY_ERROR: {"short_error":"UNAVAILABLE (code 503)","status":"UNAVAILABLE","error_code":503,"retryable":true}`, domain.ErrKindOverloaded},
		"rate":       {`AGY_ERROR: {"short_error":"too many requests","status":"RATE_LIMITED","error_code":429}`, domain.ErrKindRateLimited},
	} {
		t.Run(name, func(t *testing.T) {
			cli := fakeCLI(t, "agy", "cat >/dev/null; cat >&2 <<'EOF'\n"+tc.stderr+"\nEOF\nexit 3")
			a, _ := NewAdapter("Antigravity CLI", cli, "")
			_, err := a.GenerateCode(ctx, "x")
			if pe, ok := domain.AsProviderError(err); !ok || pe.Kind != tc.kind {
				t.Errorf("want %v, got %v", tc.kind, err)
			}
		})
	}
	t.Run("anything else stays an ordinary error", func(t *testing.T) {
		cli := fakeCLI(t, "agy", `cat >/dev/null; echo "error: the model refused" >&2; exit 1`)
		a, _ := NewAdapter("Antigravity CLI", cli, "")
		_, err := a.GenerateCode(ctx, "x")
		if err == nil {
			t.Fatal("must fail")
		}
		if _, typed := domain.AsProviderError(err); typed {
			t.Errorf("not a quota or auth failure: %v", err)
		}
	})
}
