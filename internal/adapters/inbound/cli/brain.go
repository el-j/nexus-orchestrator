package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/spf13/cobra"
)

// newBrainCmd returns the `nexus brain` sub-command group.
func newBrainCmd(brain ports.BrainService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "brain",
		Short: "Manage the Nexus Brain project knowledge repository",
	}

	cmd.AddCommand(newBrainStatusCmd(brain))
	cmd.AddCommand(newBrainIngestCmd(brain))
	cmd.AddCommand(newBrainSearchCmd(brain))
	cmd.AddCommand(newBrainInitCmd(brain))
	cmd.AddCommand(newBrainListCmd(brain))
	cmd.AddCommand(newBrainDeleteCmd(brain))
	cmd.AddCommand(newBrainContextCmd(brain))
	cmd.AddCommand(newBrainFileMapCmd(brain))

	return cmd
}

func newBrainStatusCmd(brain ports.BrainService) *cobra.Command {
	var project string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Get the knowledge base status for a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := brain.GetStatus(context.Background(), project)
			if err != nil {
				return fmt.Errorf("cli: brain status: %w", err)
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(status)
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "Project path (required)")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

func newBrainIngestCmd(brain ports.BrainService) *cobra.Command {
	var project string
	var file string
	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Ingest a markdown file (CLAUDE.md) into the project knowledge repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			count, err := brain.IngestFromFile(context.Background(), project, file)
			if err != nil {
				return fmt.Errorf("cli: brain ingest: %w", err)
			}
			fmt.Printf("Ingested %d knowledge sections from %s into project %s\n", count, file, project)
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "Project path (required)")
	cmd.Flags().StringVar(&file, "file", "", "Markdown file path to ingest (required)")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

func newBrainSearchCmd(brain ports.BrainService) *cobra.Command {
	var project string
	var query string
	var limit int
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search the knowledge base for a specific project",
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := brain.SearchKnowledge(context.Background(), project, query, limit)
			if err != nil {
				return fmt.Errorf("cli: brain search: %w", err)
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(results)
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "Project path (required)")
	cmd.Flags().StringVar(&query, "query", "", "Search query (required)")
	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum number of results to return")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("query")
	return cmd
}

func newBrainInitCmd(brain ports.BrainService) *cobra.Command {
	var project string
	var file string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialise the knowledge base for a project by ingesting its CLAUDE.md",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := brain.InitProject(context.Background(), project, file)
			if err != nil {
				return fmt.Errorf("cli: brain init: %w", err)
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(status)
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "Project path (required)")
	cmd.Flags().StringVar(&file, "file", "", "Path to CLAUDE.md (optional; auto-detected if empty)")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

func newBrainListCmd(brain ports.BrainService) *cobra.Command {
	var project string
	var kind string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List knowledge entries for a project, optionally filtered by kind",
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := brain.ListKnowledge(context.Background(), project, kind)
			if err != nil {
				return fmt.Errorf("cli: brain list: %w", err)
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(entries)
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "Project path (required)")
	cmd.Flags().StringVar(&kind, "kind", "", "Knowledge kind filter (optional)")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

func newBrainDeleteCmd(brain ports.BrainService) *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a knowledge entry by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := brain.DeleteKnowledge(context.Background(), id); err != nil {
				return fmt.Errorf("cli: brain delete: %w", err)
			}
			fmt.Printf("deleted %s\n", id)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Knowledge entry ID (required)")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newBrainContextCmd(brain ports.BrainService) *cobra.Command {
	var project string
	var maxTokens int
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Assemble a token-budget-constrained context response for a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := brain.GetContext(context.Background(), domain.ContextQuery{
				ProjectPath: project,
				MaxTokens:   maxTokens,
			})
			if err != nil {
				return fmt.Errorf("cli: brain context: %w", err)
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(resp)
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "Project path (required)")
	cmd.Flags().IntVar(&maxTokens, "max-tokens", 800, "Maximum tokens to include in context")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

func newBrainFileMapCmd(brain ports.BrainService) *cobra.Command {
	var project string
	var focus string
	cmd := &cobra.Command{
		Use:   "file-map",
		Short: "Print the file map entries for a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			paths, err := brain.GetFileMap(context.Background(), project, focus)
			if err != nil {
				return fmt.Errorf("cli: brain file-map: %w", err)
			}
			if len(paths) == 0 {
				fmt.Println("no file map entries")
				return nil
			}
			for _, p := range paths {
				fmt.Println(p)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "Project path (required)")
	cmd.Flags().StringVar(&focus, "focus", "", "Focus area filter (optional)")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}
