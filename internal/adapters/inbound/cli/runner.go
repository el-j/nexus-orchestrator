package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/pkg/runner"

	"github.com/spf13/cobra"
)

var getwd = os.Getwd

// ProviderGenerator adapts a ports.LLMClient to the runner.Generator interface.
type ProviderGenerator struct {
	client ports.LLMClient
}

// NewProviderGenerator wraps an LLMClient for standalone runner execution.
func NewProviderGenerator(c ports.LLMClient) *ProviderGenerator {
	return &ProviderGenerator{client: c}
}

// Generate implements runner.Generator by delegating to GenerateCode.
func (p *ProviderGenerator) Generate(ctx context.Context, prompt string) (string, error) {
	return p.client.GenerateCode(ctx, prompt)
}

// ProviderName returns the identifier of the underlying model provider.
func (p *ProviderGenerator) ProviderName() string {
	return p.client.ProviderName()
}

// NewRunnerStandaloneCmd builds the root cobra command for the standalone nexus-runner CLI.
func NewRunnerStandaloneCmd(gen runner.Generator, defaultProject string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nexus-runner",
		Short: "nexus-runner — Standalone verification gate & task execution engine",
		Long:  "Enforce project quality invariants (file size, test-first) and execute self-healing task correction loops.",
	}

	cmd.AddCommand(newRunnerVerifyCmd(defaultProject))
	cmd.AddCommand(newRunnerExecuteCmd(gen, defaultProject))
	cmd.AddCommand(newRunnerInspectCmd(defaultProject))

	return cmd
}

// newRunnerVerifyCmd constructs the `verify` subcommand.
func newRunnerVerifyCmd(defaultProject string) *cobra.Command {
	var (
		policyPath string
		verifyCmd  string
		fileList   []string
		project    string
		testFirst  bool
		maxSource  int
		maxTest    int
	)

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Enforce project quality invariants and execute verification commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := project
			if root == "" {
				root = defaultProject
			}
			if root == "" {
				root = "."
			}

			pol, err := runner.LoadPolicy(policyPath)
			if err != nil {
				return fmt.Errorf("runner verify: %w", err)
			}

			if maxSource > 0 {
				pol.MaxSourceLines = maxSource
			}
			if maxTest > 0 {
				pol.MaxTestLines = maxTest
			}
			if cmd.Flags().Changed("test-first") {
				pol.EnforceTestFirst = testFirst
			}
			if verifyCmd != "" {
				pol.VerificationCmd = verifyCmd
			}

			out := cmd.OutOrStdout()
			res, err := runner.Verify(cmd.Context(), root, pol, fileList...)
			if err != nil {
				return fmt.Errorf("runner verify: %w", err)
			}

			if len(res.Violations) > 0 {
				fmt.Fprintln(out, "Invariant violations detected:")
				for _, v := range res.Violations {
					fmt.Fprintf(out, "  - [%s] %s: %s\n", v.Rule, v.File, v.Details)
				}
			}

			if res.Command != "" {
				fmt.Fprintf(out, "Verification command %q exited with code %d (%dms)\n", res.Command, res.ExitCode, res.DurationMs)
				if res.Output != "" {
					fmt.Fprintln(out, strings.TrimRight(res.Output, "\n"))
				}
			}

			if !res.Passed {
				return fmt.Errorf("verification gate failed")
			}

			fmt.Fprintln(out, "All verification checks passed.")
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&policyPath, "policy", "", "path to custom policy JSON configuration")
	f.StringVar(&verifyCmd, "cmd", "", "custom verification shell command (e.g. 'go test ./...')")
	f.StringSliceVar(&fileList, "file", nil, "specific file path(s) to verify")
	f.StringVar(&project, "project", "", "project root directory (defaults to current dir)")
	f.BoolVar(&testFirst, "test-first", false, "enforce that tests accompany source modifications")
	f.IntVar(&maxSource, "max-source", 0, "override maximum allowed source lines")
	f.IntVar(&maxTest, "max-test", 0, "override maximum allowed test lines")

	return cmd
}

// newRunnerExecuteCmd constructs the `execute` subcommand.
func newRunnerExecuteCmd(gen runner.Generator, defaultProject string) *cobra.Command {
	var (
		taskID    string
		taskFile  string
		inst      string
		target    string
		verifyCmd string
		heal      bool
		turns     int
		project   string
	)

	cmd := &cobra.Command{
		Use:   "execute",
		Short: "Execute task code generation with self-healing verification",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := project
			if root == "" {
				root = defaultProject
			}
			if root == "" {
				root = "."
			}

			cfg := runner.ExecuteConfig{
				TaskID:          taskID,
				TaskFilePath:    taskFile,
				Instruction:     inst,
				TargetFile:      target,
				VerificationCmd: verifyCmd,
				MaxTurns:        turns,
				Heal:            heal,
				ProjectPath:     root,
			}

			out := cmd.OutOrStdout()
			res, err := runner.ExecuteWithHealing(cmd.Context(), gen, cfg)
			for _, logLine := range res.Logs {
				fmt.Fprintf(out, "runner: %s\n", logLine)
			}

			if err != nil {
				return fmt.Errorf("runner execute: %w", err)
			}

			if !res.Success {
				return fmt.Errorf("runner execute: task failed verification after %d turn(s)", res.Turns)
			}

			fmt.Fprintf(out, "runner execute: task completed successfully in %d turn(s)\n", res.Turns)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&taskID, "task", "", "task identifier")
	f.StringVar(&taskFile, "task-file", "", "path to task markdown file (e.g. .claude/tasks/TASK-NNN.md)")
	f.StringVar(&inst, "instruction", "", "task prompt / instruction")
	f.StringVar(&target, "target", "", "target file path for generated output")
	f.StringVar(&verifyCmd, "cmd", "", "verification command to execute")
	f.BoolVar(&heal, "heal", true, "enable automated self-healing correction turns")
	f.IntVar(&turns, "turns", 2, "maximum self-healing correction turns")
	f.StringVar(&project, "project", "", "project root directory")

	return cmd
}

// newRunnerInspectCmd constructs the `inspect` subcommand.
func newRunnerInspectCmd(defaultProject string) *cobra.Command {
	var (
		project    string
		policyPath string
		asJSON     bool
	)

	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect project repository for file size and quality invariant compliance",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := project
			if root == "" {
				root = defaultProject
			}
			if root == "" {
				cwd, err := getwd()
				if err != nil {
					return fmt.Errorf("get working directory: %w", err)
				}
				root = cwd
			}

			pol, err := runner.LoadPolicy(policyPath)
			if err != nil {
				return fmt.Errorf("runner inspect: %w", err)
			}

			report, err := runner.Inspect(cmd.Context(), root, pol)
			if err != nil {
				return fmt.Errorf("runner inspect: %w", err)
			}

			out := cmd.OutOrStdout()
			if asJSON {
				b, _ := json.MarshalIndent(report, "", "  ")
				fmt.Fprintln(out, string(b))
			} else {
				fmt.Fprintf(out, "Project:      %s\n", report.ProjectPath)
				fmt.Fprintf(out, "Total Files:  %d (source: %d, test: %d)\n", report.TotalFiles, report.SourceFiles, report.TestFiles)
				fmt.Fprintf(out, "Violations:   %d\n", len(report.Violations))
				for _, v := range report.Violations {
					fmt.Fprintf(out, "  - [%s] %s: %s\n", v.Rule, v.File, v.Details)
				}
				if report.GitStatus != "" {
					fmt.Fprintf(out, "Git Status:\n%s\n", report.GitStatus)
				}
				if report.Passed {
					fmt.Fprintln(out, "Status:       PASSED")
				} else {
					fmt.Fprintln(out, "Status:       FAILED")
				}
			}

			if !report.Passed {
				return fmt.Errorf("project has %d invariant violation(s)", len(report.Violations))
			}

			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&project, "project", "", "project root directory (default: current working directory)")
	f.StringVar(&policyPath, "policy", "", "path to custom policy JSON configuration")
	f.BoolVar(&asJSON, "json", false, "format report output as JSON")

	return cmd
}
