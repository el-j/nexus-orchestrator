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
	"nexus-orchestrator/internal/core/ports"
)

var (
	_ ports.LLMClient    = (*Adapter)(nil)
	_ ports.ChatStreamer = (*Adapter)(nil)
)

// fakeCLI writes an executable named name into a temp dir that runs body, and returns its path.
func fakeCLI(t *testing.T, name, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestChat_SendsTheConversationOnStdinAndReturnsStdout(t *testing.T) {
	cli := fakeCLI(t, "claude", `echo "args=$*"; cat`)
	a, err := NewAdapter("Claude (subscription)", cli+" -p --model sonnet", "sonnet")
	if err != nil {
		t.Fatal(err)
	}
	out, err := a.Chat(context.Background(), []domain.Message{
		{Role: domain.RoleSystem, Content: "be brief"},
		{Role: domain.RoleUser, Content: "hi"},
		{Role: domain.RoleAssistant, Content: "hello"},
		{Role: domain.RoleUser, Content: "again"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"args=-p --model sonnet", "be brief", "hi", "hello", "again"} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q lacks %q", out, want)
		}
	}
	if strings.Index(out, "hi") > strings.Index(out, "again") {
		t.Errorf("turns out of order: %q", out)
	}
	gen, err := a.GenerateCode(context.Background(), "only this")
	if err != nil || !strings.Contains(gen, "only this") {
		t.Errorf("GenerateCode = %q, %v", gen, err)
	}
}

func TestChatStream_DeliversOutputAsItIsWritten(t *testing.T) {
	cli := fakeCLI(t, "claude", `printf 'FIRST '; sleep 0.3; printf 'SECOND'`)
	a, _ := NewAdapter("c", cli, "")
	var pieces []string
	var firstAt, lastAt time.Time
	start := time.Now()
	out, err := a.ChatStream(context.Background(), []domain.Message{{Role: domain.RoleUser, Content: "x"}}, func(p string) {
		if firstAt.IsZero() {
			firstAt = time.Now()
		}
		lastAt = time.Now()
		pieces = append(pieces, p)
	})
	if err != nil || out != "FIRST SECOND" {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if firstAt.Sub(start) > 250*time.Millisecond || lastAt.Sub(firstAt) < 200*time.Millisecond {
		t.Errorf("not streamed: first after %v, last %v later; pieces %q", firstAt.Sub(start), lastAt.Sub(firstAt), pieces)
	}
}

func TestChatStream_KeepsMultibyteCharactersWhole(t *testing.T) {
	cli := fakeCLI(t, "claude", `printf 'caf\303'; sleep 0.1; printf '\251 \342\234\223'`)
	a, _ := NewAdapter("c", cli, "")
	var pieces []string
	out, err := a.ChatStream(context.Background(), []domain.Message{{Role: domain.RoleUser, Content: "x"}}, func(p string) { pieces = append(pieces, p) })
	if err != nil || out != "café ✓" {
		t.Fatalf("out=%q err=%v", out, err)
	}
	for _, p := range pieces {
		if !strings.Contains(strings.ToValidUTF8(p, "�"), p) || strings.Contains(p, "�") {
			t.Errorf("piece %q split a character", p)
		}
	}
}

func TestChat_FailuresCarryTheCLIsOwnMessage(t *testing.T) {
	cli := fakeCLI(t, "claude", `echo "Not logged in. Run claude /login" >&2; exit 3`)
	a, _ := NewAdapter("c", cli, "")
	_, err := a.Chat(context.Background(), []domain.Message{{Role: domain.RoleUser, Content: "x"}})
	if err == nil || !strings.Contains(err.Error(), "Not logged in") || !strings.Contains(err.Error(), "exit status 3") {
		t.Errorf("err = %v", err)
	}
	empty := fakeCLI(t, "claude", `true`)
	a, _ = NewAdapter("c", empty, "")
	if _, err := a.Chat(context.Background(), []domain.Message{{Role: domain.RoleUser, Content: "x"}}); err == nil {
		t.Error("an empty reply is an error")
	}
}

func TestCancelKillsTheProcess(t *testing.T) {
	cli := fakeCLI(t, "claude", `sleep 30`)
	a, _ := NewAdapter("c", cli, "")
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	done := make(chan error, 1)
	go func() {
		_, err := a.Chat(ctx, []domain.Message{{Role: domain.RoleUser, Content: "x"}})
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Error("want an error after cancel")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancel did not stop the CLI")
	}
}

func TestNewAdapter_OnlyKnownCLIsMayRun(t *testing.T) {
	for _, bad := range []string{"", "   ", "/bin/sh -c id", "rm -rf /", "curl http://x"} {
		if _, err := NewAdapter("x", bad, ""); err == nil {
			t.Errorf("%q must be refused", bad)
		}
	}
	t.Setenv("NEXUS_CLI_PROVIDERS", "mytool, other")
	if _, err := NewAdapter("x", "/opt/bin/mytool --flag", ""); err != nil {
		t.Errorf("an executable named in NEXUS_CLI_PROVIDERS is allowed: %v", err)
	}
	if _, err := NewAdapter("x", `claude "unterminated`, ""); err == nil {
		t.Error("an unparsable command line must be refused")
	}
}

func TestMetadata(t *testing.T) {
	cli := fakeCLI(t, "claude", `true`)
	a, _ := NewAdapter("Claude (subscription)", cli+" -p", "opus")
	ctx := context.Background()
	if a.ProviderName() != "Claude (subscription)" || a.ActiveModel() != "opus" || a.BaseURL() != cli+" -p" {
		t.Errorf("metadata: %q %q %q", a.ProviderName(), a.ActiveModel(), a.BaseURL())
	}
	if !a.Ping(ctx) {
		t.Error("an installed CLI pings")
	}
	if m, err := a.GetAvailableModels(ctx); err != nil || len(m) == 0 {
		t.Errorf("models = %v, %v", m, err)
	}
	if a.ContextLimit(ctx) <= 0 {
		t.Error("claude has a known context window")
	}
	missing, _ := NewAdapter("m", "/nonexistent/claude", "")
	if missing.Ping(ctx) {
		t.Error("a missing CLI must not ping")
	}
	other, _ := NewAdapter("o", fakeCLI(t, "aichat", `true`), "")
	if m, _ := other.GetAvailableModels(ctx); len(m) != 1 || m[0] != "default" {
		t.Errorf("unknown CLI models = %v", m)
	}
	if other.ContextLimit(ctx) != 0 {
		t.Error("an unknown CLI has no known context limit")
	}
}

func TestSplitCommand(t *testing.T) {
	got, err := splitCommand(`claude -p --append-system-prompt "be  brief" 'x y' ""`)
	want := []string{"claude", "-p", "--append-system-prompt", "be  brief", "x y", ""}
	if err != nil || strings.Join(got, "|") != strings.Join(want, "|") || len(got) != len(want) {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestChatStream_PassesOnABrokenFinalByteSequence(t *testing.T) {
	cli := fakeCLI(t, "claude", `printf 'ok\303'`)
	a, _ := NewAdapter("c", cli, "")
	out, err := a.ChatStream(context.Background(), []domain.Message{{Role: domain.RoleUser, Content: "x"}}, func(string) {})
	if err != nil || !strings.HasPrefix(out, "ok") {
		t.Errorf("out=%q err=%v", out, err)
	}
}
