package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/spf13/cobra"
)

func newTasksCmd(orch ports.Orchestrator) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "Worker-side task operations (claim, heartbeat, update status)",
	}
	cmd.AddCommand(newTasksClaimCmd(orch))
	cmd.AddCommand(newTasksHeartbeatCmd(orch))
	cmd.AddCommand(newTasksUpdateStatusCmd(orch))
	return cmd
}

func newTasksClaimCmd(orch ports.Orchestrator) *cobra.Command {
	var sessionID string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "claim <task-id>",
		Short: "Claim a queued task for processing",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			task, err := orch.ClaimTask(cmd.Context(), args[0], sessionID)
			if err != nil {
				return fmt.Errorf("tasks claim: %w", err)
			}
			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(task)
			}
			fmt.Printf("Claimed task: %s (status=%s)\n", task.ID, task.Status)
			return nil
		},
	}
	cmd.Flags().StringVar(&sessionID, "session-id", "", "AI session ID claiming the task (required)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	_ = cmd.MarkFlagRequired("session-id")
	return cmd
}

func newTasksHeartbeatCmd(orch ports.Orchestrator) *cobra.Command {
	var sessionID string
	cmd := &cobra.Command{
		Use:   "heartbeat <task-id>",
		Short: "Send a heartbeat for a processing task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := orch.HeartbeatTask(cmd.Context(), args[0], sessionID); err != nil {
				return fmt.Errorf("tasks heartbeat: %w", err)
			}
			fmt.Printf("Heartbeat sent for task: %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&sessionID, "session-id", "", "AI session ID (required)")
	_ = cmd.MarkFlagRequired("session-id")
	return cmd
}

func newTasksUpdateStatusCmd(orch ports.Orchestrator) *cobra.Command {
	var sessionID, status, logs string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "update-status <task-id>",
		Short: "Update the status of a processing task (COMPLETED or FAILED)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			task, err := orch.UpdateTaskStatus(cmd.Context(), args[0], sessionID, domain.TaskStatus(status), logs)
			if err != nil {
				return fmt.Errorf("tasks update-status: %w", err)
			}
			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(task)
			}
			fmt.Printf("Task %s status updated to %s\n", task.ID, task.Status)
			return nil
		},
	}
	cmd.Flags().StringVar(&sessionID, "session-id", "", "AI session ID (required)")
	cmd.Flags().StringVar(&status, "status", "", "New status: COMPLETED or FAILED (required)")
	cmd.Flags().StringVar(&logs, "logs", "", "Optional execution logs")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")
	_ = cmd.MarkFlagRequired("session-id")
	_ = cmd.MarkFlagRequired("status")
	return cmd
}
