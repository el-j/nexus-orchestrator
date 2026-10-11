package sys_scanner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

var _ ports.AgentScanner = (*Scanner)(nil)

// ScanAgents runs all agent detection probes concurrently and returns a deduplicated list.
func (s *Scanner) ScanAgents(ctx context.Context) ([]domain.DiscoveredAgent, error) {
	probes := []func(context.Context) []domain.DiscoveredAgent{
		s.probeClaudeConfig,
		s.probeVSCodeExtensions,
		s.probeMCPPorts,
		s.probeProcessFlags,
		s.probeAgentProcesses,
		s.probeClaudeSubAgents,
	}

	ch := make(chan []domain.DiscoveredAgent, len(probes))
	for _, probe := range probes {
		probe := probe
		go func() {
			ch <- probe(ctx)
		}()
	}

	merged := map[string]domain.DiscoveredAgent{}
	for range probes {
		agents := <-ch
		for _, a := range agents {
			if existing, ok := merged[a.ID]; ok {
				if a.IsRunning {
					existing.IsRunning = true
				}
				if a.MCPEndpoint != "" {
					existing.MCPEndpoint = a.MCPEndpoint
				}
				if a.ConfigPath != "" {
					existing.ConfigPath = a.ConfigPath
				}
				if a.CLIPath != "" {
					existing.CLIPath = a.CLIPath
				}
				merged[a.ID] = existing
			} else {
				a.LastSeen = time.Now()
				merged[a.ID] = a
			}
		}
	}

	// Post-merge pass: populate SubAgentIDs on parent agents.
	for id, a := range merged {
		if a.ParentAgentID == "" {
			continue
		}
		parent, ok := merged[a.ParentAgentID]
		if !ok {
			continue
		}
		parent.SubAgentIDs = append(parent.SubAgentIDs, id)
		merged[a.ParentAgentID] = parent
	}

	out := make([]domain.DiscoveredAgent, 0, len(merged))
	for _, a := range merged {
		out = append(out, a)
	}
	return out, nil
}

func (s *Scanner) probeClaudeConfig(ctx context.Context) []domain.DiscoveredAgent {
	var results []domain.DiscoveredAgent
	home := userHome()
	if home == "" {
		return nil
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		var m map[string]interface{}
		if json.Unmarshal(data, &m) == nil {
			results = append(results, domain.DiscoveredAgent{
				ID:              "claude-cli",
				Kind:            domain.AgentKindClaudeCLI,
				Name:            "Claude CLI",
				DetectionMethod: "fs-config",
				ConfigPath:      settingsPath,
			})
		}
	}

	desktopPaths := []string{
		filepath.Join(home, "Library", "Application Support", "Claude"),
		filepath.Join(home, ".config", "claude"),
	}
	for _, p := range desktopPaths {
		if _, err := os.Stat(p); err == nil {
			results = append(results, domain.DiscoveredAgent{
				ID:              "claude-desktop",
				Kind:            domain.AgentKindClaudeDesktop,
				Name:            "Claude Desktop",
				DetectionMethod: "fs-config",
				ConfigPath:      p,
			})
			break
		}
	}

	return results
}

var vscodeExtMap = map[string]struct {
	kind domain.AgentKind
	name string
}{
	"saoudrizwan.claude-dev":        {domain.AgentKindCline, "Cline"},
	"continue.continue":             {domain.AgentKindContinue, "Continue"},
	"codeium.codeium":               {domain.AgentKindCodeGPT, "Codeium"},
	"codegpt.codegpt":               {domain.AgentKindCodeGPT, "CodeGPT"},
	"anysphere.cursor-always-local": {domain.AgentKindCursor, "Cursor AI"},
	"github.copilot":                {domain.AgentKindCopilot, "GitHub Copilot"},
}

func probeVSCodeExtensionsDir(_ context.Context, extDir string) []domain.DiscoveredAgent {
	entries, err := os.ReadDir(extDir)
	if err != nil {
		return nil
	}

	seen := map[string]bool{}
	var results []domain.DiscoveredAgent
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		for prefix, info := range vscodeExtMap {
			if strings.HasPrefix(name, prefix) && !seen[prefix] {
				seen[prefix] = true
				results = append(results, domain.DiscoveredAgent{
					ID:              string(info.kind),
					Kind:            info.kind,
					Name:            info.name,
					DetectionMethod: "vscode-extension",
					ConfigPath:      filepath.Join(extDir, name),
				})
			}
		}
	}
	return results
}

func (s *Scanner) probeVSCodeExtensions(ctx context.Context) []domain.DiscoveredAgent {
	home := userHome()
	if home == "" {
		return nil
	}
	return probeVSCodeExtensionsDir(ctx, filepath.Join(home, ".vscode", "extensions"))
}

func probeMCPPortList(ctx context.Context, ports []int) []domain.DiscoveredAgent {
	initReq := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"nexus-probe","version":"1"}}}`
	var results []domain.DiscoveredAgent

	for _, port := range ports {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		dialer := net.Dialer{Timeout: 500 * time.Millisecond}
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			continue
		}
		conn.Close()

		reqCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		req, _ := http.NewRequestWithContext(reqCtx, http.MethodPost,
			fmt.Sprintf("http://127.0.0.1:%d/mcp", port),
			strings.NewReader(initReq))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		cancel()
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var rpcResp struct {
			Result struct {
				ServerInfo struct {
					Name string `json:"name"`
				} `json:"serverInfo"`
			} `json:"result"`
		}
		if json.Unmarshal(body, &rpcResp) == nil && rpcResp.Result.ServerInfo.Name != "" {
			results = append(results, domain.DiscoveredAgent{
				ID:              fmt.Sprintf("mcp-%d", port),
				Kind:            domain.AgentKindGeneric,
				Name:            rpcResp.Result.ServerInfo.Name,
				DetectionMethod: "port-mcp",
				MCPEndpoint:     fmt.Sprintf("http://127.0.0.1:%d/mcp", port),
				IsRunning:       true,
			})
		}
	}
	return results
}

func (s *Scanner) probeMCPPorts(ctx context.Context) []domain.DiscoveredAgent {
	mcpPorts := []int{3000, 3001, 3100, 5100, 6006, 7007, 8008, 9009}
	return probeMCPPortList(ctx, mcpPorts)
}

func (s *Scanner) probeProcessFlags(ctx context.Context) []domain.DiscoveredAgent {
	if goos == "windows" {
		return nil
	}
	patterns := []string{"--mcp", "--mcp-server"}
	seen := map[string]bool{}
	var results []domain.DiscoveredAgent
	for _, pat := range patterns {
		out, err := runPgrep(ctx, pat)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			// pattern "" => a line without a command column yields an empty name.
			found, procName, pid := parsePgrep(line, "")
			if !found || procName == "" || seen[procName] {
				continue
			}
			seen[procName] = true
			results = append(results, domain.DiscoveredAgent{
				ID:              "proc-mcp-" + procName,
				Kind:            domain.AgentKindGeneric,
				Name:            procName,
				DetectionMethod: "process-flag",
				ProcessName:     procName,
				IsRunning:       true,
				PID:             pid,
			})
		}
	}
	return results
}

func (s *Scanner) probeAgentProcesses(ctx context.Context) []domain.DiscoveredAgent {
	type patternDef struct {
		pattern string
		kind    domain.AgentKind
		name    string
	}
	patterns := []patternDef{
		{"Claude", domain.AgentKindClaudeDesktop, "Claude Desktop"},
		{"Antigravity", domain.AgentKindAntigravity, "Antigravity"},
	}
	var results []domain.DiscoveredAgent
	for _, p := range patterns {
		found, matched, pid, _ := detectProcess(ctx, p.pattern)
		if !found {
			continue
		}
		results = append(results, domain.DiscoveredAgent{
			ID:              string(p.kind),
			Kind:            p.kind,
			Name:            p.name,
			DetectionMethod: "process",
			ProcessName:     matched,
			IsRunning:       true,
			PID:             pid,
		})
	}
	return results
}

// probeClaudeSubAgents reads Claude Code session JSONL files from
// ~/.claude/projects/ to discover active Claude CLI sessions and sub-agents.
func (s *Scanner) probeClaudeSubAgents(ctx context.Context) []domain.DiscoveredAgent {
	return probeClaudeSubAgentsDir(ctx, "")
}

// claudeStaleThreshold, claudeActiveThreshold and claudeMaxLines bound what the
// Claude session probe reads: sessions untouched for two hours are ignored, one
// written to within five minutes counts as running, and only the head of each file
// is parsed (they grow to many megabytes).
const (
	claudeStaleThreshold  = 2 * time.Hour
	claudeActiveThreshold = 5 * time.Minute
	claudeMaxLines        = 500
	// claudeMaxSubAgents caps the sub-agents reported per project so a long agentic
	// run cannot flood the agent list.
	claudeMaxSubAgents = 25
)

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
