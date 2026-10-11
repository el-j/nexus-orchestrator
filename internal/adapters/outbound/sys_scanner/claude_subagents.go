package sys_scanner

import (
	"bufio"
	"context"
	"encoding/json"
	"nexus-orchestrator/internal/core/domain"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// probeClaudeSubAgents reads Claude Code session JSONL files from
// ~/.claude/projects/ to discover active Claude CLI sessions and sub-agents.
func (s *Scanner) probeClaudeSubAgents(ctx context.Context) []domain.DiscoveredAgent {
	return probeClaudeSubAgentsDir(ctx, "")
}

// probeClaudeSubAgentsDir is the testable inner implementation.
// If homeDir is empty, os.UserHomeDir() is used.
//
// Claude Code lays sessions out as
//
//	~/.claude/projects/<project-path-with-dashes>/<sessionId>.jsonl                      main session
//	~/.claude/projects/<project-path-with-dashes>/<sessionId>/subagents/agent-<id>.jsonl  sub-agent
//
// A project directory yields one agent (its most recently active session). Each
// sub-agent file yields a child agent linked to that project agent. The link comes
// from the directory structure: a sub-agent's own "parentUuid" is null on its
// first message and otherwise points at a message inside the sidechain, so it
// cannot identify a parent.
func probeClaudeSubAgentsDir(_ context.Context, homeDir string) []domain.DiscoveredAgent {
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return nil
		}
	}

	projectsDir := filepath.Join(homeDir, ".claude", "projects")
	if _, err := os.Stat(projectsDir); err != nil {
		return nil
	}

	now := time.Now()
	var results []domain.DiscoveredAgent

	projectDirs, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil
	}
	for _, projectEntry := range projectDirs {
		if !projectEntry.IsDir() {
			continue
		}
		dirName := projectEntry.Name()
		projectPath := filepath.Join(projectsDir, dirName)
		entries, err := os.ReadDir(projectPath)
		if err != nil {
			continue
		}

		parentID := "claude-cli-" + dirName
		var sessions []domain.DiscoveredAgent
		var subAgents []domain.DiscoveredAgent
		var latest time.Time

		for _, entry := range entries {
			switch {
			case entry.IsDir():
				subAgents = append(subAgents, claudeSubAgents(filepath.Join(projectPath, entry.Name(), "subagents"), parentID, now)...)
			case strings.HasSuffix(entry.Name(), ".jsonl"):
				info, err := entry.Info()
				if err != nil || now.Sub(info.ModTime()) > claudeStaleThreshold {
					continue
				}
				agent := parseClaudeSessionFile(filepath.Join(projectPath, entry.Name()), claudeMaxLines)
				if agent == nil {
					continue
				}
				agent.IsRunning = now.Sub(info.ModTime()) < claudeActiveThreshold
				agent.Kind = domain.AgentKindClaudeCLI
				agent.DetectionMethod = "claude-session-file"
				agent.LastSeen = info.ModTime()
				sessions = append(sessions, *agent)
				if info.ModTime().After(latest) {
					latest = info.ModTime()
				}
			}
		}

		if len(sessions) > 0 {
			// The most recently active session represents the project.
			var rep domain.DiscoveredAgent
			for _, sess := range sessions {
				if sess.LastSeen.Equal(latest) {
					rep = sess
					break
				}
			}
			rep.ID = parentID // stable per project directory, not per session file
			// The newest session may be too fresh to have recorded a cwd or model yet;
			// the project's other sessions share them.
			for _, sess := range sessions {
				if rep.WorkingDir == "" {
					rep.WorkingDir = sess.WorkingDir
				}
				if rep.ModelID == "" {
					rep.ModelID = sess.ModelID
				}
			}
			name := "Claude Code"
			if rep.WorkingDir != "" {
				name = "Claude Code " + filepath.Base(rep.WorkingDir)
			}
			if len(sessions) > 1 {
				name += " (" + strconv.Itoa(len(sessions)) + " sessions)"
			}
			rep.Name = name
			results = append(results, rep)
		}

		// Newest sub-agents first, bounded per project.
		sort.Slice(subAgents, func(i, j int) bool { return subAgents[i].LastSeen.After(subAgents[j].LastSeen) })
		if len(subAgents) > claudeMaxSubAgents {
			subAgents = subAgents[:claudeMaxSubAgents]
		}
		results = append(results, subAgents...)
	}
	return results
}

// claudeSubAgents reads the agent-*.jsonl files of one session's subagents
// directory. Files untouched for claudeStaleThreshold are skipped.
func claudeSubAgents(dir, parentID string, now time.Time) []domain.DiscoveredAgent {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []domain.DiscoveredAgent
	for _, f := range files {
		name := f.Name()
		if f.IsDir() || !strings.HasPrefix(name, "agent-") || !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		info, err := f.Info()
		if err != nil || now.Sub(info.ModTime()) > claudeStaleThreshold {
			continue
		}
		meta := readClaudeJSONL(filepath.Join(dir, name), claudeMaxLines)
		if meta == nil {
			continue
		}
		agentID := meta.AgentID
		if agentID == "" {
			agentID = strings.TrimSuffix(strings.TrimPrefix(name, "agent-"), ".jsonl")
		}
		label := meta.Slug
		if label == "" {
			label = agentID
			if len(label) > 8 {
				label = label[:8]
			}
		}
		out = append(out, domain.DiscoveredAgent{
			ID:              "claude-agent-" + agentID,
			Kind:            domain.AgentKindClaudeCLI,
			Name:            "Sub-agent " + label,
			DetectionMethod: "claude-subagent-file",
			IsRunning:       now.Sub(info.ModTime()) < claudeActiveThreshold,
			LastSeen:        info.ModTime(),
			ModelID:         meta.Model,
			WorkingDir:      meta.CWD,
			ParentAgentID:   parentID,
		})
	}
	return out
}

// claudeMeta is what the session probe needs from the head of a session file.
type claudeMeta struct {
	AgentID string
	Model   string
	CWD     string
	Slug    string
}

// readClaudeJSONL reads up to maxLines lines of a Claude Code JSONL file and
// collects the first value of each field it cares about. Assistant messages carry
// the model under message.model; older exports put it at the top level, so both
// are read. It returns nil when the file cannot be opened.
func readClaudeJSONL(filePath string, maxLines int) *claudeMeta {
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	type line struct {
		AgentID string `json:"agentId"`
		Model   string `json:"model"`
		CWD     string `json:"cwd"`
		Slug    string `json:"slug"`
		Message struct {
			Model string `json:"model"`
		} `json:"message"`
	}

	meta := &claudeMeta{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8<<20) // tool results can make single lines very long
	for n := 0; sc.Scan() && n < maxLines; n++ {
		text := strings.TrimSpace(sc.Text())
		if text == "" {
			continue
		}
		var l line
		if json.Unmarshal([]byte(text), &l) != nil {
			continue
		}
		if meta.AgentID == "" {
			meta.AgentID = l.AgentID
		}
		if meta.Model == "" {
			if l.Model != "" {
				meta.Model = l.Model
			} else {
				meta.Model = l.Message.Model
			}
		}
		if meta.CWD == "" {
			meta.CWD = l.CWD
		}
		if meta.Slug == "" {
			meta.Slug = l.Slug
		}
	}
	return meta
}

// parseClaudeSessionFile reads the head of a main session file and returns its
// agent metadata. The ID falls back to the file name when no agentId is present.
// It returns nil when the file cannot be opened.
func parseClaudeSessionFile(filePath string, maxLines int) *domain.DiscoveredAgent {
	meta := readClaudeJSONL(filePath, maxLines)
	if meta == nil {
		return nil
	}
	id := meta.AgentID
	if id == "" {
		base := filepath.Base(filePath)
		id = strings.TrimSuffix(base, filepath.Ext(base))
	}
	return &domain.DiscoveredAgent{ID: id, ModelID: meta.Model, WorkingDir: meta.CWD}
}
