package cli

import (
	"fmt"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/spf13/cobra"
)

// newUsageCmd returns the `nexus usage` command: tokens and cost per provider, model and role from
// the daemon's usage ledger.
func newUsageCmd(orch ports.Orchestrator) *cobra.Command {
	var since string
	cmd := &cobra.Command{
		Use:   "usage",
		Short: "Show the tokens and cost the models consumed, per provider, model and role",
		RunE: func(cmd *cobra.Command, args []string) error {
			q, ok := orch.(ports.UsageQuerier)
			if !ok {
				return fmt.Errorf("usage: the usage ledger is not available")
			}
			from, err := domain.ParseSince(since, time.Now())
			if err != nil {
				return fmt.Errorf("usage: %w", err)
			}
			sums, err := q.GetUsage(cmd.Context(), from)
			if err != nil {
				return fmt.Errorf("usage: %w", err)
			}
			out := cmd.OutOrStdout()
			if len(sums) == 0 {
				fmt.Fprintln(out, "No usage recorded in this window.")
				return nil
			}
			fmt.Fprintf(out, "%-22s  %-24s  %-12s  %6s  %10s  %10s  %10s  %9s\n", "Provider", "Model", "Role", "Calls", "Input", "Output", "Cached", "Cost USD")
			for _, s := range sums {
				fmt.Fprintf(out, "%-22s  %-24s  %-12s  %6d  %10d  %10d  %10d  %9.4f\n", s.Provider, s.Model, s.Role, s.Calls, s.InputTokens, s.OutputTokens, s.CachedTokens, s.CostUSD)
			}
			t := domain.UsageTotals(sums)
			fmt.Fprintf(out, "%-22s  %-24s  %-12s  %6d  %10d  %10d  %10d  %9.4f\n", "Total", "", "", t.Calls, t.InputTokens, t.OutputTokens, t.CachedTokens, t.CostUSD)
			return nil
		},
	}
	cmd.Flags().StringVar(&since, "since", "24h", "window: a duration (90m, 24h, 7d), an RFC 3339 time, or all")
	return cmd
}
