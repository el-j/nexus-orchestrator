package activity_claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// historyEntry represents a single line in ~/.claude/history.jsonl.
type historyEntry struct {
	Display   string `json:"display"`
	Timestamp int64  `json:"timestamp"` // unix milliseconds
	Project   string `json:"project"`
	SessionID string `json:"sessionId"`
}

// ClaudeHistoryReader reads ~/.claude/history.jsonl (global Claude CLI command log).
// Each line format:
// {"display":"<command>","timestamp":<unix_ms>,"project":"/path","sessionId":"<id>"}
type ClaudeHistoryReader struct {
	historyPath string
	lastPos     int64
	mu          sync.Mutex
}

// NewClaudeHistoryReader returns a reader pointing at the default history file.
func NewClaudeHistoryReader() *ClaudeHistoryReader {
	home, _ := os.UserHomeDir()
	return &ClaudeHistoryReader{
		historyPath: filepath.Join(home, ".claude", "history.jsonl"),
	}
}

// NewClaudeHistoryReaderAt returns a reader that reads from the given path.
// Intended for testing.
func NewClaudeHistoryReaderAt(path string) *ClaudeHistoryReader {
	return &ClaudeHistoryReader{historyPath: path}
}

// SourceName satisfies ports.ActivityReader.
func (r *ClaudeHistoryReader) SourceName() string {
	return "claude-history"
}

// ReadActivities returns new activities appended to history.jsonl since the
// given timestamp. Returns (nil, nil) when the file does not exist.
func (r *ClaudeHistoryReader) ReadActivities(ctx context.Context, since time.Time) ([]domain.AIActivity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	f, err := os.Open(r.historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("activity_claude: open history: %w", err)
	}
	defer f.Close()

	if _, err := f.Seek(r.lastPos, io.SeekStart); err != nil {
		return nil, fmt.Errorf("activity_claude: seek history: %w", err)
	}

	sinceMs := since.UnixMilli()
	var activities []domain.AIActivity

	pos := r.lastPos
	reader := bufio.NewReader(f)

	for {
		if ctx.Err() != nil {
			return activities, ctx.Err()
		}

		line, err := reader.ReadString('\n')
		if err == io.EOF {
			// Do not process a partial line; do not advance lastPos.
			break
		}
		if err != nil {
			return nil, fmt.Errorf("activity_claude: read history: %w", err)
		}

		// Only advance position for complete lines (those ending with '\n').
		pos += int64(len(line))

		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			continue
		}

		var entry historyEntry
		if err := json.Unmarshal([]byte(trimmed), &entry); err != nil {
			// Skip malformed lines.
			continue
		}

		if entry.Timestamp < sinceMs {
			continue
		}

		activities = append(activities, domain.AIActivity{
			ID:           "claude-history-" + entry.SessionID + "-" + strconv.FormatInt(entry.Timestamp, 10),
			SessionID:    entry.SessionID,
			AgentName:    "claude",
			ActivityType: domain.ActivityTypeMessage,
			Summary:      truncateSummary(entry.Display, 80),
			ProjectPath:  entry.Project,
			Timestamp:    time.UnixMilli(entry.Timestamp),
			Metadata:     map[string]string{"source": "cli-history"},
		})
	}

	r.lastPos = pos
	return activities, nil
}

// truncateSummary returns s truncated to at most max runes.
func truncateSummary(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
