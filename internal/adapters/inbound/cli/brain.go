package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

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
