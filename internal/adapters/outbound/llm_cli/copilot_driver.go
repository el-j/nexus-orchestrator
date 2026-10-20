package llm_cli

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"nexus-orchestrator/internal/core/domain"
)

// The Copilot CLI driver. The prompt goes to `copilot` on stdin, which makes it non-interactive; the
// adapter adds `-s` (the answer only, no statistics), `--no-ask-user` (never wait for a question to be
// answered) and `--no-color`, and `--model <model>` when a model is configured. A bare `-p` /
// `--prompt` is dropped: the CLI wants the prompt text as its value, so it fails with "a value is
// required" when the prompt arrives on stdin (older configs have it). Tools that need approval stay
// denied, since nobody can approve them. Quota, rate-limit and authentication failures the CLI prints
// become typed domain.ProviderError values, so routing cools the provider down.

var copilotLeanFlags = []string{"-s", "--no-ask-user", "--no-color"}

func isCopilot(exe string) bool {
	return strings.TrimSuffix(strings.ToLower(filepath.Base(exe)), ".exe") == "copilot"
}

// withCopilotFlags returns args without a valueless -p/--prompt and with the lean flags and (when model
// is set) --model <model> unless the command line already has them.
func withCopilotFlags(args []string, model string) []string {
	out := make([]string, 0, len(args)+len(copilotLeanFlags)+2)
	for _, a := range args {
		if a != "-p" && a != "--prompt" {
			out = append(out, a)
		}
	}
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
		if a == "--model" || strings.HasPrefix(a, "--model=") {
			return true
		}
	}
	return false
}

// http429 matches the status as a word, so "line 4290" is not a rate limit.
var http429 = regexp.MustCompile(`\b429\b`)

// classifyCLIFailure turns a non-zero CLI exit message into a typed domain.ProviderError when it
// reports an exhausted quota, a rate limit or an authentication failure. It returns nil otherwise.
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
		http429.MatchString(low):
		return &domain.ProviderError{
			Provider: provider,
			Kind:     domain.ErrKindRateLimited,
			Status:   429,
			Msg:      fmt.Sprintf("%s: rate limited (429): %s", provider, detail),
		}
	case strings.Contains(low, "not logged in") || strings.Contains(low, "unauthorized") ||
		strings.Contains(low, "unauthenticated") || strings.Contains(low, "please authenticate") ||
		strings.Contains(low, "copilot login"):
		return &domain.ProviderError{
			Provider: provider,
			Kind:     domain.ErrKindAuth,
			Status:   401,
			Msg:      fmt.Sprintf("%s: not authorised (401): %s", provider, detail),
		}
	}
	return nil
}
