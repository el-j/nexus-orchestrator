package llm_cli

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"nexus-orchestrator/internal/core/domain"
)

// The Copilot CLI driver (`copilot -p`). When run as `copilot`, the adapter ensures `-p`
// (non-interactive prompt), `-s` (answer only, no spinner/chrome), `--no-ask-user` and
// `--model <model>` (when configured and not already in the command line), and turns quota,
// rate-limit and authentication errors printed by the CLI into typed domain.ProviderError values.

var copilotLeanFlags = []string{"-p", "-s", "--no-ask-user"}

func isCopilot(exe string) bool {
	return strings.TrimSuffix(strings.ToLower(filepath.Base(exe)), ".exe") == "copilot"
}

// withCopilotFlags appends -p, -s, --no-ask-user and (when model is set) --model <model>
// if the command line does not already include them.
func withCopilotFlags(args []string, model string) []string {
	out := slices.Clone(args)
	for _, flag := range copilotLeanFlags {
		if !slices.Contains(out, flag) {
			out = append(out, flag)
		}
	}
	if model != "" && !hasModelFlag(out) {
		out = append(out, "--model", model)
	}
	return out
}

func hasModelFlag(args []string) bool {
	for _, a := range args {
		if a == "--model" || a == "-m" || strings.HasPrefix(a, "--model=") {
			return true
		}
	}
	return false
}

// classifyCLIFailure turns a non-zero CLI exit message into a typed domain.ProviderError when it
// reports an exhausted quota, rate limit, overload, or authentication failure.
func classifyCLIFailure(provider, detail string) error {
	low := strings.ToLower(detail)
	switch {
	case strings.Contains(low, "quota") || strings.Contains(low, "resource_exhausted") ||
		strings.Contains(low, "billing") || strings.Contains(low, "premium request"):
		return &domain.ProviderError{
			Provider: provider,
			Kind:     domain.ErrKindQuotaExhausted,
			Status:   429,
			Msg:      fmt.Sprintf("%s: rate limited (429): quota exhausted: %s", provider, detail),
		}
	case strings.Contains(low, "rate limit") || strings.Contains(low, "too many requests") ||
		strings.Contains(low, "429"):
		return &domain.ProviderError{
			Provider: provider,
			Kind:     domain.ErrKindRateLimited,
			Status:   429,
			Msg:      fmt.Sprintf("%s: rate limited (429): %s", provider, detail),
		}
	case strings.Contains(low, "not logged in") || strings.Contains(low, "authenticate") ||
		strings.Contains(low, "unauthorized") || strings.Contains(low, "unauthenticated"):
		return &domain.ProviderError{
			Provider: provider,
			Kind:     domain.ErrKindAuth,
			Status:   401,
			Msg:      fmt.Sprintf("%s: not authorised (401): %s", provider, detail),
		}
	}
	return nil
}
