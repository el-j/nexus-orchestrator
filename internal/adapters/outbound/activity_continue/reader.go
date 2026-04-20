// Package activity_continue reads ~/.continue/sessions/ to observe Continue IDE activity.
package activity_continue

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// sessionIndex is the top-level structure of ~/.continue/sessions/sessions.json.
type sessionIndex struct {
	Sessions []sessionIndexEntry `json:"sessions"`
}

type sessionIndexEntry struct {
	SessionID          string `json:"sessionId"`
	Title              string `json:"title"`
	DateCreated        string `json:"dateCreated"`
	WorkspaceDirectory string `json:"workspaceDirectory"`
}

// sessionFile is the structure of ~/.continue/sessions/<sessionId>.json.
type sessionFile struct {
	SessionID string           `json:"sessionId"`
	Title     string           `json:"title"`
	ModelID   string           `json:"model,omitempty"`
	History   []historyMessage `json:"history"`
}

type historyMessage struct {
	Role      string          `json:"role"`
	Content   json.RawMessage `json:"content"` // string or structured content blocks
	Timestamp int64           `json:"timestamp"`
	Model     string          `json:"model,omitempty"`
}

// ContinueSessionReader reads ~/.continue/sessions/ to observe Continue IDE activity.
type ContinueSessionReader struct {
	sessionsDir string
	seenMTimes  map[string]int64 // sessionId -> last-seen modification time (nanoseconds)
	mu          sync.Mutex
}

// NewContinueSessionReader creates a reader that defaults to ~/.continue/sessions.
func NewContinueSessionReader() *ContinueSessionReader {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return &ContinueSessionReader{
		sessionsDir: filepath.Join(home, ".continue", "sessions"),
		seenMTimes:  make(map[string]int64),
	}
}

// NewContinueSessionReaderWithDir creates a reader pointing at an explicit sessions directory.
// Primarily intended for testing.
func NewContinueSessionReaderWithDir(dir string) *ContinueSessionReader {
	return &ContinueSessionReader{
		sessionsDir: dir,
		seenMTimes:  make(map[string]int64),
	}
}

// SourceName identifies this reader.
func (r *ContinueSessionReader) SourceName() string {
	return "continue-sessions"
}

// ReadActivities returns one AIActivity per Continue session modified since the given timestamp.
func (r *ContinueSessionReader) ReadActivities(ctx context.Context, since time.Time) ([]domain.AIActivity, error) {
	indexPath := filepath.Join(r.sessionsDir, "sessions.json")

	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("activity_continue: read sessions index: %w", err)
	}

	var idx sessionIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		// Unrecognised format — return empty, not an error.
		return nil, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	var activities []domain.AIActivity

	for _, entry := range idx.Sessions {
		if ctx.Err() != nil {
			return activities, ctx.Err()
		}

		sessionPath := filepath.Join(r.sessionsDir, entry.SessionID+".json")

		info, err := os.Stat(sessionPath)
		if err != nil {
			continue // session file not present yet
		}

		modNano := info.ModTime().UnixNano()
		if !info.ModTime().After(since) {
			continue
		}

		activity, err := r.buildActivity(entry, sessionPath, info.ModTime())
		if err != nil {
			// Unreadable / unexpected format — skip silently.
			continue
		}

		if activity == nil {
			continue
		}

		// Only include if newest message is after since.
		if !activity.Timestamp.After(since) {
			continue
		}

		activities = append(activities, *activity)
		r.seenMTimes[entry.SessionID] = modNano
	}

	return activities, nil
}

// buildActivity parses a single session file and returns the corresponding AIActivity.
// Returns nil if the history is empty or no parseable timestamp is available.
func (r *ContinueSessionReader) buildActivity(entry sessionIndexEntry, sessionPath string, fileMod time.Time) (*domain.AIActivity, error) {
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("activity_continue: read session file: %w", err)
	}

	var sf sessionFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, nil // unrecognised format
	}

	// Prefer the title from the individual session file; fall back to index.
	title := sf.Title
	if title == "" {
		title = entry.Title
	}
	if title == "" {
		title = "Continue chat"
	}
	if len(title) > 80 {
		title = title[:80]
	}

	// Determine timestamp from the last message in history.
	ts := fileMod
	if len(sf.History) > 0 {
		last := sf.History[len(sf.History)-1]
		if last.Timestamp > 0 {
			ts = time.Unix(last.Timestamp, 0)
		}
	}

	messageCount := len(sf.History)

	// Extract model: prefer session-level, fall back to last assistant message.
	modelID := sf.ModelID
	if modelID == "" {
		for i := len(sf.History) - 1; i >= 0; i-- {
			if sf.History[i].Role == "assistant" && sf.History[i].Model != "" {
				modelID = sf.History[i].Model
				break
			}
		}
	}

	a := &domain.AIActivity{
		ID:           "continue-" + entry.SessionID,
		SessionID:    entry.SessionID,
		AgentName:    "continue",
		ActivityType: domain.ActivityTypeMessage,
		Summary:      title,
		ProjectPath:  entry.WorkspaceDirectory,
		Model:        modelID,
		Timestamp:    ts,
		Metadata: map[string]string{
			"messageCount": strconv.Itoa(messageCount),
			"model":        modelID,
			"source":       "continue-sessions",
		},
	}

	return a, nil
}
