package httpapi_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/core/domain"
)

// TestLogHub_Write_ParsesLogEntry verifies that Write stores a log entry in the
// ring buffer with the trimmed message string.
func TestLogHub_Write_ParsesLogEntry(t *testing.T) {
	hub := httpapi.NewLogHubWithWriter(io.Discard)
	_, err := hub.Write([]byte("hello world\n"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	buf := hub.Buffer()
	if len(buf) != 1 {
		t.Fatalf("expected 1 entry in buffer, got %d", len(buf))
	}
	if buf[0].Message != "hello world" {
		t.Errorf("expected message %q, got %q", "hello world", buf[0].Message)
	}
}

// TestLogHub_Write_TeeesToStderr verifies that each Write call also sends the
// bytes to the configured writer (injected via NewLogHubWithWriter).
func TestLogHub_Write_TeeesToStderr(t *testing.T) {
	var buf bytes.Buffer
	hub := httpapi.NewLogHubWithWriter(&buf)
	_, err := hub.Write([]byte("tee message\n"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !strings.Contains(buf.String(), "tee message") {
		t.Errorf("expected 'tee message' in captured output, got %q", buf.String())
	}
}

// TestLogHub_Buffer_CapAt500 verifies that the ring buffer holds at most 500
// entries; writing 600 lines must result in exactly 500 buffered entries.
func TestLogHub_Buffer_CapAt500(t *testing.T) {
	hub := httpapi.NewLogHubWithWriter(io.Discard)
	for i := 0; i < 600; i++ {
		_, _ = hub.Write([]byte(fmt.Sprintf("line %d\n", i)))
	}
	buf := hub.Buffer()
	if len(buf) != 500 {
		t.Errorf("expected 500 entries (ring buffer cap), got %d", len(buf))
	}
}

// TestLogHub_Subscribe_ReceivesEntries verifies that a subscriber channel
// receives a new log entry within 1 second of it being written.
func TestLogHub_Subscribe_ReceivesEntries(t *testing.T) {
	hub := httpapi.NewLogHubWithWriter(io.Discard)
	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	_, _ = hub.Write([]byte("subscribed message\n"))

	select {
	case entry := <-ch:
		if !strings.Contains(entry.Message, "subscribed message") {
			t.Errorf("expected 'subscribed message' in entry, got %q", entry.Message)
		}
	case <-time.After(1 * time.Second):
		t.Error("did not receive log entry within 1 second")
	}
}

// TestLogHub_Concurrent_NoRace exercises 10 goroutines each writing 100 log
// lines concurrently. The test verifies no data races (via -race) and that
// the buffer contains at most 500 entries.
func TestLogHub_Concurrent_NoRace(t *testing.T) {
	hub := httpapi.NewLogHubWithWriter(io.Discard)
	var wg sync.WaitGroup
	const goroutines = 10
	const linesEach = 100
	for i := 0; i < goroutines; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < linesEach; j++ {
				_, _ = hub.Write([]byte(fmt.Sprintf("goroutine %d line %d\n", i, j)))
			}
		}()
	}
	wg.Wait()
	buf := hub.Buffer()
	if len(buf) == 0 {
		t.Error("expected non-empty buffer after concurrent writes")
	}
	if len(buf) > 500 {
		t.Errorf("expected at most 500 entries, got %d", len(buf))
	}
}

// TestLogHub_LevelDetection_Error verifies that a message containing "ERROR:"
// is classified as LogLevelError.
func TestLogHub_LevelDetection_Error(t *testing.T) {
	hub := httpapi.NewLogHubWithWriter(io.Discard)
	_, _ = hub.Write([]byte("ERROR: something bad\n"))
	buf := hub.Buffer()
	if len(buf) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(buf))
	}
	if buf[0].Level != domain.LogLevelError {
		t.Errorf("expected level %q, got %q", domain.LogLevelError, buf[0].Level)
	}
}

// TestLogHub_LevelDetection_Warn verifies that a message containing "WARN:"
// is classified as LogLevelWarn.
func TestLogHub_LevelDetection_Warn(t *testing.T) {
	hub := httpapi.NewLogHubWithWriter(io.Discard)
	_, _ = hub.Write([]byte("WARN: disk full\n"))
	buf := hub.Buffer()
	if len(buf) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(buf))
	}
	if buf[0].Level != domain.LogLevelWarn {
		t.Errorf("expected level %q, got %q", domain.LogLevelWarn, buf[0].Level)
	}
}

// TestLogHub_LevelDetection_Default verifies that a plain message with no
// recognized keyword prefix is classified as LogLevelInfo.
func TestLogHub_LevelDetection_Default(t *testing.T) {
	hub := httpapi.NewLogHubWithWriter(io.Discard)
	_, _ = hub.Write([]byte("normal message\n"))
	buf := hub.Buffer()
	if len(buf) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(buf))
	}
	if buf[0].Level != domain.LogLevelInfo {
		t.Errorf("expected level %q, got %q", domain.LogLevelInfo, buf[0].Level)
	}
}
