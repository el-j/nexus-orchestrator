package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"nexus-orchestrator/internal/adapters/inbound/mcp"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/spf13/cobra"
)

// NewBrainCmd returns the `nexus brain` sub-command group for the unified CLI.
func NewBrainCmd(brain ports.BrainService) *cobra.Command {
	return newBrainCmd(brain)
}

// NewBrainStandaloneCmd builds the root command for the standalone nexus-brain binary.
func NewBrainStandaloneCmd(brain ports.BrainService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nexus-brain",
		Short: "Nexus Brain — Standalone RAG, semantic search, and knowledge management",
	}

	cmd.AddCommand(newBrainStatusCmd(brain))
	cmd.AddCommand(newBrainIngestCmd(brain))
	cmd.AddCommand(newBrainSearchCmd(brain))
	cmd.AddCommand(newBrainInitCmd(brain))
	cmd.AddCommand(newBrainListCmd(brain))
	cmd.AddCommand(newBrainDeleteCmd(brain))
	cmd.AddCommand(newBrainContextCmd(brain))
	cmd.AddCommand(newBrainFileMapCmd(brain))
	cmd.AddCommand(newBrainMcpCmd(brain))

	return cmd
}

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
	cmd.AddCommand(newBrainMcpCmd(brain))

	return cmd
}

// newBrainMcpCmd builds "brain mcp": runs a dedicated Brain MCP server over stdio or HTTP.
func newBrainMcpCmd(brain ports.BrainService) *cobra.Command {
	var (
		stdio bool
		port  int
	)
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run a dedicated Nexus Brain MCP server exposing knowledge tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := mcp.NewMcpServer(nil, brain)
			if stdio || port == 0 {
				return srv.ServeStdio(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout())
			}
			addr := fmt.Sprintf("127.0.0.1:%d", port)
			fmt.Fprintf(cmd.ErrOrStderr(), "nexus-brain: serving MCP on http://%s/mcp\n", addr)
			return mcp.StartMCPServer(cmd.Context(), nil, brain, addr)
		},
	}
	cmd.Flags().BoolVar(&stdio, "stdio", true, "Serve MCP over stdio (default)")
	cmd.Flags().IntVar(&port, "port", 0, "Serve MCP over HTTP on the given port (e.g. 63990)")
	return cmd
}

// newBrainStatusCmd builds "brain status": how many entries a project holds and when it was last
// scanned.
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
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(status)
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "Project path (required)")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

// newBrainIngestCmd builds "brain ingest": it stores a markdown file or a single entry in the knowledge
// base.
func newBrainIngestCmd(brain ports.BrainService) *cobra.Command {
	var project, file, kind, topic, content string
	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Ingest a markdown file or a single knowledge entry into the project knowledge repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" && content != "" {
				k := domain.KnowledgeKind(kind)
				if k == "" {
					k = domain.KnowledgeLearning
				}
				entry, err := brain.IngestKnowledge(context.Background(), domain.ProjectKnowledge{
					ProjectPath: project,
					Kind:        k,
					Topic:       topic,
					Content:     content,
				})
				if err != nil {
					return fmt.Errorf("cli: brain ingest: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Ingested knowledge entry %q (%s) into project %s\n", entry.Topic, entry.Kind, project)
				return nil
			}
			if file == "" {
				return fmt.Errorf("cli: brain ingest: --file or --content is required")
			}
			count, err := brain.IngestFromFile(context.Background(), project, file)
			if err != nil {
				return fmt.Errorf("cli: brain ingest: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Ingested %d knowledge sections from %s into project %s\n", count, file, project)
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "Project path (required)")
	cmd.Flags().StringVar(&file, "file", "", "Markdown file path to ingest")
	cmd.Flags().StringVar(&kind, "kind", "", "Knowledge kind (for single entry ingest)")
	cmd.Flags().StringVar(&topic, "topic", "", "Topic title (for single entry ingest)")
	cmd.Flags().StringVar(&content, "content", "", "Markdown content (for single entry ingest)")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

// newBrainSearchCmd builds "brain search": full-text search over a project's knowledge or across all projects.
func newBrainSearchCmd(brain ports.BrainService) *cobra.Command {
	var project string
	var query string
	var limit int
	var all bool
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search the knowledge base for a specific project or across all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			if query == "" && len(args) > 0 {
				query = strings.Join(args, " ")
			}
			if query == "" {
				return fmt.Errorf("cli: brain search: query is required")
			}
			target := project
			if all || target == "" {
				target = "*"
			}
			results, err := brain.SearchKnowledge(context.Background(), target, query, limit)
			if err != nil {
				return fmt.Errorf("cli: brain search: %w", err)
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(results)
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "Project path (empty or '*' searches all projects)")
	cmd.Flags().StringVar(&query, "query", "", "Search query (or provide as arguments)")
	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum number of results to return")
	cmd.Flags().BoolVar(&all, "all", false, "Search across all projects")
	return cmd
}

// newBrainInitCmd builds "brain init": it registers a project and reads its CLAUDE.md into the
// knowledge base.
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
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(status)
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "Project path (required)")
	cmd.Flags().StringVar(&file, "file", "", "Path to CLAUDE.md (optional; auto-detected if empty)")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

// newBrainListCmd builds "brain list": the stored entries of a project, optionally of one kind.
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
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(entries)
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "Project path (required)")
	cmd.Flags().StringVar(&kind, "kind", "", "Knowledge kind filter (optional)")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

// newBrainDeleteCmd builds "brain delete": it removes one knowledge entry by id.
func newBrainDeleteCmd(brain ports.BrainService) *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a knowledge entry by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := brain.DeleteKnowledge(context.Background(), id); err != nil {
				return fmt.Errorf("cli: brain delete: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "deleted %s\n", id)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Knowledge entry ID (required)")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

// newBrainContextCmd builds "brain context": the knowledge that answers a question, within a token
// budget.
func newBrainContextCmd(brain ports.BrainService) *cobra.Command {
	var project, question string
	var maxTokens int
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Assemble a token-budget-constrained context response for a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			q := domain.ContextQuery{
				ProjectPath: project,
				Question:    question,
				MaxTokens:   maxTokens,
			}
			var (
				resp domain.ContextResponse
				err  error
			)
			if question != "" {
				resp, err = brain.GetFocusedContext(context.Background(), q)
			} else {
				resp, err = brain.GetContext(context.Background(), q)
			}
			if err != nil {
				return fmt.Errorf("cli: brain context: %w", err)
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(resp)
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "Project path (required)")
	cmd.Flags().StringVarP(&question, "question", "q", "", "Focused question to rank knowledge entries by relevance (optional)")
	cmd.Flags().IntVar(&maxTokens, "max-tokens", 800, "Maximum tokens to include in context")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

// newBrainFileMapCmd builds "brain file-map": the project's file map, narrowed to a focus area when one
// is given.
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
				fmt.Fprintln(cmd.OutOrStdout(), "no file map entries")
				return nil
			}
			for _, p := range paths {
				fmt.Fprintln(cmd.OutOrStdout(), p)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "Project path (required)")
	cmd.Flags().StringVar(&focus, "focus", "", "Focus area filter (optional)")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}
