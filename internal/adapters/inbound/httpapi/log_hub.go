package httpapi

import (
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

const logRingSize = 500

// LogHub implements io.Writer. It parses log lines into LogEntry values,
// maintains a ring buffer, broadcasts entries via a channel, and tees to stderr.
type LogHub struct {
	mu     sync.Mutex
	ring   []domain.LogEntry
	head   int
	size   int
	subs   []chan domain.LogEntry
	stderr io.Writer
}

// NewLogHub creates a LogHub that tees to os.Stderr.
func NewLogHub() *LogHub {
	return &LogHub{
		ring:   make([]domain.LogEntry, logRingSize),
		stderr: os.Stderr,
	}
}

// NewLogHubWithWriter creates a LogHub that tees to the given writer.
// Intended for testing.
func NewLogHubWithWriter(w io.Writer) *LogHub {
	return &LogHub{
		ring:   make([]domain.LogEntry, logRingSize),
		stderr: w,
	}
}

// Write implements io.Writer — called by log.Printf.
func (h *LogHub) Write(p []byte) (int, error) {
	// Tee to stderr first
	_, _ = h.stderr.Write(p)

	msg := strings.TrimRight(string(p), "\n")
	entry := domain.LogEntry{
		Timestamp: time.Now(),
		Level:     detectLevel(msg),
		Source:    "daemon",
		Message:   msg,
	}

	h.mu.Lock()
	h.ring[h.head%logRingSize] = entry
	h.head++
	if h.size < logRingSize {
		h.size++
	}
	subs := make([]chan domain.LogEntry, len(h.subs))
	copy(subs, h.subs)
	h.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- entry:
		default:
		}
	}
	return len(p), nil
}

// Buffer returns a snapshot of the ring buffer (oldest first).
func (h *LogHub) Buffer() []domain.LogEntry {
	h.mu.Lock()
	defer h.mu.Unlock()

	out := make([]domain.LogEntry, h.size)
	if h.size < logRingSize {
		copy(out, h.ring[:h.size])
	} else {
		start := h.head % logRingSize
		copy(out, h.ring[start:])
		copy(out[logRingSize-start:], h.ring[:start])
	}
	return out
}

// Subscribe returns a channel that receives new log entries. Call Unsubscribe when done.
func (h *LogHub) Subscribe() chan domain.LogEntry {
	ch := make(chan domain.LogEntry, 64)
	h.mu.Lock()
	h.subs = append(h.subs, ch)
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes and closes the subscription channel.
func (h *LogHub) Unsubscribe(ch chan domain.LogEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, s := range h.subs {
		if s == ch {
			h.subs = append(h.subs[:i], h.subs[i+1:]...)
			close(ch)
			return
		}
	}
}

func detectLevel(msg string) domain.LogLevel {
	upper := strings.ToUpper(msg)
	switch {
	case strings.Contains(upper, "ERROR:") || strings.Contains(upper, "[ERROR]"):
		return domain.LogLevelError
	case strings.Contains(upper, "WARN:") || strings.Contains(upper, "[WARN]"):
		return domain.LogLevelWarn
	case strings.Contains(upper, "DEBUG:") || strings.Contains(upper, "[DEBUG]"):
		return domain.LogLevelDebug
	default:
		return domain.LogLevelInfo
	}
}
