package llm_cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// The Antigravity CLI driver (`agy`, Google's headless agent client; brew install --cask
// antigravity-cli). Behaviour recorded from agy 1.4.12: the prompt goes on stdin, which makes it
// non-interactive; `-p`/`--print`/`--prompt` want the prompt text as their value and would swallow the
// next flag, so a bare one is dropped. `agy models` lists the model ids, one per line, id first. A
// failure exits non-zero (3 for a quota) and prints `error: <text>` and a line
// `AGY_ERROR: {"status":"RESOURCE_EXHAUSTED","error_code":429,"short_error":"... Resets in 43h54m17s."}`
// on stderr, from which the failure is typed and the reset time read.

func isAgy(exe string) bool {
	return strings.TrimSuffix(strings.ToLower(filepath.Base(exe)), ".exe") == "agy"
}

// withAgyFlags returns args without a valueless print flag and with --model <model> when a model is
// configured and the command line has none.
func withAgyFlags(args []string, model string) []string {
	out := make([]string, 0, len(args)+2)
	for _, a := range args {
		if a != "-p" && a != "--print" && a != "--prompt" {
			out = append(out, a)
		}
	}
	if model != "" && !hasModelFlag(out) {
		out = append(out, "--model", model)
	}
	return out
}

// agyModels asks `agy models` for the model ids, or returns "default" when it cannot.
func agyModels(ctx context.Context, exe string) []string {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, exe, "models").Output()
	if err != nil {
		return []string{"default"}
	}
	var ids []string
	for _, line := range strings.Split(string(out), "\n") {
		if id, _, found := strings.Cut(line, "\t"); found && strings.TrimSpace(id) != "" {
			ids = append(ids, strings.TrimSpace(id))
		}
	}
	if len(ids) == 0 {
		return []string{"default"}
	}
	return ids
}

var (
	agyErrorLine = regexp.MustCompile(`AGY_ERROR:\s*(\{.*\})`)
	agyResetIn   = regexp.MustCompile(`(?i)resets? in ((?:\d+h)?(?:\d+m)?(?:\d+s)?)`)
)

// agyError is the JSON agy prints after AGY_ERROR:.
type agyError struct {
	ShortError string `json:"short_error"`
	Status     string `json:"status"`
	ErrorCode  int    `json:"error_code"`
}

// classifyAgyFailure types an agy failure by its AGY_ERROR line (status and code), reads "Resets in
// 43h54m17s" into ResetAt, and falls back to the message text. It returns nil for an ordinary failure.
func classifyAgyFailure(provider, detail string, now time.Time) error {
	m := agyErrorLine.FindStringSubmatch(detail)
	if m == nil {
		return classifyCLIFailure(provider, detail)
	}
	var e agyError
	if json.Unmarshal([]byte(m[1]), &e) != nil {
		return classifyCLIFailure(provider, detail)
	}
	text := e.ShortError
	if text == "" {
		text = detail
	}
	pe := &domain.ProviderError{Provider: provider, Status: e.ErrorCode}
	status := strings.ToUpper(e.Status)
	switch {
	case status == "RESOURCE_EXHAUSTED" || strings.Contains(strings.ToLower(text), "quota"):
		pe.Kind, pe.Status = domain.ErrKindQuotaExhausted, 429
		pe.Msg = fmt.Sprintf("%s: rate limited (429): quota exhausted: %s", provider, text)
	case e.ErrorCode == 429 || status == "RATE_LIMITED":
		pe.Kind, pe.Status = domain.ErrKindRateLimited, 429
		pe.Msg = fmt.Sprintf("%s: rate limited (429): %s", provider, text)
	case e.ErrorCode == 401 || e.ErrorCode == 403 || status == "UNAUTHENTICATED" || status == "PERMISSION_DENIED":
		pe.Kind = domain.ErrKindAuth
		pe.Msg = fmt.Sprintf("%s: not authorised (%d): %s", provider, e.ErrorCode, text)
	case e.ErrorCode == 503 || e.ErrorCode == 529 || status == "UNAVAILABLE":
		pe.Kind = domain.ErrKindOverloaded
		pe.Msg = fmt.Sprintf("%s: overloaded (%d): %s", provider, e.ErrorCode, text)
	default:
		return nil
	}
	if r := agyResetIn.FindStringSubmatch(text); len(r) > 1 && r[1] != "" {
		if d, err := time.ParseDuration(r[1]); err == nil && d > 0 {
			pe.ResetAt = now.Add(d)
		}
	}
	return pe
}
