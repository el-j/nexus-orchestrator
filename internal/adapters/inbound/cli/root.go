package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"nexus-orchestrator/internal/core/ports"

	"github.com/spf13/cobra"
)

// NewRootCmd builds and returns the root Cobra command tree.
func NewRootCmd(orch ports.Orchestrator) *cobra.Command {
	root := &cobra.Command{
		Use:   "nexus",
		Short: "NexusAI — Local-first AI orchestrator CLI",
		Long:  "Control the NexusAI daemon: list the queue, submit tasks, and check provider status.",
	}

	root.AddCommand(newQueueCmd(orch))
	root.AddCommand(newProvidersCmd(orch))

	return root
}

// newQueueCmd returns the `nexus queue` sub-command group.
func newQueueCmd(orch ports.Orchestrator) *cobra.Command {
	queue := &cobra.Command{
		Use:   "queue",
		Short: "Manage the task queue",
	}

	list := &cobra.Command{
		Use:   "list",
		Short: "List all pending and processing tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			tasks, err := orch.GetQueue()
			if err != nil {
				return fmt.Errorf("queue list: %w", err)
			}
			if len(tasks) == 0 {
				fmt.Println("Queue is empty.")
				return nil
			}
			for _, t := range tasks {
				fmt.Printf("[%-10s] %s → %s\n\t%q\n",
					t.Status, t.ProjectPath, t.TargetFile, t.Instruction)
			}
			return nil
		},
	}

	get := &cobra.Command{
		Use:   "get <task-id>",
		Short: "Fetch details for a specific task by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			task, err := orch.GetTask(args[0])
			if err != nil {
				return fmt.Errorf("queue get: %w", err)
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(task)
		},
	}

	cancel := &cobra.Command{
		Use:   "cancel <task-id>",
		Short: "Cancel a queued task by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := orch.CancelTask(args[0]); err != nil {
				return fmt.Errorf("cancel: %w", err)
			}
			fmt.Printf("Task %s cancelled.\n", args[0])
			return nil
		},
	}

	queue.AddCommand(list, get, cancel)
	return queue
}

// newProvidersCmd returns the `nexus providers` command which queries the live
// daemon for the status of each registered LLM backend.
func newProvidersCmd(orch ports.Orchestrator) *cobra.Command {
	return &cobra.Command{
		Use:   "providers",
		Short: "Show the status of all registered LLM backends",
		RunE: func(cmd *cobra.Command, args []string) error {
			providers, err := orch.GetProviders()
			if err != nil {
				return fmt.Errorf("providers: %w", err)
			}
			if len(providers) == 0 {
				fmt.Println("No providers registered.")
				return nil
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(providers)
		},
	}
}
