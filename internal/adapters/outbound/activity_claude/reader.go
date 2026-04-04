// Package activity_claude implements ports.ActivityReader for the Claude Code agent.
// It reads session JSONL files written by Claude Code to ~/.claude/projects/.
package activity_claude

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// ClaudeJSONLReader reads Claude Code session JSONL files from ~/.claude/projects/.
// It implements ports.ActivityReader for the Claude Code agent.
type ClaudeJSONLReader struct {
	baseDir string           // usually ~/.claude/projects
	filePos map[string]int64 // file path -> last read byte offset (in-memory)
	mu      sync.Mutex
}

// NewClaudeJSONLReader creates a reader pointing at the default ~/.claude/projects directory.
func NewClaudeJSONLReader() *ClaudeJSONLReader {
	home, _ := os.UserHomeDir()
	return &ClaudeJSONLReader{
		baseDir: filepath.Join(home, ".claude", "projects"),
		filePos: make(map[string]int64),
	}
}

// NewClaudeJSONLReaderAt creates a reader pointing at the given base directory.
// Useful for testing or when Claude projects are stored in a non-standard location.
func NewClaudeJSONLReaderAt(baseDir string) *ClaudeJSONLReader {
	return &ClaudeJSONLReader{
		baseDir: baseDir,
		filePos: make(map[string]int64),
	}
}

// SourceName identifies this reader.
func (r *ClaudeJSONLReader) SourceName() string { return "claude-jsonl" }

// claudeLine is the JSON structure of a single record in a Claude JSONL file.
type claudeLine struct {
	Type      string    `json:"type"`
	UUID      string    `json:"uuid"`
	SessionID string    `json:"sessionId"`
	CWD       string    `json:"cwd"`
	Timestamp time.Time `json:"timestamp"`
	Name      string    `json:"name"` // tool name; present only when type="tool_use"
	Message   struct {
		Model string `json:"model"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// ReadActivities walks baseDir for JSONL files modified since the given time,
// reads newly appended lines from each file, and returns parsed activities.
func (r *ClaudeJSONLReader) ReadActivities(ctx context.Context, since time.Time) ([]domain.AIActivity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, err := os.Stat(r.baseDir); os.IsNotExist(err) {
		return nil, nil
	}

	var activities []domain.AIActivity

	walkErr := filepath.WalkDir(r.baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries; continue walk
		}
		if d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}
		// Skip files with no changes since the cutoff.
		if info.ModTime().Before(since) {
			return nil
		}

		acts, newOffset, readErr := r.readNewLines(path, r.filePos[path], since)
		if readErr != nil {
			return nil // skip problematic files; continue walk
		}
		r.filePos[path] = newOffset
		activities = append(activities, acts...)
		return nil
	})

	return activities, walkErr
}

// readNewLines opens path, seeks to offset, reads all complete new lines, and
// returns activities together with the updated file offset.
func (r *ClaudeJSONLReader) readNewLines(path string, offset int64, since time.Time) ([]domain.AIActivity, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, offset, fmt.Errorf("activity_claude: open: %w", err)
	}
	defer f.Close()

	if offset > 0 {
		if _, seekErr := f.Seek(offset, io.SeekStart); seekErr != nil {
			// File may have been truncated and rewritten; restart from beginning.
			offset = 0
			if _, seekErr2 := f.Seek(0, io.SeekStart); seekErr2 != nil {
				return nil, 0, fmt.Errorf("activity_claude: seek: %w", seekErr2)
			}
		}
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, offset, fmt.Errorf("activity_claude: read: %w", err)
	}
	if len(data) == 0 {
		return nil, offset, nil
	}

	// Advance offset only through the last complete line to avoid consuming
	// partial lines that are still being written.
	lastNL := bytes.LastIndexByte(data, '\n')
	if lastNL < 0 {
		// No complete line yet; hold position and retry next call.
		return nil, offset, nil
	}
	newOffset := offset + int64(lastNL) + 1
	data = data[:lastNL+1] // process complete lines only

	segments := bytes.Split(data, []byte("\n"))
	var activities []domain.AIActivity
	pos := offset // byte position of the current segment's start in the file

	for i, raw := range segments {
		segStart := pos

		// Advance pos past this segment plus its trailing newline (if not last).
		if i < len(segments)-1 {
			pos += int64(len(raw)) + 1
		}

		trimmed := bytes.TrimSpace(raw)
		if len(trimmed) == 0 {
			continue
		}

		activity, ok := parseClaudeLine(trimmed, path, segStart)
		if !ok {
			continue
		}
		// Skip activities older than the cutoff (relevant on first full-file read).
		if !activity.Timestamp.IsZero() && activity.Timestamp.Before(since) {
			continue
		}
		activities = append(activities, activity)
	}

	return activities, newOffset, nil
}

// parseClaudeLine parses a single JSONL line into an AIActivity.
// Returns (zero, false) when the line is unparseable or the type is unrecognised.
func parseClaudeLine(line []byte, filePath string, lineOffset int64) (domain.AIActivity, bool) {
	var cl claudeLine
	if err := json.Unmarshal(line, &cl); err != nil {
		return domain.AIActivity{}, false
	}

	// Build a stable ID.
	var id string
	if cl.UUID != "" {
		if cl.SessionID != "" {
			id = "claude-" + cl.SessionID + "-" + cl.UUID
		} else {
			id = "claude-" + cl.UUID
		}
	} else {
		h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", filePath, lineOffset)))
		id = fmt.Sprintf("claude-%x", h[:8])
	}

	var actType domain.ActivityType
	var summary string

	switch cl.Type {
	case "user":
		actType = domain.ActivityTypeMessage
		summary = "User prompt"
	case "assistant":
		actType = domain.ActivityTypeGeneration
		if cl.Message.Usage.OutputTokens > 0 {
			summary = fmt.Sprintf("Responding (%d tokens)", cl.Message.Usage.OutputTokens)
		} else {
			summary = "Responding"
		}
	case "tool_use":
		actType = domain.ActivityTypeToolUse
		name := cl.Name
		if name == "" {
			name = "unknown"
		}
		summary = "Using " + name
	default:
		return domain.AIActivity{}, false
	}

	return domain.AIActivity{
		ID:           id,
		SessionID:    cl.SessionID,
		AgentName:    "claude",
		ActivityType: actType,
		Summary:      summary,
		ProjectPath:  cl.CWD,
		Model:        cl.Message.Model,
		TokensIn:     cl.Message.Usage.InputTokens,
		TokensOut:    cl.Message.Usage.OutputTokens,
		Timestamp:    cl.Timestamp,
	}, true
}
