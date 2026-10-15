// Package connect points the AI agents installed on this machine at the Nexus MCP server.
//
// An agent that cannot call Nexus tools can only work around it, for example by editing the
// database or orchestrator.json by hand, which is how task states end up that Nexus never ran.
// Run finds the agents it knows (Antigravity, Cursor, Claude Code), shows which of them are not
// connected and, with Write, connects them: the MCP config file is merged (other servers and
// keys stay, the old file is kept next to it as a backup) or the agent's own command is run.
package connect

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Options tells Run where to look, what to write and how to talk to the outside.
type Options struct {
	// Home is the user's home directory.
	Home string
	// MCPURL is the daemon's MCP endpoint, for example http://127.0.0.1:63988/mcp.
	MCPURL string
	// StdioPath is the nexus-mcp-stdio binary, for agents that start MCP servers as commands;
	// empty when it was not found.
	StdioPath string
	// Write applies the changes; without it Run only reports.
	Write bool
	// Only limits Run to one agent by name; empty means every known agent.
	Only string
	// Out receives the report.
	Out io.Writer
	// Exec runs a command and returns its output (injected for tests).
	Exec func(name string, args ...string) (string, error)
	// HasCommand reports whether a command is on the PATH (injected for tests).
	HasCommand func(name string) bool
}

// agents are the names Run knows, in the order they are reported.
var agents = []string{"antigravity", "cursor", "claude"}

// Run reports (and with Options.Write applies) the connection of every known agent. It returns
// an error when a config file could not be read or written, after it has handled the others.
func Run(o Options) error {
	if o.Only != "" {
		known := false
		for _, a := range agents {
			known = known || a == o.Only
		}
		if !known {
			return fmt.Errorf("unknown agent %q (known: %s)", o.Only, strings.Join(agents, ", "))
		}
	}
	var errs []error
	for _, name := range agents {
		if o.Only != "" && o.Only != name {
			continue
		}
		var err error
		switch name {
		case "antigravity":
			err = jsonAgent(o, name, antigravityPath(o.Home), func() (map[string]any, string) {
				if o.StdioPath == "" {
					return nil, "nexus-mcp-stdio not found (run `make build`, or put it on the PATH)"
				}
				return map[string]any{"command": o.StdioPath, "args": []string{}, "env": map[string]string{"NEXUS_MCP_URL": o.MCPURL}}, ""
			})
		case "cursor":
			err = jsonAgent(o, name, existing(filepath.Join(o.Home, ".cursor", "mcp.json"), filepath.Join(o.Home, ".cursor")),
				func() (map[string]any, string) { return map[string]any{"url": o.MCPURL}, "" })
		case "claude":
			err = claudeAgent(o)
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	if o.Write {
		fmt.Fprintln(o.Out, "\nRestart the agent (or reload its window) so it picks up the new MCP server.")
	}
	return errors.Join(errs...)
}

// antigravityPath is the MCP config of Antigravity: the file in ~/.gemini/antigravity (often a
// symlink to ~/.gemini/config), else the one in ~/.gemini/config. "" when Antigravity is not there.
func antigravityPath(home string) string {
	for _, cand := range [][2]string{
		{filepath.Join(home, ".gemini", "antigravity", "mcp_config.json"), filepath.Join(home, ".gemini", "antigravity")},
		{filepath.Join(home, ".gemini", "config", "mcp_config.json"), filepath.Join(home, ".gemini", "config")},
	} {
		if p := existing(cand[0], cand[1]); p != "" {
			return p
		}
	}
	return ""
}

// existing returns file when the file or its directory exists (the agent is installed), else "".
func existing(file, dir string) string {
	for _, p := range []string{file, dir} {
		if _, err := os.Stat(p); err == nil {
			return file
		}
	}
	return ""
}

func tilde(home, p string) string {
	if rel, err := filepath.Rel(home, p); err == nil && !strings.HasPrefix(rel, "..") {
		return "~/" + filepath.ToSlash(rel)
	}
	return p
}

// jsonAgent connects an agent whose MCP servers live under "mcpServers" in a JSON file.
// entry builds the server entry, or says why it cannot.
func jsonAgent(o Options, name, path string, entry func() (map[string]any, string)) error {
	label := fmt.Sprintf("%-12s", name)
	if path == "" {
		fmt.Fprintf(o.Out, "%s not found (not installed)\n", label)
		return nil
	}
	shown := tilde(o.Home, path)
	// Writing replaces the file, which would turn a symlink into a plain file: write its target.
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	raw, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(o.Out, "%s %s: cannot read: %v\n", label, shown, err)
		return fmt.Errorf("%s: %w", shown, err)
	}
	doc := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &doc); err != nil || doc == nil {
			fmt.Fprintf(o.Out, "%s %s: not valid JSON, left alone (fix or empty the file first)\n", label, shown)
			return fmt.Errorf("%s is not valid JSON", shown)
		}
	}
	want, problem := entry()
	if problem != "" {
		fmt.Fprintf(o.Out, "%s %s: %s\n", label, shown, problem)
		return nil
	}
	servers, _ := doc["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	if cur, ok := servers["nexus"]; ok && sameJSON(cur, want) {
		fmt.Fprintf(o.Out, "%s %s: already connected\n", label, shown)
		return nil
	}
	if !o.Write {
		fmt.Fprintf(o.Out, "%s %s: not connected (run with --write to add the nexus server)\n", label, shown)
		return nil
	}
	servers["nexus"] = want
	doc["mcpServers"] = servers
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("%s: %w", shown, err)
	}
	if len(raw) > 0 {
		if err := os.WriteFile(path+".nexus-backup", raw, 0o600); err != nil {
			return fmt.Errorf("%s: backup: %w", shown, err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("%s: %w", shown, err)
	}
	tmp := path + ".nexus-tmp"
	if err := os.WriteFile(tmp, append(out, '\n'), 0o600); err != nil {
		return fmt.Errorf("%s: %w", shown, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("%s: %w", shown, err)
	}
	fmt.Fprintf(o.Out, "%s %s: connected\n", label, shown)
	return nil
}

// sameJSON compares two JSON values by their encoding.
func sameJSON(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return bytes.Equal(x, y)
}

// claudeAgent connects Claude Code with its own `claude mcp add`.
func claudeAgent(o Options) error {
	label := fmt.Sprintf("%-12s", "claude")
	if !o.HasCommand("claude") {
		fmt.Fprintf(o.Out, "%s not found (the claude command is not on the PATH)\n", label)
		return nil
	}
	if _, err := o.Exec("claude", "mcp", "get", "nexus"); err == nil {
		fmt.Fprintf(o.Out, "%s claude mcp: already connected\n", label)
		return nil
	}
	add := []string{"mcp", "add", "--transport", "http", "--scope", "user", "nexus", o.MCPURL}
	if !o.Write {
		fmt.Fprintf(o.Out, "%s not connected; run with --write, or: claude %s\n", label, strings.Join(add, " "))
		return nil
	}
	if out, err := o.Exec("claude", add...); err != nil {
		fmt.Fprintf(o.Out, "%s `claude mcp add` failed: %v %s\n", label, err, strings.TrimSpace(out))
		return fmt.Errorf("claude mcp add: %w", err)
	}
	fmt.Fprintf(o.Out, "%s connected (user scope)\n", label)
	return nil
}
