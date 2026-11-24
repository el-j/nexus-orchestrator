package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/spf13/cobra"
)

// autoWaitInterval controls how often `nexus auto start --wait` polls the daemon.
var autoWaitInterval = 50 * time.Millisecond

// newAutoCmd returns the `nexus auto` command group for driving Project Auto-Mode.
func newAutoCmd(orch ports.Orchestrator) *cobra.Command {
	auto := &cobra.Command{
		Use:   "auto",
		Short: "Drive Project Auto-Mode (autonomous Brain -> Plan -> Execute -> Verify -> Learn loop)",
	}
	auto.AddCommand(newAutoStartCmd(orch), newAutoStatusCmd(orch), newAutoStopCmd(orch))
	return auto
}

// newAutoStartCmd returns `nexus auto start [goal-or-spec]`.
func newAutoStartCmd(orch ports.Orchestrator) *cobra.Command {
	var (
		projectPath string
		verifyCmd   string
		maxTasks    int
		maxTurns    int
		maxRepairs  int
		wait        bool
		asJSON      bool
	)
	cmd := &cobra.Command{
		Use:   "start [goal-or-spec-file]",
		Short: "Start Auto-Mode for a project (decomposes goal or executes backlog)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, ok := orch.(ports.AutoModeManager)
			if !ok {
				return fmt.Errorf("auto: auto-mode is not available")
			}
			goal := ""
			if len(args) == 1 {
				resolved, err := resolveInstruction(args[0])
				if err != nil {
					return err
				}
				goal = resolved
			}
			absProj, _ := filepath.Abs(projectPath)
			run, err := mgr.StartAutoMode(cmd.Context(), domain.AutoModeRequest{
				ProjectPath:       absProj,
				Goal:              goal,
				VerifyCommand:     verifyCmd,
				MaxTasks:          maxTasks,
				MaxAgentTurns:     maxTurns,
				MaxRepairAttempts: maxRepairs,
			})
			if err != nil {
				return fmt.Errorf("auto start: %w", err)
			}
			if wait {
				run = waitForAutoRun(cmd, mgr, run)
			}
			return printAutoRun(cmd.OutOrStdout(), run, asJSON)
		},
	}
	cmd.Flags().StringVarP(&projectPath, "project", "p", ".", "Target project path")
	cmd.Flags().StringVarP(&verifyCmd, "verify", "v", "", "Final verification command (defaults to .nexus/policy.json)")
	cmd.Flags().IntVar(&maxTasks, "max-tasks", 10, "Maximum tasks in this Auto-Mode run")
	cmd.Flags().IntVar(&maxTurns, "max-turns", 8, "Maximum agentic turns per task")
	cmd.Flags().IntVar(&maxRepairs, "max-repairs", 2, "Maximum automatic repair attempts on final verify failure")
	cmd.Flags().BoolVarP(&wait, "wait", "w", false, "Wait until Auto-Mode settles")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output JSON")
	return cmd
}

// newAutoStatusCmd returns `nexus auto status [run-id]`.
func newAutoStatusCmd(orch ports.Orchestrator) *cobra.Command {
	var (
		projectPath string
		asJSON      bool
	)
	cmd := &cobra.Command{
		Use:   "status [run-id]",
		Short: "Show Auto-Mode run status",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, ok := orch.(ports.AutoModeManager)
			if !ok {
				return fmt.Errorf("auto: auto-mode is not available")
			}
			if len(args) == 1 {
				run, err := mgr.GetAutoModeRun(args[0])
				if err != nil {
					return fmt.Errorf("auto status: %w", err)
				}
				return printAutoRun(cmd.OutOrStdout(), run, asJSON)
			}
			filter := ""
			if projectPath != "" {
				filter, _ = filepath.Abs(projectPath)
			}
			runs := mgr.ListAutoModeRuns(filter)
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(runs)
			}
			if len(runs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No Auto-Mode runs found.")
				return nil
			}
			for _, r := range runs {
				fmt.Fprintf(cmd.OutOrStdout(), "[%-10s] %s  (%d/%d tasks)  %s\n", r.Phase, r.ID, r.CompletedTasks, len(r.TaskIDs), r.ProjectPath)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&projectPath, "project", "p", "", "Filter by project path")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output JSON")
	return cmd
}

// newAutoStopCmd returns `nexus auto stop <run-id>`.
func newAutoStopCmd(orch ports.Orchestrator) *cobra.Command {
	return &cobra.Command{
		Use:   "stop <run-id>",
		Short: "Stop an active Auto-Mode run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, ok := orch.(ports.AutoModeManager)
			if !ok {
				return fmt.Errorf("auto: auto-mode is not available")
			}
			if err := mgr.StopAutoMode(args[0]); err != nil {
				return fmt.Errorf("auto stop: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Auto-Mode run %s stopped.\n", args[0])
			return nil
		},
	}
}

// waitForAutoRun polls mgr until the run reaches a terminal phase or cmd.Context() ends.
func waitForAutoRun(cmd *cobra.Command, mgr ports.AutoModeManager, run domain.AutoModeRun) domain.AutoModeRun {
	ticker := time.NewTicker(autoWaitInterval)
	defer ticker.Stop()
	for !run.Phase.IsTerminal() {
		select {
		case <-cmd.Context().Done():
			return run
		case <-ticker.C:
			if next, err := mgr.GetAutoModeRun(run.ID); err == nil {
				run = next
			}
		}
	}
	return run
}

// printAutoRun formats an AutoModeRun as JSON or a concise summary line.
func printAutoRun(w io.Writer, run domain.AutoModeRun, asJSON bool) error {
	if asJSON {
		if err := json.NewEncoder(w).Encode(run); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(w, "Auto-Mode %s [%s] project=%s tasks=%d (done=%d failed=%d)\n",
			run.ID, run.Phase, run.ProjectPath, len(run.TaskIDs), run.CompletedTasks, run.FailedTasks)
		if run.Summary != "" {
			fmt.Fprintln(w, run.Summary)
		}
		if run.Error != "" {
			fmt.Fprintf(w, "error: %s\n", run.Error)
		}
	}
	if run.Phase == domain.AutoPhaseFailed {
		return fmt.Errorf("auto-mode failed: %s", run.Error)
	}
	return nil
}
