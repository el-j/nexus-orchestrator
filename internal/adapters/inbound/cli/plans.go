package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"nexus-orchestrator/internal/core/ports"

	"github.com/spf13/cobra"
)

func newPlansCmd(orch ports.Orchestrator) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plans",
		Short: "Manage discovered plan and task files",
	}
	cmd.AddCommand(newPlansDiscoveredCmd(orch))
	return cmd
}

func newPlansDiscoveredCmd(orch ports.Orchestrator) *cobra.Command {
	var projectPath string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "discovered",
		Short: "List plan and task files discovered near a project path",
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := orch.GetDiscoveredPlanFiles(cmd.Context(), projectPath)
			if err != nil {
				return fmt.Errorf("plans discovered: %w", err)
			}
			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(files)
			}
			fmt.Printf("%-8s  %-10s  %-8s  %-6s  %s\n", "KIND", "FORMAT", "ACTIVE", "PROJECT", "PATH")
			fmt.Println(strings.Repeat("-", 100))
			for _, f := range files {
				active := "no"
				if f.IsActive {
					active = "yes"
				}
				proj := f.ProjectPath
				if len(proj) > 30 {
					proj = "..." + proj[len(proj)-27:]
				}
				fmt.Printf("%-8s  %-10s  %-8s  %-30s  %s\n", f.Kind, f.Format, active, proj, f.Path)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&projectPath, "project-path", "", "Filter by project path")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}
