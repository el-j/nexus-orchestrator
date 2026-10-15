package connect_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nexus-orchestrator/internal/connect"
)

type env struct {
	home string
	opts connect.Options
	out  *bytes.Buffer
	ran  [][]string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{home: t.TempDir(), out: &bytes.Buffer{}}
	e.opts = connect.Options{
		Home:      e.home,
		MCPURL:    "http://127.0.0.1:63988/mcp",
		StdioPath: "/opt/nexus/nexus-mcp-stdio",
		Out:       e.out,
		Exec: func(name string, args ...string) (string, error) {
			e.ran = append(e.ran, append([]string{name}, args...))
			return "", nil
		},
		HasCommand: func(string) bool { return false },
	}
	return e
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func servers(t *testing.T, path string) map[string]map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Servers map[string]map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("%s is not JSON: %v\n%s", path, err, b)
	}
	return doc.Servers
}

// Antigravity's MCP config was an empty file: the agent had no way to call Nexus at all.
func TestConnect_WritesTheAntigravityConfigThroughTheSymlinkAndKeepsOtherServers(t *testing.T) {
	e := newEnv(t)
	real := filepath.Join(e.home, ".gemini", "config", "mcp_config.json")
	write(t, real, "")
	link := filepath.Join(e.home, ".gemini", "antigravity", "mcp_config.json")
	if err := os.MkdirAll(filepath.Dir(link), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}

	// Dry run changes nothing and says what it would do.
	if err := connect.Run(e.opts); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(real); len(b) != 0 {
		t.Fatalf("a dry run must not write: %q", b)
	}
	if !strings.Contains(e.out.String(), "antigravity") || !strings.Contains(e.out.String(), "--write") {
		t.Errorf("dry run output:\n%s", e.out)
	}

	e.opts.Write = true
	if err := connect.Run(e.opts); err != nil {
		t.Fatal(err)
	}
	got := servers(t, real)["nexus"]
	if got["command"] != "/opt/nexus/nexus-mcp-stdio" || got["env"].(map[string]any)["NEXUS_MCP_URL"] != "http://127.0.0.1:63988/mcp" {
		t.Errorf("entry = %v", got)
	}
	if info, _ := os.Lstat(link); info.Mode()&os.ModeSymlink == 0 {
		t.Error("the symlink must stay a symlink")
	}

	// Another server in the file stays, and a second run changes nothing.
	write(t, real, `{"mcpServers":{"other":{"command":"x"}},"keep":true}`)
	for i := 0; i < 2; i++ {
		if err := connect.Run(e.opts); err != nil {
			t.Fatal(err)
		}
	}
	s := servers(t, real)
	if s["other"]["command"] != "x" || s["nexus"]["command"] != "/opt/nexus/nexus-mcp-stdio" {
		t.Errorf("servers = %v", s)
	}
	if b, _ := os.ReadFile(real); !strings.Contains(string(b), `"keep": true`) {
		t.Errorf("unrelated keys must survive: %s", b)
	}
	if _, err := os.Stat(real + ".nexus-backup"); err != nil {
		t.Errorf("the previous file must be kept as a backup: %v", err)
	}
	if !strings.Contains(e.out.String(), "already connected") {
		t.Errorf("the second run must say so:\n%s", e.out)
	}
}

func TestConnect_NeverTouchesAFileItCannotRead(t *testing.T) {
	e := newEnv(t)
	e.opts.Write = true
	p := filepath.Join(e.home, ".gemini", "config", "mcp_config.json")
	write(t, p, "{this is not json")
	if err := connect.Run(e.opts); err == nil || !strings.Contains(err.Error(), "mcp_config.json") {
		t.Errorf("want an error naming the file, got %v", err)
	}
	if b, _ := os.ReadFile(p); string(b) != "{this is not json" {
		t.Errorf("an unreadable config must be left alone: %q", b)
	}
	if !strings.Contains(e.out.String(), "not valid JSON") {
		t.Errorf("output:\n%s", e.out)
	}
}

func TestConnect_CursorUsesTheHTTPEndpointAndAnAbsentAgentIsSkipped(t *testing.T) {
	e := newEnv(t)
	e.opts.Write = true
	e.opts.StdioPath = "" // Cursor needs none
	if err := connect.Run(e.opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(e.out.String(), "not found") {
		t.Errorf("agents that are not installed are listed as such:\n%s", e.out)
	}
	if _, err := os.Stat(filepath.Join(e.home, ".cursor")); err == nil {
		t.Error("nothing may be created for an agent that is not installed")
	}

	write(t, filepath.Join(e.home, ".cursor", "mcp.json"), `{"mcpServers":{}}`)
	if err := connect.Run(e.opts); err != nil {
		t.Fatal(err)
	}
	if got := servers(t, filepath.Join(e.home, ".cursor", "mcp.json"))["nexus"]; got["url"] != "http://127.0.0.1:63988/mcp" {
		t.Errorf("cursor entry = %v", got)
	}
}

func TestConnect_AntigravityNeedsTheStdioProxy(t *testing.T) {
	e := newEnv(t)
	e.opts.Write = true
	e.opts.StdioPath = ""
	write(t, filepath.Join(e.home, ".gemini", "config", "mcp_config.json"), "")
	if err := connect.Run(e.opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(e.out.String(), "nexus-mcp-stdio not found") {
		t.Errorf("output:\n%s", e.out)
	}
	if b, _ := os.ReadFile(filepath.Join(e.home, ".gemini", "config", "mcp_config.json")); len(b) != 0 {
		t.Errorf("nothing may be written without the proxy: %q", b)
	}
}

func TestConnect_ClaudeCodeIsConnectedThroughItsOwnCommand(t *testing.T) {
	e := newEnv(t)
	e.opts.HasCommand = func(name string) bool { return name == "claude" }
	// not connected yet: "claude mcp get nexus" fails
	e.opts.Exec = func(name string, args ...string) (string, error) {
		e.ran = append(e.ran, append([]string{name}, args...))
		if len(args) > 1 && args[1] == "get" {
			return "", os.ErrNotExist
		}
		return "", nil
	}
	if err := connect.Run(e.opts); err != nil {
		t.Fatal(err)
	}
	if len(e.ran) != 1 || strings.Join(e.ran[0], " ") != "claude mcp get nexus" {
		t.Errorf("a dry run only asks: %v", e.ran)
	}
	if !strings.Contains(e.out.String(), "claude mcp add --transport http --scope user nexus http://127.0.0.1:63988/mcp") {
		t.Errorf("the command to run is shown:\n%s", e.out)
	}

	e.ran, e.opts.Write = nil, true
	if err := connect.Run(e.opts); err != nil {
		t.Fatal(err)
	}
	want := "claude mcp add --transport http --scope user nexus http://127.0.0.1:63988/mcp"
	if len(e.ran) != 2 || strings.Join(e.ran[1], " ") != want {
		t.Errorf("ran %v, want the add command", e.ran)
	}
}

func TestConnect_OnlyOneAgent(t *testing.T) {
	e := newEnv(t)
	e.opts.Write = true
	e.opts.Only = "cursor"
	write(t, filepath.Join(e.home, ".gemini", "config", "mcp_config.json"), "")
	write(t, filepath.Join(e.home, ".cursor", "mcp.json"), "{}")
	if err := connect.Run(e.opts); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(e.home, ".gemini", "config", "mcp_config.json")); len(b) != 0 {
		t.Error("only the named agent may be changed")
	}
	if servers(t, filepath.Join(e.home, ".cursor", "mcp.json"))["nexus"] == nil {
		t.Error("cursor was not connected")
	}
	e.opts.Only = "nobody"
	if err := connect.Run(e.opts); err == nil || !strings.Contains(err.Error(), "unknown agent") {
		t.Errorf("an unknown agent is an error: %v", err)
	}
}

func TestConnect_ReportsProblemsWithoutGivingUp(t *testing.T) {
	// a config that is a directory cannot be read
	e := newEnv(t)
	e.opts.Write = true
	if err := os.MkdirAll(filepath.Join(e.home, ".cursor", "mcp.json"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := connect.Run(e.opts); err == nil || !strings.Contains(e.out.String(), "cannot read") {
		t.Errorf("err=%v\n%s", err, e.out)
	}

	// a read-only directory cannot be written to; the others are still handled
	e = newEnv(t)
	e.opts.Write = true
	dir := filepath.Join(e.home, ".cursor")
	write(t, filepath.Join(dir, "mcp.json"), "{}")
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(dir, 0o700) }()
	if os.Geteuid() != 0 {
		if err := connect.Run(e.opts); err == nil {
			t.Error("a config that cannot be written must be an error")
		}
	}

	// claude: already connected is reported, a failing add is an error
	e = newEnv(t)
	e.opts.HasCommand = func(string) bool { return true }
	if err := connect.Run(e.opts); err != nil || !strings.Contains(e.out.String(), "already connected") {
		t.Errorf("err=%v\n%s", err, e.out)
	}
	e = newEnv(t)
	e.opts.Write = true
	e.opts.HasCommand = func(string) bool { return true }
	e.opts.Exec = func(name string, args ...string) (string, error) {
		if args[1] == "get" {
			return "", os.ErrNotExist
		}
		return "boom", os.ErrPermission
	}
	if err := connect.Run(e.opts); err == nil || !strings.Contains(e.out.String(), "failed") {
		t.Errorf("err=%v\n%s", err, e.out)
	}
}
