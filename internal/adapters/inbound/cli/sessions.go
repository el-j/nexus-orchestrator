package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/spf13/cobra"
)

func newAISessionsCmd(orch ports.Orchestrator) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai-sessions",
		Short: "Manage AI agent sessions registered with the daemon",
	}
	cmd.AddCommand(newAISessionsListCmd(orch))
	cmd.AddCommand(newAISessionsRegisterCmd(orch))
	cmd.AddCommand(newAISessionsDeregisterCmd(orch))
	cmd.AddCommand(newAISessionsHeartbeatCmd(orch))
	cmd.AddCommand(newAISessionsTerminateCmd(orch))
	cmd.AddCommand(newAISessionsPurgeCmd(orch))
	cmd.AddCommand(newAISessionsDelegateCmd(orch))
	return cmd
}

func newAISessionsListCmd(orch ports.Orchestrator) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered AI sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			sessions, err := orch.ListAISessions(cmd.Context())
			if err != nil {
				return fmt.Errorf("ai-sessions list: %w", err)
			}
			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(sessions)
			}
			fmt.Printf("%-36s  %-20s  %-10s  %-30s  %s\n", "ID", "AGENT", "STATUS", "PROJECT", "UPDATED")
			fmt.Println(strings.Repeat("-", 110))
			for _, s := range sessions {
				fmt.Printf("%-36s  %-20s  %-10s  %-30s  %s\n",
					s.ID, s.AgentName, s.Status, s.ProjectPath, s.UpdatedAt.Format("2006-01-02 15:04:05"))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	return cmd
}

func newAISessionsRegisterCmd(orch ports.Orchestrator) *cobra.Command {
	var agentName, projectPath, model string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new AI session",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := domain.AISession{
				AgentName:   agentName,
				ProjectPath: projectPath,
				ModelID:     model,
			}
			created, err := orch.RegisterAISession(cmd.Context(), s)
			if err != nil {
				return fmt.Errorf("ai-sessions register: %w", err)
			}
			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(created)
			}
			fmt.Printf("Registered session: %s (agent=%s)\n", created.ID, created.AgentName)
			return nil
		},
	}
	cmd.Flags().StringVar(&agentName, "agent-name", "", "Agent name (required)")
	cmd.Flags().StringVar(&projectPath, "project-path", "", "Project path")
	cmd.Flags().StringVar(&model, "model", "", "Model ID")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	_ = cmd.MarkFlagRequired("agent-name")
	return cmd
}

func newAISessionsDeregisterCmd(orch ports.Orchestrator) *cobra.Command {
	return &cobra.Command{
		Use:   "deregister <id>",
		Short: "Deregister an AI session by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := orch.DeregisterAISession(cmd.Context(), args[0]); err != nil {
				return fmt.Errorf("ai-sessions deregister: %w", err)
			}
			fmt.Printf("Deregistered session: %s\n", args[0])
			return nil
		},
	}
}

func newAISessionsHeartbeatCmd(orch ports.Orchestrator) *cobra.Command {
	return &cobra.Command{
		Use:   "heartbeat <id>",
		Short: "Send a heartbeat for an AI session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := orch.HeartbeatAISession(cmd.Context(), args[0]); err != nil {
				return fmt.Errorf("ai-sessions heartbeat: %w", err)
			}
			fmt.Printf("Heartbeat sent for session: %s\n", args[0])
			return nil
		},
	}
}

func newAISessionsTerminateCmd(orch ports.Orchestrator) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "terminate <id>",
		Short: "Terminate an AI session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := orch.TerminateAISession(cmd.Context(), args[0], force); err != nil {
				return fmt.Errorf("ai-sessions terminate: %w", err)
			}
			fmt.Printf("Terminated session: %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Force kill the session process")
	return cmd
}

func newAISessionsPurgeCmd(orch ports.Orchestrator) *cobra.Command {
	return &cobra.Command{
		Use:   "purge",
		Short: "Purge all disconnected AI sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := orch.PurgeDisconnectedSessions(cmd.Context())
			if err != nil {
				return fmt.Errorf("ai-sessions purge: %w", err)
			}
			fmt.Printf("Purged %d disconnected session(s).\n", n)
			return nil
		},
	}
}

func newAISessionsDelegateCmd(orch ports.Orchestrator) *cobra.Command {
	return &cobra.Command{
		Use:   "delegate <session-id>",
		Short: "Get delegation instructions for an AI session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instruction, err := orch.DelegateToNexus(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("ai-sessions delegate: %w", err)
			}
			fmt.Println(instruction)
			return nil
		},
	}
}
