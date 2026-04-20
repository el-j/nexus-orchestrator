package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/spf13/cobra"
)

func newConfigCmd(orch ports.Orchestrator) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Get or set daemon runtime configuration",
	}
	cmd.AddCommand(newConfigGetCmd(orch))
	cmd.AddCommand(newConfigSetCmd(orch))
	return cmd
}

func newConfigGetCmd(orch ports.Orchestrator) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Show current daemon runtime configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := orch.GetRuntimeConfig(cmd.Context())
			if err != nil {
				return fmt.Errorf("config get: %w", err)
			}
			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(cfg)
			}
			fmt.Printf("queueCap=%d\n", cfg.QueueCap)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newConfigSetCmd(orch ports.Orchestrator) *cobra.Command {
	var queueCap int
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Update daemon runtime configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			update := domain.RuntimeConfigUpdate{}
			if cmd.Flags().Changed("queue-cap") {
				update.QueueCap = &queueCap
			}
			cfg, err := orch.UpdateRuntimeConfig(cmd.Context(), update)
			if err != nil {
				return fmt.Errorf("config set: %w", err)
			}
			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(cfg)
			}
			fmt.Printf("queueCap=%d\n", cfg.QueueCap)
			return nil
		},
	}
	cmd.Flags().IntVar(&queueCap, "queue-cap", 0, "Maximum number of queued tasks")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}
