// Package cli provides the Cobra-based CLI inbound adapter for nexusOrchestrator.
// It is a thin HTTP client that forwards commands to the daemon API.
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

// NewRootCmd builds and returns the root Cobra command tree.
func NewRootCmd(orch ports.Orchestrator) *cobra.Command {
	root := &cobra.Command{
		Use:   "nexus",
		Short: "nexusOrchestrator — Local-first AI orchestrator CLI",
		Long:  "Control the nexusOrchestrator daemon: list the queue, submit tasks, and check provider status.",
	}

	root.AddCommand(newQueueCmd(orch))
	root.AddCommand(newProvidersCmd(orch))
	root.AddCommand(newDraftCmd(orch))
	root.AddCommand(newBacklogCmd(orch))
	root.AddCommand(newPromoteCmd(orch))
	root.AddCommand(newUpdateCmd(orch))

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

// newDraftCmd returns the `nexus draft` command which creates a DRAFT task.
func newDraftCmd(orch ports.Orchestrator) *cobra.Command {
	var project, instruction, target, provider, model, tags string
	var priority int

	cmd := &cobra.Command{
		Use:   "draft",
		Short: "Create a draft task without queuing it",
		RunE: func(cmd *cobra.Command, args []string) error {
			task := domain.Task{
				ProjectPath:  project,
				Instruction:  instruction,
				TargetFile:   target,
				ProviderName: provider,
				ModelID:      model,
				Priority:     priority,
			}
			if cmd.Flags().Changed("tags") {
				for _, tag := range strings.Split(tags, ",") {
					task.Tags = append(task.Tags, strings.TrimSpace(tag))
				}
			}
			id, err := orch.CreateDraft(task)
			if err != nil {
				return fmt.Errorf("cli: draft: %w", err)
			}
			fmt.Printf("Draft created: %s\n", id)
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Project path (required)")
	cmd.Flags().StringVar(&instruction, "instruction", "", "Task instruction (required)")
	cmd.Flags().StringVar(&target, "target", "", "Target file")
	cmd.Flags().StringVar(&provider, "provider", "", "Provider name")
	cmd.Flags().StringVar(&model, "model", "", "Model ID")
	cmd.Flags().IntVar(&priority, "priority", 2, "Priority (1=highest, 2=medium, 3+=low)")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tags")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("instruction")

	return cmd
}

// newBacklogCmd returns the `nexus backlog` command which lists DRAFT/BACKLOG tasks.
func newBacklogCmd(orch ports.Orchestrator) *cobra.Command {
	var project string

	cmd := &cobra.Command{
		Use:   "backlog",
		Short: "List backlog (DRAFT/BACKLOG) tasks for a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			tasks, err := orch.GetBacklog(project)
			if err != nil {
				return fmt.Errorf("cli: backlog: %w", err)
			}
			if len(tasks) == 0 {
				fmt.Println("No backlog items for project.")
				return nil
			}
			fmt.Printf("%-36s  %8s  %-10s  %-12s  %s\n", "ID", "Priority", "Status", "Provider", "Instruction")
			fmt.Println(strings.Repeat("-", 100))
			for _, t := range tasks {
				instr := t.Instruction
				if len(instr) > 60 {
					instr = instr[:60]
				}
				fmt.Printf("%-36s  %8d  %-10s  %-12s  %s\n",
					t.ID, t.Priority, t.Status, t.ProviderName, instr)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Project path (required)")
	_ = cmd.MarkFlagRequired("project")

	return cmd
}

// newPromoteCmd returns the `nexus promote` command which moves a task to QUEUED.
func newPromoteCmd(orch ports.Orchestrator) *cobra.Command {
	return &cobra.Command{
		Use:   "promote <task-id>",
		Short: "Promote a draft/backlog task to the execution queue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			err := orch.PromoteTask(id)
			if err != nil {
				if strings.Contains(err.Error(), "not found") {
					fmt.Printf("Task %s not found\n", id)
					return nil
				}
				return fmt.Errorf("cli: promote: %w", err)
			}
			fmt.Printf("Task %s promoted to queue\n", id)
			return nil
		},
	}
}

// newUpdateCmd returns the `nexus update` command which patches mutable task fields.
func newUpdateCmd(orch ports.Orchestrator) *cobra.Command {
	var instruction, provider, model, tags string
	var priority int

	cmd := &cobra.Command{
		Use:   "update <task-id>",
		Short: "Update mutable fields on an existing task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			var updates domain.Task
			if cmd.Flags().Changed("instruction") {
				updates.Instruction = instruction
			}
			if cmd.Flags().Changed("provider") {
				updates.ProviderName = provider
			}
			if cmd.Flags().Changed("model") {
				updates.ModelID = model
			}
			if cmd.Flags().Changed("priority") {
				updates.Priority = priority
			}
			if cmd.Flags().Changed("tags") {
				for _, tag := range strings.Split(tags, ",") {
					updates.Tags = append(updates.Tags, strings.TrimSpace(tag))
				}
			}
			updated, err := orch.UpdateTask(id, updates)
			if err != nil {
				return fmt.Errorf("cli: update: %w", err)
			}
			fmt.Printf("Task %s updated: status=%s, priority=%d\n", updated.ID, updated.Status, updated.Priority)
			return nil
		},
	}

	cmd.Flags().StringVar(&instruction, "instruction", "", "New instruction")
	cmd.Flags().StringVar(&provider, "provider", "", "Provider name")
	cmd.Flags().StringVar(&model, "model", "", "Model ID")
	cmd.Flags().IntVar(&priority, "priority", 0, "Priority (1=highest, 2=medium, 3+=low)")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tags")

	return cmd
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
