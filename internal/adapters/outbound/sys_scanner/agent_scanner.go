package sys_scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
			if len(parts) > 1 {
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
		found, matched, _ := detectProcess(ctx, p.pattern)
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
		})
	}
	return results
}
