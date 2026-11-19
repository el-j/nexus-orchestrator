//go:build !windows

package llm_cli

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// brokenReader hands out some text and then fails with an error that is not the end of the output.
type brokenReader struct{ sent bool }

func (b *brokenReader) Read(p []byte) (int, error) {
	if !b.sent {
		b.sent = true
		return copy(p, "partial"), nil
	}
	return 0, errors.New("connection reset")
}

func TestPumpReply_ReturnsWhatItReadTogetherWithAReadError(t *testing.T) {
	var chunks []string
	got, err := pumpReply(&brokenReader{}, func(s string) { chunks = append(chunks, s) })
	if got != "partial" || err == nil || !strings.Contains(err.Error(), "connection reset") {
		t.Errorf("reply=%q err=%v", got, err)
	}
	if len(chunks) != 1 || chunks[0] != "partial" {
		t.Errorf("chunks = %v", chunks)
	}
	if got, err := pumpReply(strings.NewReader("done"), func(string) {}); got != "done" || err != nil {
		t.Errorf("a clean end must not be an error: %q %v", got, err)
	}
}

func TestRun_ReportsACommandThatCannotStartAndFailuresWithoutStderr(t *testing.T) {
	ctx := context.Background()
	t.Run("the program does not exist", func(t *testing.T) {
		a, err := NewAdapter("Missing", "/no/such/folder/aichat", "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := a.GenerateCode(ctx, "x"); err == nil || !strings.Contains(err.Error(), "start") {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("stderr is empty, so what the command printed explains the failure", func(t *testing.T) {
		cli := fakeCLI(t, "aichat", `echo "model not found"; exit 2`)
		a, _ := NewAdapter("aichat", cli, "")
		if _, err := a.GenerateCode(ctx, "x"); err == nil || !strings.Contains(err.Error(), "model not found") {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("a failure that says nothing at all", func(t *testing.T) {
		cli := fakeCLI(t, "aichat", `exit 3`)
		a, _ := NewAdapter("aichat", cli, "")
		if _, err := a.GenerateCode(ctx, "x"); err == nil || !strings.Contains(err.Error(), "exit status 3") {
			t.Errorf("err = %v", err)
		}
	})
}

func TestClassifyAgyFailure_FallsBackWhenTheErrorLineIsBrokenOrNotAKnownKind(t *testing.T) {
	now := time.Now()
	// The JSON after AGY_ERROR is malformed: the plain-text rules still apply.
	err := classifyAgyFailure("agy", `exit status 3: error: quota reached
AGY_ERROR: {"short_error": RESOURCE}`, now)
	if pe, ok := domain.AsProviderError(err); err != nil && (!ok || pe.Kind != domain.ErrKindQuotaExhausted) {
		t.Errorf("a broken line must still read the text: %v", err)
	}
	// No short_error: the whole detail is the message.
	err = classifyAgyFailure("agy", `exit status 3: AGY_ERROR: {"status":"UNAVAILABLE","error_code":503}`, now)
	pe, ok := domain.AsProviderError(err)
	if !ok || pe.Kind != domain.ErrKindOverloaded || !strings.Contains(pe.Msg, "AGY_ERROR") {
		t.Errorf("overloaded without a short_error: %v", err)
	}
	// An error of a kind routing cannot act on is an ordinary failure.
	if err := classifyAgyFailure("agy", `exit status 3: AGY_ERROR: {"status":"INVALID_ARGUMENT","error_code":400,"short_error":"bad flag"}`, now); err != nil {
		t.Errorf("an unknown kind must not be typed: %v", err)
	}
}

func TestParseClaudeStream_ReturnsTheReadErrorOfABrokenStream(t *testing.T) {
	_, err := parseClaudeStream(&brokenReader{}, func(string) {})
	if err == nil || !strings.Contains(err.Error(), "connection reset") {
		t.Errorf("err = %v", err)
	}
}
