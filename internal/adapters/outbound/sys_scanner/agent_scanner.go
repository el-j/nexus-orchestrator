package sys_scanner

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	home := os.Getenv("HOME")

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		var m map[string]json.RawMessage
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
	home := os.Getenv("HOME")
	extDir := filepath.Join(home, ".vscode", "extensions")
	if runtime.GOOS == "windows" {
		extDir = filepath.Join(os.Getenv("USERPROFILE"), ".vscode", "extensions")
	}
	return probeVSCodeExtensionsDir(ctx, extDir)
}

func probeMCPPortList(ctx context.Context, ports []int) []domain.DiscoveredAgent {
	initReq := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"nexus-probe","version":"1"}}}`
	var results []domain.DiscoveredAgent

	for _, port := range ports {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
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
	if runtime.GOOS == "windows" {
		return nil
	}
	patterns := []string{"--mcp", "--mcp-server"}
	seen := map[string]bool{}
	var results []domain.DiscoveredAgent
	for _, pat := range patterns {
		out, err := exec.CommandContext(ctx, "pgrep", "-lf", pat).Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, " ", 2)
			procName := ""
			pid := 0
			if len(parts) > 1 {
				pid, _ = strconv.Atoi(parts[0])
				procName = strings.TrimSpace(parts[1])
				if idx := strings.Index(procName, " "); idx > 0 {
					procName = procName[:idx]
				}
			}
			if procName == "" || seen[procName] {
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

// probeClaudeSubAgentsDir is the testable inner implementation.
// If homeDir is empty, os.UserHomeDir() is used.
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
	const staleThreshold = 2 * time.Hour
	const activeThreshold = 5 * time.Minute
	const maxLines = 500

	var results []domain.DiscoveredAgent

	// Walk up to 2 levels deep: projectsDir/<encoded-path>/<session>.jsonl
	encodedDirs, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil
	}

	for _, encodedEntry := range encodedDirs {
		if !encodedEntry.IsDir() {
			continue
		}
		encodedName := encodedEntry.Name()
		encodedPath := filepath.Join(projectsDir, encodedName)

		// Decode base64 directory name to get the working directory.
		decodedBytes, decErr := base64.StdEncoding.DecodeString(encodedName)
		workingDirFromDir := ""
		if decErr == nil {
			workingDirFromDir = string(decodedBytes)
		}

		sessionFiles, err := os.ReadDir(encodedPath)
		if err != nil {
			continue
		}

		// Collect all valid sessions for this directory, then emit one agent.
		var dirSessions []domain.DiscoveredAgent
		var latestMtime time.Time

		for _, sessionEntry := range sessionFiles {
			if sessionEntry.IsDir() {
				continue
			}
			if !strings.HasSuffix(sessionEntry.Name(), ".jsonl") {
				continue
			}

			filePath := filepath.Join(encodedPath, sessionEntry.Name())
			info, err := sessionEntry.Info()
			if err != nil {
				continue
			}
			mtime := info.ModTime()
			if now.Sub(mtime) > staleThreshold {
				continue
			}

			agent := parseClaudeSessionFile(filePath, maxLines)
			if agent == nil {
				continue
			}

			// Fall back to directory-decoded working dir if JSONL cwd was empty.
			if agent.WorkingDir == "" {
				agent.WorkingDir = workingDirFromDir
			}

			agent.IsRunning = now.Sub(mtime) < activeThreshold
			agent.Kind = domain.AgentKindClaudeCLI
			agent.DetectionMethod = "claude-session-file"
			agent.LastSeen = mtime

			dirSessions = append(dirSessions, *agent)
			if mtime.After(latestMtime) {
				latestMtime = mtime
			}
		}

		if len(dirSessions) == 0 {
			continue
		}

		// Pick the most recently seen session as the representative.
		var rep domain.DiscoveredAgent
		for _, s := range dirSessions {
			if s.LastSeen.Equal(latestMtime) {
				rep = s
				break
			}
		}

		// Stable ID per project directory (not per session file).
		rep.ID = "claude-cli-" + encodedName

		// Build Name with session count.
		baseName := "Claude Code"
		if rep.WorkingDir != "" {
			baseName = "Claude Code " + filepath.Base(rep.WorkingDir)
		}
		if len(dirSessions) > 1 {
			baseName += " (" + strconv.Itoa(len(dirSessions)) + " sessions)"
		}
		rep.Name = baseName

		results = append(results, rep)
	}

	return results
}

// parseClaudeSessionFile reads up to maxLines from a JSONL session file and
// extracts agent metadata. Returns nil if no agentId was found.
func parseClaudeSessionFile(filePath string, maxLines int) *domain.DiscoveredAgent {
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	type jsonlMsg struct {
		AgentID     string `json:"agentId"`
		IsSidechain bool   `json:"isSidechain"`
		ParentUUID  string `json:"parentUuid"`
		Model       string `json:"model"`
		CWD         string `json:"cwd"`
	}

	var (
		agentID     string
		model       string
		cwd         string
		isSidechain bool
		parentUUID  string
	)

	sc := bufio.NewScanner(f)
	lineCount := 0
	for sc.Scan() {
		if lineCount >= maxLines {
			break
		}
		lineCount++

		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}

		var msg jsonlMsg
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		if agentID == "" && msg.AgentID != "" {
			agentID = msg.AgentID
		}
		if model == "" && msg.Model != "" {
			model = msg.Model
		}
		if cwd == "" && msg.CWD != "" {
			cwd = msg.CWD
		}
		if msg.IsSidechain {
			isSidechain = true
		}
		if parentUUID == "" && msg.ParentUUID != "" {
			parentUUID = msg.ParentUUID
		}
	}

	// Use filename (without extension) as fallback ID.
	if agentID == "" {
		base := filepath.Base(filePath)
		agentID = strings.TrimSuffix(base, filepath.Ext(base))
	}

	agent := &domain.DiscoveredAgent{
		ID:         agentID,
		ModelID:    model,
		WorkingDir: cwd,
	}

	if isSidechain && parentUUID != "" {
		agent.ParentAgentID = parentUUID
	}

	return agent
}
