package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/spf13/cobra"
)

// newPolicyCmd returns the `nexus policy` command: show or change a project's .nexus/policy.json
// (the routing and token settings: ladder, budgets, quota threshold, context and history budgets).
func newPolicyCmd(orch ports.Orchestrator) *cobra.Command {
	cmd := &cobra.Command{Use: "policy", Short: "Show or change a project's routing and token policy (.nexus/policy.json)"}
	cmd.AddCommand(newPolicyShowCmd(orch), newPolicySetCmd(orch))
	return cmd
}

func policyManager(orch ports.Orchestrator) (ports.PolicyManager, error) {
	pm, ok := orch.(ports.PolicyManager)
	if !ok {
		return nil, fmt.Errorf("policy: policy management is not available")
	}
	return pm, nil
}

func newPolicyShowCmd(orch ports.Orchestrator) *cobra.Command {
	var project string
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Print the project's policy as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			pm, err := policyManager(orch)
			if err != nil {
				return err
			}
			pol, found, err := pm.GetProjectPolicy(cmd.Context(), project)
			if err != nil {
				return fmt.Errorf("policy: %w", err)
			}
			out := cmd.OutOrStdout()
			if !found {
				fmt.Fprintln(out, "no policy file: the defaults apply (create one with `policy set`)")
				return nil
			}
			b, _ := json.MarshalIndent(pol, "", "  ")
			fmt.Fprintln(out, string(b))
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "absolute path of the project (required)")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

func newPolicySetCmd(orch ports.Orchestrator) *cobra.Command {
	var project string
	var threshold float64
	var contextBudget, historyBudget int
	var budgets, ladders []string
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Change fields of the project's policy; fields not given stay as they are",
		Example: `  nexus policy set --project /work/app --budget "Claude (subscription)=300000"
  nexus policy set --project /work/app --ladder coder=Ollama/*,Anthropic/claude-sonnet*
  nexus policy set --project /work/app --quota-threshold 0.8 --context-budget 8000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			f := cmd.Flags()
			if !f.Changed("quota-threshold") && !f.Changed("context-budget") && !f.Changed("history-budget") &&
				len(budgets) == 0 && len(ladders) == 0 {
				return fmt.Errorf("policy: nothing to change: give --quota-threshold, --context-budget, --history-budget, --budget or --ladder")
			}
			pm, err := policyManager(orch)
			if err != nil {
				return err
			}
			pol, _, err := pm.GetProjectPolicy(cmd.Context(), project)
			if err != nil {
				return fmt.Errorf("policy: %w", err)
			}
			if f.Changed("quota-threshold") {
				pol.QuotaThreshold = threshold
			}
			if f.Changed("context-budget") {
				pol.ContextBudget = contextBudget
			}
			if f.Changed("history-budget") {
				pol.HistoryBudget = historyBudget
			}
			if err := applyBudgets(&pol, budgets); err != nil {
				return err
			}
			if err := applyLadders(&pol, ladders); err != nil {
				return err
			}
			if err := pm.SetProjectPolicy(cmd.Context(), project, pol); err != nil {
				return fmt.Errorf("policy: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Policy of %s saved; it applies to the next task.\n", project)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&project, "project", "", "absolute path of the project (required)")
	f.Float64Var(&threshold, "quota-threshold", 0, "plan share (0 to 1) at which routing prefers the other providers")
	f.IntVar(&contextBudget, "context-budget", 0, "token cap for a task's context files (0: no cap)")
	f.IntVar(&historyBudget, "history-budget", 0, "token cap for the session history sent with a task (0: newest messages only)")
	f.StringArrayVar(&budgets, "budget", nil, `daily token budget "Provider=tokens" (repeatable; =0 removes it)`)
	f.StringArrayVar(&ladders, "ladder", nil, `model ladder "role=pattern,pattern" (repeatable; empty patterns remove the role)`)
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

// applyBudgets sets or removes ("=0") the daily budgets given as "Provider=tokens".
func applyBudgets(pol *domain.ProjectPolicy, specs []string) error {
	for _, spec := range specs {
		name, tokens, ok := strings.Cut(spec, "=")
		n, err := strconv.Atoi(strings.TrimSpace(tokens))
		if !ok || strings.TrimSpace(name) == "" || err != nil || n < 0 {
			return fmt.Errorf("policy: --budget %q must look like \"Provider=tokens\"", spec)
		}
		name = strings.TrimSpace(name)
		if n == 0 {
			delete(pol.Budgets, name)
			continue
		}
		if pol.Budgets == nil {
			pol.Budgets = map[string]domain.ProviderBudget{}
		}
		pol.Budgets[name] = domain.ProviderBudget{DailyTokens: n}
	}
	return nil
}

// applyLadders sets or removes (empty list) the ladders given as "role=pattern,pattern".
func applyLadders(pol *domain.ProjectPolicy, specs []string) error {
	for _, spec := range specs {
		role, patterns, ok := strings.Cut(spec, "=")
		role = strings.TrimSpace(role)
		if !ok || role == "" {
			return fmt.Errorf("policy: --ladder %q must look like \"role=pattern,pattern\"", spec)
		}
		var rungs []string
		for _, p := range strings.Split(patterns, ",") {
			if p = strings.TrimSpace(p); p != "" {
				rungs = append(rungs, p)
			}
		}
		if len(rungs) == 0 {
			delete(pol.Ladder, role)
			continue
		}
		if pol.Ladder == nil {
			pol.Ladder = map[string][]string{}
		}
		pol.Ladder[role] = rungs
	}
	return nil
}
