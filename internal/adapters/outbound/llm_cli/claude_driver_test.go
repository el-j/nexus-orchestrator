//go:build !windows

package llm_cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// fixture is a real `claude -p --output-format stream-json` run (claude 2.1.301, trimmed of
// session data); see testdata/claude-stream.jsonl.
func fixture(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs("testdata/claude-stream.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// streamCLI returns a fake claude that records its arguments and the ANTHROPIC_* environment into
// record, then prints the file at stdoutFile.
func streamCLI(t *testing.T, stdoutFile, record string) string {
	t.Helper()
	return fakeCLI(t, "claude", `echo "$*" > `+record+`; env | grep '^ANTHROPIC_' >> `+record+`; cat `+stdoutFile)
}

const streamCmd = " -p --output-format stream-json --model haiku"

func TestClaudeStream_RealRunGivesTextChunksUsageAndQuota(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-must-not-reach-the-child")
	record := filepath.Join(t.TempDir(), "record")
	a, err := NewAdapter("Claude", streamCLI(t, fixture(t), record)+streamCmd, "haiku")
	if err != nil {
		t.Fatal(err)
	}
	var chunks []string
	out, err := a.ChatStream(context.Background(), []domain.Message{{Role: domain.RoleUser, Content: "count"}}, func(s string) { chunks = append(chunks, s) })
	if err != nil {
		t.Fatal(err)
	}
	if out != "1, 2, 3" || strings.Join(chunks, "|") != "1|, 2, 3" {
		t.Errorf("out=%q chunks=%q", out, chunks)
	}
	u, ok := a.LastUsage()
	if !ok || u.Model != "claude-haiku-5-5" || u.InputTokens != 2 || u.OutputTokens == 0 || u.CachedTokens < 10000 || u.CostUSD <= 0 {
		t.Errorf("usage: %+v %v", u, ok)
	}
	q, ok := a.LastQuota()
	if !ok || q.Window != "seven_day" || q.Status != "allowed_warning" || q.Utilization < 0.5 || q.ResetsAt.Before(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("quota: %+v %v", q, ok)
	}
	rec, _ := os.ReadFile(record)
	for _, want := range []string{"--include-partial-messages", "--verbose", "--tools", "--no-session-persistence", "--disable-slash-commands", "--setting-sources", "--strict-mcp-config", "--output-format stream-json", "--model haiku"} {
		if !strings.Contains(string(rec), want) {
			t.Errorf("arguments lack %q: %s", want, rec)
		}
	}
	if strings.Contains(string(rec), "ANTHROPIC_API_KEY") {
		t.Errorf("the API key must not reach the child (it would bill the API, not the subscription): %s", rec)
	}
}

func TestClaudeStream_DoesNotRepeatFlagsTheCommandAlreadyHas(t *testing.T) {
	record := filepath.Join(t.TempDir(), "record")
	cmd := streamCLI(t, fixture(t), record) + " -p --output-format stream-json --include-partial-messages --verbose --tools Read --no-session-persistence --disable-slash-commands --setting-sources user --strict-mcp-config"
	a, err := NewAdapter("Claude", cmd, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.GenerateCode(context.Background(), "x"); err != nil {
		t.Fatal(err)
	}
	rec, _ := os.ReadFile(record)
	for _, flag := range []string{"--verbose", "--tools", "--no-session-persistence", "--disable-slash-commands", "--include-partial-messages", "--setting-sources", "--strict-mcp-config"} {
		if strings.Count(string(rec), flag) != 1 {
			t.Errorf("%s must appear once: %s", flag, rec)
		}
	}
}

func writeStream(t *testing.T, lines ...string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "out.jsonl")
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func runStream(t *testing.T, lines ...string) (string, error) {
	t.Helper()
	record := filepath.Join(t.TempDir(), "record")
	a, err := NewAdapter("Claude", streamCLI(t, writeStream(t, lines...), record)+streamCmd, "")
	if err != nil {
		t.Fatal(err)
	}
	return a.GenerateCode(context.Background(), "x")
}

// The error shapes below are SYNTHETIC: they follow the fields the real client prints for a success
// (rate_limit_event.rate_limit_info, result.is_error, api_error_status) with the values a refusal
// would carry. They have not been captured from a real refusal.
const (
	rejectedEvent = `{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","resetsAt":1792209600,"rateLimitType":"five_hour","utilization":1}}`
	weeklyRejects = `{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","resetsAt":1792209600,"rateLimitType":"seven_day","utilization":1}}`
	errorResult   = `{"type":"result","subtype":"success","is_error":true,"result":"You've hit your limit","api_error_status":429,"terminal_reason":"error"}`
)

func TestClaudeStream_ARejectedLimitIsATypedErrorWithTheResetTime(t *testing.T) {
	_, err := runStream(t, rejectedEvent, errorResult)
	pe, ok := domain.AsProviderError(err)
	if !ok {
		t.Fatalf("want a ProviderError, got %v", err)
	}
	if pe.Kind != domain.ErrKindRateLimited || pe.ResetAt.Unix() != 1792209600 || !strings.Contains(pe.Error(), "five_hour") {
		t.Errorf("five-hour window: %+v", pe)
	}
	_, err = runStream(t, weeklyRejects, errorResult)
	if pe, ok := domain.AsProviderError(err); !ok || pe.Kind != domain.ErrKindQuotaExhausted {
		t.Errorf("a weekly window is an exhausted quota: %v", err)
	}
}

func TestClaudeStream_ErrorsWithoutALimitKeepTheClientsText(t *testing.T) {
	_, err := runStream(t, `{"type":"result","subtype":"success","is_error":true,"result":"model not found","api_error_status":404}`)
	if err == nil || !strings.Contains(err.Error(), "model not found") {
		t.Fatalf("err = %v", err)
	}
	if _, ok := domain.AsProviderError(err); ok {
		t.Error("not a limit")
	}
	_, err = runStream(t, `{"type":"result","subtype":"success","is_error":true,"result":"busy","api_error_status":529}`)
	if pe, ok := domain.AsProviderError(err); !ok || pe.Kind != domain.ErrKindOverloaded {
		t.Errorf("529 is overload: %v", err)
	}
	_, err = runStream(t, `{"type":"result","subtype":"success","is_error":true,"result":"limit","api_error_status":429}`)
	if pe, ok := domain.AsProviderError(err); !ok || pe.Kind != domain.ErrKindRateLimited {
		t.Errorf("a bare 429 is a rate limit: %v", err)
	}
}

func TestClaudeStream_ToleratesNoiseAndPlainOutput(t *testing.T) {
	// Unknown event types, blank lines and a non-JSON line are skipped or passed through as text.
	out, err := runStream(t, ``, `{"type":"something_new","x":1}`, `{"type":"result","subtype":"success","is_error":false,"result":"fine"}`)
	if err != nil || out != "fine" {
		t.Errorf("out=%q err=%v", out, err)
	}
	out, err = runStream(t, `just text from an older client`)
	if err != nil || out != "just text from an older client" {
		t.Errorf("plain: out=%q err=%v", out, err)
	}
}

func TestClaudeStream_UsesStreamedTextWhenThereIsNoResultEvent(t *testing.T) {
	out, err := runStream(t, `{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"partial"}}}`)
	if err != nil || out != "partial" {
		t.Errorf("out=%q err=%v", out, err)
	}
	if _, err := runStream(t, `{"type":"system","subtype":"init"}`); err == nil || !strings.Contains(err.Error(), "printed nothing") {
		t.Errorf("an empty answer is an error: %v", err)
	}
}

func TestClaudeStream_NonZeroExitCarriesStderr(t *testing.T) {
	a, err := NewAdapter("Claude", fakeCLI(t, "claude", `echo "boom: not logged in" >&2; exit 3`)+streamCmd, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.GenerateCode(context.Background(), "x"); err == nil || !strings.Contains(err.Error(), "not logged in") {
		t.Errorf("err = %v", err)
	}
}

func TestClaudeStream_CancelStopsTheProcess(t *testing.T) {
	a, err := NewAdapter("Claude", fakeCLI(t, "claude", `sleep 30`)+streamCmd, "")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := a.GenerateCode(ctx, "x"); err == nil || time.Since(start) > 10*time.Second {
		t.Errorf("err=%v after %v", err, time.Since(start))
	}
}

func TestPlainModeIsUnchanged(t *testing.T) {
	// Without --output-format stream-json nothing about the invocation changes: no extra flags, no
	// usage, and the key stays in the environment as the user set it.
	record := filepath.Join(t.TempDir(), "record")
	a, err := NewAdapter("Claude", fakeCLI(t, "claude", `echo "$*" > `+record+`; echo hi`)+" -p --model sonnet", "")
	if err != nil {
		t.Fatal(err)
	}
	if out, err := a.GenerateCode(context.Background(), "x"); err != nil || out != "hi" {
		t.Fatalf("%q %v", out, err)
	}
	if rec, _ := os.ReadFile(record); strings.TrimSpace(string(rec)) != "-p --model sonnet" {
		t.Errorf("args changed: %q", rec)
	}
	if _, ok := a.LastUsage(); ok {
		t.Error("plain mode has no usage")
	}
	if _, ok := a.LastQuota(); ok {
		t.Error("plain mode has no quota")
	}
}

func TestIsClaudeStream(t *testing.T) {
	cases := []struct {
		exe  string
		args []string
		want bool
	}{
		{"claude", []string{"-p", "--output-format", "stream-json"}, true},
		{"/usr/bin/claude.exe", []string{"--output-format=stream-json"}, true},
		{"claude", []string{"-p", "--output-format", "json"}, false},
		{"claude", []string{"-p", "--output-format"}, false},
		{"claude", []string{"-p"}, false},
		{"aichat", []string{"--output-format", "stream-json"}, false},
	}
	for _, c := range cases {
		if got := isClaudeStream(c.exe, c.args); got != c.want {
			t.Errorf("%s %v = %v", c.exe, c.args, got)
		}
	}
}

func TestClaudeStream_ModelComesFromTheResultWhenThereIsNoInitEvent(t *testing.T) {
	record := filepath.Join(t.TempDir(), "record")
	line := `{"type":"result","subtype":"success","is_error":false,"result":"ok","modelUsage":{"claude-sonnet-5-5":{}},"usage":{"input_tokens":1,"output_tokens":2}}`
	a, err := NewAdapter("Claude", streamCLI(t, writeStream(t, line), record)+streamCmd, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.GenerateCode(context.Background(), "x"); err != nil {
		t.Fatal(err)
	}
	if u, ok := a.LastUsage(); !ok || u.Model != "claude-sonnet-5-5" || u.InputTokens != 1 || u.OutputTokens != 2 {
		t.Errorf("usage %+v", u)
	}
}

func TestClaudeStream_ACommandThatCannotStartOrExitsSilentlyIsAnError(t *testing.T) {
	a, err := NewAdapter("Claude", "/nonexistent/dir/claude"+streamCmd, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.GenerateCode(context.Background(), "x"); err == nil || !strings.Contains(err.Error(), "start") {
		t.Errorf("start failure: %v", err)
	}
	b, err := NewAdapter("Claude", fakeCLI(t, "claude", `exit 2`)+streamCmd, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.GenerateCode(context.Background(), "x"); err == nil || !strings.Contains(err.Error(), "exit status 2") {
		t.Errorf("silent failure: %v", err)
	}
}
