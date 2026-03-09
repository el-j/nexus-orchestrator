package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"nexus-ai/internal/core/ports"
)

// NewRootCmd builds and returns the root Cobra command tree.
func NewRootCmd(orch ports.Orchestrator) *cobra.Command {
	root := &cobra.Command{
		Use:   "nexus",
		Short: "NexusAI — Local-first AI orchestrator CLI",
		Long:  "Control the NexusAI daemon: list the queue, submit tasks, and check provider status.",
	}

	root.AddCommand(newQueueCmd(orch))
	root.AddCommand(newProvidersCmd())

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

	queue.AddCommand(list, cancel)
	return queue
}

// newProvidersCmd returns the `nexus providers` command.
func newProvidersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "providers",
		Short: "Show detected LLM providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			// In a full implementation these are read from a status endpoint.
			// For now we surface a user-friendly placeholder.
			status := []map[string]string{
				{"provider": "LM Studio", "url": "http://127.0.0.1:1234", "note": "auto-detected on port 1234"},
				{"provider": "Ollama", "url": "http://127.0.0.1:11434", "note": "auto-detected on port 11434"},
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(status)
		},
	}
}
