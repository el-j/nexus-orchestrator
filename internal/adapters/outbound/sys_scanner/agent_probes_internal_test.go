package sys_scanner

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

// isolateHome points the user's home at a fresh temp dir (Unix HOME and Windows USERPROFILE).
func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestProbeClaudeConfig_DetectsCLIAndDesktop(t *testing.T) {
	home := isolateHome(t)
	writeFile(t, filepath.Join(home, ".claude", "settings.json"), `{"theme":"dark"}`)
	if err := os.MkdirAll(filepath.Join(home, ".config", "claude"), 0o750); err != nil {
		t.Fatal(err)
	}
	got := newTestScanner().probeClaudeConfig(context.Background())
	ids := map[string]string{}
	for _, a := range got {
		ids[a.ID] = a.ConfigPath
	}
	if ids["claude-cli"] != filepath.Join(home, ".claude", "settings.json") || ids["claude-desktop"] != filepath.Join(home, ".config", "claude") {
		t.Errorf("detected = %v", ids)
	}
}

func TestProbeClaudeConfig_IgnoresInvalidSettingsAndNothingInstalled(t *testing.T) {
	home := isolateHome(t)
	if got := newTestScanner().probeClaudeConfig(context.Background()); len(got) != 0 {
		t.Errorf("empty home: %v", got)
	}
	writeFile(t, filepath.Join(home, ".claude", "settings.json"), `{not json`)
	if got := newTestScanner().probeClaudeConfig(context.Background()); len(got) != 0 {
		t.Errorf("a corrupt settings.json is not evidence of an install: %v", got)
	}
}

// A project's own .claude/settings.json must never be mistaken for the user's
// configuration when the home directory cannot be determined.
func TestHomeRelativeProbesAreSkippedWhenHomeIsUnknown(t *testing.T) {
	work := t.TempDir()
	writeFile(t, filepath.Join(work, ".claude", "settings.json"), `{}`)
	writeFile(t, filepath.Join(work, ".vscode", "extensions", "github.copilot-1.0.0", "package.json"), `{}`)
	old, _ := os.Getwd()
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	for _, k := range []string{"HOME", "USERPROFILE", "home"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	if userHome() != "" {
		t.Skip("platform resolves a home directory without HOME/USERPROFILE")
	}
	s := newTestScanner()
	if got := s.probeClaudeConfig(context.Background()); len(got) != 0 {
		t.Errorf("relative .claude/settings.json was picked up: %v", got)
	}
	if got := s.probeVSCodeExtensions(context.Background()); len(got) != 0 {
		t.Errorf("relative .vscode/extensions was picked up: %v", got)
	}
}

func TestProbeVSCodeExtensions_FromHome(t *testing.T) {
	home := isolateHome(t)
	for _, d := range []string{"github.copilot-1.2.3", "saoudrizwan.claude-dev-3.0.0", "unrelated.ext-1.0.0"} {
		if err := os.MkdirAll(filepath.Join(home, ".vscode", "extensions", d), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(home, ".vscode", "extensions", "continue.continue-9"), "file, not a dir")
	got := newTestScanner().probeVSCodeExtensions(context.Background())
	kinds := map[domain.AgentKind]bool{}
	for _, a := range got {
		kinds[a.Kind] = true
	}
	if !kinds[domain.AgentKindCopilot] || !kinds[domain.AgentKindCline] || len(got) != 2 {
		t.Errorf("got %+v", got)
	}
}

func stubPgrep(t *testing.T, fn func(pattern string) ([]byte, error)) {
	t.Helper()
	old := runPgrep
	runPgrep = func(_ context.Context, p string) ([]byte, error) { return fn(p) }
	t.Cleanup(func() { runPgrep = old })
}

func stubGOOS(t *testing.T, name string) {
	t.Helper()
	old := goos
	goos = name
	t.Cleanup(func() { goos = old })
}

func TestProbeProcessFlags_DeduplicatesAndSkipsBlankRecords(t *testing.T) {
	stubGOOS(t, "linux")
	stubPgrep(t, func(p string) ([]byte, error) {
		switch p {
		case "--mcp":
			return []byte("101 /usr/bin/mcp-tool --mcp\n\n102 /usr/bin/other --mcp\n103\n"), nil
		case "--mcp-server":
			return []byte("104 /usr/bin/mcp-tool --mcp-server\n"), nil // duplicate name
		}
		return nil, errors.New("unexpected pattern")
	})
	got := newTestScanner().probeProcessFlags(context.Background())
	names := map[string]int{}
	for _, a := range got {
		names[a.Name] = a.PID
		if !a.IsRunning || a.DetectionMethod != "process-flag" || !strings.HasPrefix(a.ID, "proc-mcp-") {
			t.Errorf("bad agent: %+v", a)
		}
	}
	if len(got) != 2 || names["/usr/bin/mcp-tool"] != 101 || names["/usr/bin/other"] != 102 {
		t.Errorf("got %v", names)
	}
}

func TestProbeProcessFlags_NoPgrepOrWindows(t *testing.T) {
	stubGOOS(t, "linux")
	stubPgrep(t, func(string) ([]byte, error) { return nil, errors.New("exit status 1") })
	if got := newTestScanner().probeProcessFlags(context.Background()); len(got) != 0 {
		t.Errorf("no matches: %v", got)
	}
	stubGOOS(t, "windows")
	if got := newTestScanner().probeProcessFlags(context.Background()); got != nil {
		t.Errorf("windows has no pgrep: %v", got)
	}
}

func TestProbeAgentProcesses_UsesProcessDetection(t *testing.T) {
	stubGOOS(t, "linux")
	stubPgrep(t, func(p string) ([]byte, error) {
		if p == "Claude" {
			return []byte("77 /Applications/Claude.app/Contents/MacOS/Claude\n"), nil
		}
		return nil, errors.New("none")
	})
	got := newTestScanner().probeAgentProcesses(context.Background())
	if len(got) != 1 || got[0].Kind != domain.AgentKindClaudeDesktop || got[0].PID != 77 || !got[0].IsRunning {
		t.Errorf("got %+v", got)
	}
}

func TestDetectProcess_WindowsBranchUsesTasklist(t *testing.T) {
	stubGOOS(t, "windows")
	old := runTasklist
	defer func() { runTasklist = old }()

	runTasklist = func(context.Context) ([]byte, error) {
		return []byte("\"Antigravity.exe\",\"900\",\"Console\"\r\n"), nil
	}
	found, name, pid, err := detectProcess(context.Background(), "antigravity")
	if err != nil || !found || name != "Antigravity.exe" || pid != 900 {
		t.Errorf("found=%v name=%q pid=%d err=%v", found, name, pid, err)
	}

	runTasklist = func(context.Context) ([]byte, error) { return nil, errors.New("tasklist missing") }
	if _, _, _, err := detectProcess(context.Background(), "x"); err == nil {
		t.Error("a failing tasklist must be reported")
	}
	// probeProcess swallows detection errors rather than failing the scan.
	got, err := newTestScanner().probeProcess(context.Background(), processPattern{pattern: "x", name: "X"})
	if err != nil || got != nil {
		t.Errorf("probeProcess: %v %v", got, err)
	}
}

func TestProbeCLIAndProbeProcess_HonourCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := newTestScanner()
	if _, err := s.probeCLI(ctx, cliTarget{binary: "sh", name: "sh"}); err == nil {
		t.Error("probeCLI must stop on a cancelled context")
	}
	if _, err := s.probeProcess(ctx, processPattern{pattern: "x"}); err == nil {
		t.Error("probeProcess must stop on a cancelled context")
	}
}

func TestProbeCLI_FindsBinaryOnPath(t *testing.T) {
	bin := t.TempDir()
	name := "nexus-fake-cli"
	writeFile(t, filepath.Join(bin, name), "#!/bin/sh\n")
	if err := os.Chmod(filepath.Join(bin, name), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	got, err := newTestScanner().probeCLI(context.Background(), cliTarget{binary: name, name: "Fake", kind: domain.ProviderKindCLI})
	if err != nil || len(got) != 1 || got[0].CLIPath == "" || got[0].Status != domain.DiscoveryStatusInstalled {
		t.Errorf("%+v %v", got, err)
	}
	if got, _ := newTestScanner().probeCLI(context.Background(), cliTarget{binary: "definitely-not-installed-xyz"}); got != nil {
		t.Errorf("missing binary: %v", got)
	}
}

func TestProbeProcess_ReportsRunningProcess(t *testing.T) {
	stubGOOS(t, "linux")
	stubPgrep(t, func(string) ([]byte, error) { return []byte("5 /opt/Claude\n"), nil })
	got, err := newTestScanner().probeProcess(context.Background(), processPattern{pattern: "Claude", name: "Claude", kind: domain.ProviderKindDesktopApp})
	if err != nil || len(got) != 1 || got[0].Status != domain.DiscoveryStatusRunning || got[0].ProcessName != "/opt/Claude" || got[0].ID != "proc-claude" {
		t.Errorf("%+v %v", got, err)
	}
}

func TestProbeMCPPortList_IdentifiesMCPServers(t *testing.T) {
	port := serverPort(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/mcp" {
			fmt.Fprint(w, `{"result":{"serverInfo":{"name":"my-mcp"}}}`)
			return
		}
		http.NotFound(w, r)
	})
	notMCP := serverPort(t, func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, `{"result":{}}`) })
	hangup := func() int {
		l, _ := net.Listen("tcp", "127.0.0.1:0")
		go func() {
			for {
				c, err := l.Accept()
				if err != nil {
					return
				}
				c.Close() // accepts then drops: the HTTP request errors
			}
		}()
		t.Cleanup(func() { l.Close() })
		return l.Addr().(*net.TCPAddr).Port
	}()
	closed := func() int {
		l, _ := net.Listen("tcp", "127.0.0.1:0")
		p := l.Addr().(*net.TCPAddr).Port
		l.Close()
		return p
	}()

	got := probeMCPPortList(context.Background(), []int{port, notMCP, hangup, closed})
	if len(got) != 1 || got[0].Name != "my-mcp" || got[0].ID != fmt.Sprintf("mcp-%d", port) || !got[0].IsRunning {
		t.Errorf("got %+v", got)
	}
}
