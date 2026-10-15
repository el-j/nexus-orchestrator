package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"nexus-orchestrator/internal/connect"

	"github.com/spf13/cobra"
)

// newConnectCmd returns `nexus connect`: shows which AI agents on this machine can call the
// Nexus MCP tools and, with --write, connects them.
func newConnectCmd() *cobra.Command {
	var write bool
	var only string
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect the AI agents on this machine (Antigravity, Cursor, Claude Code) to the Nexus MCP server",
		Long: "An agent that cannot call Nexus tools can only work around them, for example by editing the " +
			"database by hand. This finds the agents installed here, shows which are not connected and, with " +
			"--write, adds the nexus MCP server to their config (other servers stay; the old file is kept as a backup).",
		RunE: func(cmd *cobra.Command, _ []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("home directory: %w", err)
			}
			url := strings.TrimSpace(os.Getenv("NEXUS_MCP_URL"))
			if url == "" {
				url = "http://127.0.0.1:63988/mcp"
			}
			return connect.Run(connect.Options{
				Home: home, MCPURL: url, StdioPath: findStdioProxy(), Write: write, Only: only, Out: cmd.OutOrStdout(),
				Exec: func(name string, args ...string) (string, error) {
					out, err := exec.CommandContext(cmd.Context(), name, args...).CombinedOutput()
					return string(out), err
				},
				HasCommand: func(name string) bool { _, err := exec.LookPath(name); return err == nil },
			})
		},
	}
	cmd.Flags().BoolVar(&write, "write", false, "apply the changes (without it, only report)")
	cmd.Flags().StringVar(&only, "agent", "", "connect only this agent: antigravity, cursor or claude")
	return cmd
}

// findStdioProxy locates nexus-mcp-stdio next to this binary, else on the PATH; "" when missing.
func findStdioProxy() string {
	if self, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(self); err == nil {
			self = resolved
		}
		cand := filepath.Join(filepath.Dir(self), "nexus-mcp-stdio")
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	if p, err := exec.LookPath("nexus-mcp-stdio"); err == nil {
		return p
	}
	return ""
}
