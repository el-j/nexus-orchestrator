// Package sys_scanner implements the ports.SystemScanner interface.
package sys_scanner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

var _ ports.SystemScanner = (*Scanner)(nil)

type Scanner struct{ httpClient *http.Client }

func New() *Scanner {
	return &Scanner{httpClient: &http.Client{Timeout: 2 * time.Second}}
}

type portTarget struct {
	name, endpoint string
	port           int
	kind           domain.ProviderKind
}

type cliTarget struct {
	binary, name string
	kind         domain.ProviderKind
}

type processPattern struct {
	pattern, name string
	kind          domain.ProviderKind
}

var portTargets = []portTarget{
	{"LM Studio", "/v1/models", 1234, domain.ProviderKindLMStudio},
	{"Ollama", "/api/tags", 11434, domain.ProviderKindOllama},
	{"LocalAI", "/v1/models", 8080, domain.ProviderKindLocalAI},
	{"vLLM", "/v1/models", 8000, domain.ProviderKindVLLM},
	{"text-generation-webui", "/v1/models", 5000, domain.ProviderKindTextGenUI},
	{"Antigravity", "/v1/models", 4315, domain.ProviderKindDesktopApp},
}

var cliTargets = []cliTarget{
	{"claude", "Claude CLI", domain.ProviderKindCLI},
	{"ollama", "Ollama CLI", domain.ProviderKindOllama},
	{"lms", "LM Studio CLI", domain.ProviderKindLMStudio},
	{"aichat", "aichat", domain.ProviderKindCLI},
	{"llm", "llm (Python)", domain.ProviderKindCLI},
}

var processPatterns = []processPattern{
	{"Claude", "Claude", domain.ProviderKindDesktopApp},
	{"Antigravity", "Antigravity", domain.ProviderKindDesktopApp},
	{"ChatGPT", "ChatGPT", domain.ProviderKindDesktopApp},
	{"Copilot", "Copilot", domain.ProviderKindDesktopApp},
}

func (s *Scanner) Scan(ctx context.Context) ([]domain.DiscoveredProvider, error) {
	deadline := time.Now().Add(5 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	type probeFn func() ([]domain.DiscoveredProvider, error)
	var probes []probeFn
	for _, t := range portTargets {
		t := t
		probes = append(probes, func() ([]domain.DiscoveredProvider, error) { return s.probePort(ctx, t) })
	}
	for _, t := range cliTargets {
		t := t
		probes = append(probes, func() ([]domain.DiscoveredProvider, error) { return s.probeCLI(ctx, t) })
	}
	for _, p := range processPatterns {
		p := p
		probes = append(probes, func() ([]domain.DiscoveredProvider, error) { return s.probeProcess(ctx, p) })
	}
	const concurrency = 8
	sem := make(chan struct{}, concurrency)
	type result struct {
		providers []domain.DiscoveredProvider
		err       error
	}
	resultCh := make(chan result, len(probes))
	var wg sync.WaitGroup
	for _, probe := range probes {
		probe := probe
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				resultCh <- result{err: ctx.Err()}
				return
			}
			provs, err := probe()
			resultCh <- result{providers: provs, err: err}
		}()
	}
	go func() { wg.Wait(); close(resultCh) }()
	rawByName := make(map[string]domain.DiscoveredProvider)
	for r := range resultCh {
		if r.err != nil {
			if !errors.Is(r.err, context.Canceled) && !errors.Is(r.err, context.DeadlineExceeded) {
				log.Printf("sys_scanner: probe error: %v", r.err)
			}
			continue
		}
		for _, p := range r.providers {
			if ex, ok := rawByName[p.Name]; ok {
				rawByName[p.Name] = mergeProviders(ex, p)
			} else {
				rawByName[p.Name] = p
			}
		}
	}
	providers := make([]domain.DiscoveredProvider, 0, len(rawByName))
	for _, p := range rawByName {
		providers = append(providers, p)
	}
	sort.Slice(providers, func(i, j int) bool { return providers[i].Name < providers[j].Name })
	return providers, nil
}

func mergeProviders(a, b domain.DiscoveredProvider) domain.DiscoveredProvider {
	out := a
	if statusRank(b.Status) > statusRank(out.Status) {
		out.Status = b.Status
	}
	if strings.HasPrefix(b.ID, "port-") && !strings.HasPrefix(out.ID, "port-") {
		out.ID = b.ID
	}
	if out.BaseURL == "" {
		out.BaseURL = b.BaseURL
	}
	if out.CLIPath == "" {
		out.CLIPath = b.CLIPath
	}
	if out.ProcessName == "" {
		out.ProcessName = b.ProcessName
	}
	if len(out.Models) == 0 {
		out.Models = b.Models
	}
	return out
}

func statusRank(s domain.DiscoveryStatus) int {
	switch s {
	case domain.DiscoveryStatusReachable:
		return 3
	case domain.DiscoveryStatusRunning:
		return 2
	case domain.DiscoveryStatusInstalled:
		return 1
	}
	return 0
}

func (s *Scanner) probePort(ctx context.Context, t portTarget) ([]domain.DiscoveredProvider, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", t.port)
	dialCtx, dialCancel := context.WithTimeout(ctx, 2*time.Second)
	defer dialCancel()
	conn, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", addr)
	if err != nil {
		return nil, nil
	}
	conn.Close()
	url := fmt.Sprintf("http://127.0.0.1:%d%s", t.port, t.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("sys_scanner: build request for %s: %w", t.name, err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil // something listens there, but it does not speak HTTP
	}
	defer resp.Body.Close()
	// Ports such as 5000, 8000 and 8080 are shared with AirPlay, dev servers and
	// proxies. Only a 2xx answer carrying a model-list shaped body counts as an
	// LLM server; anything else is not reported as a provider.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	models, ok := parseModelList(body)
	if !ok {
		return nil, nil
	}

	// For Ollama, also probe /api/ps to detect actively loaded / generating models.
	var activeModels []string
	generating := false
	if t.kind == domain.ProviderKindOllama {
		activeModels, generating = s.probeOllamaPS(ctx, t.port)
	}

	return []domain.DiscoveredProvider{makePortProvider(t, models, activeModels, generating)}, nil
}

// probeOllamaPS calls the Ollama /api/ps endpoint to discover models currently
// loaded in memory and whether any generation is actively in progress.
func (s *Scanner) probeOllamaPS(ctx context.Context, port int) (activeModels []string, generating bool) {
	url := fmt.Sprintf("http://127.0.0.1:%d/api/ps", port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false
	}
	resp, err := s.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return nil, false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	var ps struct {
		Models []struct {
			Name  string `json:"name"`
			Model string `json:"model"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &ps); err != nil {
		return nil, false
	}
	for _, m := range ps.Models {
		name := m.Name
		if name == "" {
			name = m.Model
		}
		if name != "" {
			activeModels = append(activeModels, name)
			generating = true // at least one model is loaded/active
		}
	}
	return activeModels, generating
}

func makePortProvider(t portTarget, models []string, activeModels []string, generating bool) domain.DiscoveredProvider {
	return domain.DiscoveredProvider{
		ID: fmt.Sprintf("port-%d", t.port), Name: t.name, Kind: t.kind,
		Method: domain.DiscoveryMethodPort, Status: domain.DiscoveryStatusReachable,
		BaseURL:      fmt.Sprintf("http://127.0.0.1:%d", t.port),
		Models:       models,
		ActiveModels: activeModels,
		Generating:   generating,
		LastSeen:     time.Now().UTC(),
	}
}

// parseModels returns the model names advertised in an OpenAI-style
// ({"data":[{"id":...}]}) or Ollama-style ({"models":[{"name":...}]}) listing.
func parseModels(body []byte) []string {
	models, _ := parseModelList(body)
	return models
}

// parseModelList is parseModels plus a flag telling whether body has the shape
// of a model listing at all (a "data" or "models" array, possibly empty). An
// idle server with no model loaded is still a valid listing.
func parseModelList(body []byte) (models []string, ok bool) {
	var shape struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &shape); err != nil {
		return nil, false
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, false
	}
	_, hasData := raw["data"]
	_, hasModels := raw["models"]
	if !hasData && !hasModels {
		return nil, false
	}
	for _, d := range shape.Data {
		if d.ID != "" {
			models = append(models, d.ID)
		}
	}
	if len(models) > 0 {
		return models, true
	}
	for _, m := range shape.Models {
		if m.Name != "" {
			models = append(models, m.Name)
		}
	}
	return models, true
}

func (s *Scanner) probeCLI(ctx context.Context, t cliTarget) ([]domain.DiscoveredProvider, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	path, err := exec.LookPath(t.binary)
	if err != nil {
		return nil, nil
	}
	return []domain.DiscoveredProvider{{
		ID: fmt.Sprintf("cli-%s", t.binary), Name: t.name, Kind: t.kind,
		Method: domain.DiscoveryMethodCLI, Status: domain.DiscoveryStatusInstalled,
		CLIPath: path, LastSeen: time.Now().UTC(),
	}}, nil
}

func (s *Scanner) probeProcess(ctx context.Context, p processPattern) ([]domain.DiscoveredProvider, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	found, matched, _, err := detectProcess(ctx, p.pattern)
	if err != nil || !found {
		return nil, nil
	}
	return []domain.DiscoveredProvider{{
		ID:   fmt.Sprintf("proc-%s", strings.ToLower(p.pattern)),
		Name: p.name, Kind: p.kind,
		Method: domain.DiscoveryMethodProcess, Status: domain.DiscoveryStatusRunning,
		ProcessName: matched, LastSeen: time.Now().UTC(),
	}}, nil
}

// userHome returns the current user's home directory, or "" when it cannot be
// determined. Callers must skip home-relative probes on "": filepath.Join("", x)
// is a path relative to the working directory, which would make a project's own
// .claude/ folder look like the user's configuration.
func userHome() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

// Seams replaced by tests; production code never reassigns them.
var (
	goos = runtime.GOOS
	// runPgrep returns the output of `pgrep -lf pattern`.
	runPgrep = func(ctx context.Context, pattern string) ([]byte, error) {
		return exec.CommandContext(ctx, "pgrep", "-lf", pattern).Output()
	}
	// runTasklist returns the output of `tasklist /fo csv /nh`.
	runTasklist = func(ctx context.Context) ([]byte, error) {
		return exec.CommandContext(ctx, "tasklist", "/fo", "csv", "/nh").Output()
	}
)

func detectProcess(ctx context.Context, pattern string) (bool, string, int, error) {
	if goos == "windows" {
		return detectProcessWindows(ctx, pattern)
	}
	return detectProcessPgrep(ctx, pattern)
}

func detectProcessPgrep(ctx context.Context, pattern string) (bool, string, int, error) {
	out, err := runPgrep(ctx, pattern)
	if err != nil {
		return false, "", 0, nil
	}
	found, name, pid := parsePgrep(string(out), pattern)
	return found, name, pid, nil
}

// parsePgrep interprets `pgrep -lf` output ("<pid> <command line>" per line).
// Only the first match is used; when no command column is present the pattern
// itself is returned as the name.
func parsePgrep(out, pattern string) (found bool, name string, pid int) {
	line, _, _ := strings.Cut(strings.TrimSpace(out), "\n")
	line = strings.TrimSpace(line)
	if line == "" {
		return false, "", 0
	}
	parts := strings.SplitN(line, " ", 2)
	name = pattern
	if len(parts) == 2 {
		pid, _ = strconv.Atoi(parts[0])
		name = strings.TrimSpace(parts[1])
		if idx := strings.Index(name, " "); idx > 0 {
			name = name[:idx]
		}
	}
	return true, name, pid
}

func detectProcessWindows(ctx context.Context, pattern string) (bool, string, int, error) {
	out, err := runTasklist(ctx)
	if err != nil {
		return false, "", 0, fmt.Errorf("sys_scanner: tasklist: %w", err)
	}
	found, name, pid := parseTasklist(string(out), pattern)
	return found, name, pid, nil
}

// parseTasklist interprets `tasklist /fo csv /nh` output and returns the first
// process whose line contains pattern (case-insensitive), with its image name
// and PID.
func parseTasklist(out, pattern string) (found bool, name string, pid int) {
	lower := strings.ToLower(pattern)
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(strings.ToLower(line), lower) {
			continue
		}
		line = strings.TrimSpace(line)
		parts := strings.Split(line, ",")
		if len(parts) > 1 {
			pid, _ = strconv.Atoi(strings.Trim(parts[1], `"`))
		}
		if len(line) > 0 && line[0] == '"' {
			if end := strings.Index(line[1:], "\""); end >= 0 {
				return true, line[1 : end+1], pid
			}
		}
		return true, pattern, pid
	}
	return false, "", 0
}
