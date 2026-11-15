package sys_scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
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

	found := make([][]domain.DiscoveredAgent, 0, len(probes))
	for range probes {
		found = append(found, <-ch)
	}
	return mergeAgents(found, time.Now()), nil
}

// mergeAgents joins what the probes found into one list without duplicates: an agent seen by several
// probes keeps everything any of them learned (it is running if one saw it running), a newly seen
// agent is stamped with now, and each sub-agent is added to the SubAgentIDs of its parent.
func mergeAgents(found [][]domain.DiscoveredAgent, now time.Time) []domain.DiscoveredAgent {
	merged := map[string]domain.DiscoveredAgent{}
	for _, agents := range found {
		for _, a := range agents {
			existing, ok := merged[a.ID]
			if !ok {
				a.LastSeen = now
				merged[a.ID] = a
				continue
			}
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
			if a.PID > 0 {
				existing.PID = a.PID
			}
			if a.ProcessName != "" {
				existing.ProcessName = a.ProcessName
			}
			merged[a.ID] = existing
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
	return out
}

// probeClaudeConfig reports Claude Code (settings in ~/.claude) and the Claude desktop app (its application
// support folder) when their configuration exists in the user's home.
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

// probeVSCodeExtensionsDir reports the agent extensions installed in one VS Code extensions directory.
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

// probeVSCodeExtensions looks for agent extensions in the user's VS Code extensions directory.
func (s *Scanner) probeVSCodeExtensions(ctx context.Context) []domain.DiscoveredAgent {
	home := userHome()
	if home == "" {
		return nil
	}
	return probeVSCodeExtensionsDir(ctx, filepath.Join(home, ".vscode", "extensions"))
}

// probeMCPPortList sends an MCP initialize request to each local port and reports those that answer as
// MCP servers.
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

// probeMCPPorts probes the usual local MCP ports.
func (s *Scanner) probeMCPPorts(ctx context.Context) []domain.DiscoveredAgent {
	mcpPorts := []int{3000, 3001, 3100, 5100, 6006, 7007, 8008, 9009}
	return probeMCPPortList(ctx, mcpPorts)
}

// probeProcessFlags finds running processes started with an MCP flag; it does nothing on Windows.
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

// probeAgentProcesses finds running agent programs by their process names.
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
