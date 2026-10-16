package cli

import (
	"fmt"

	"nexus-orchestrator/internal/core/ports"

	"github.com/spf13/cobra"
)

// newRoutingCmd returns the `nexus routing` command: how the daemon would order the providers for a
// task of a role in a project right now.
func newRoutingCmd(orch ports.Orchestrator) *cobra.Command {
	var project, role string
	cmd := &cobra.Command{
		Use:   "routing",
		Short: "Show the order in which providers would be tried for a role in a project (ladder, cooldowns, quota)",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, ok := orch.(ports.RoutingPreviewer)
			if !ok {
				return fmt.Errorf("routing: the routing preview is not available")
			}
			preview, err := p.PreviewRouting(cmd.Context(), project, role)
			if err != nil {
				return fmt.Errorf("routing: %w", err)
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Role: %s   quota threshold: %.2f\n", orDash(preview.Role), preview.QuotaThreshold)
			if len(preview.Ladder) == 0 {
				fmt.Fprintln(out, "Ladder: none (providers are ordered by role and availability)")
			} else {
				fmt.Fprintf(out, "Ladder (cheapest first): %v\n", preview.Ladder)
			}
			fmt.Fprintf(out, "%3s  %-22s  %-28s  %4s  %s\n", "#", "Provider", "Model", "Rung", "State")
			for _, e := range preview.Order {
				state := "ready"
				switch {
				case !e.CoolingUntil.IsZero():
					state = fmt.Sprintf("cooling down until %s (%s)", e.CoolingUntil.Local().Format("Mon 15:04"), e.CooldownReason)
				case e.Utilization != nil && *e.Utilization >= preview.QuotaThreshold:
					state = fmt.Sprintf("plan %.0f%% used: tried after the others", *e.Utilization*100)
				case e.Utilization != nil:
					state = fmt.Sprintf("ready, plan %.0f%% used", *e.Utilization*100)
				}
				fmt.Fprintf(out, "%3d  %-22s  %-28s  %4s  %s\n", e.Position, e.Provider, e.Model, rungLabel(e.Rung), state)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "absolute path of the project (required)")
	cmd.Flags().StringVar(&role, "role", "", "the task role, for example coder")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func rungLabel(r int) string {
	if r < 0 {
		return "-"
	}
	return fmt.Sprint(r + 1)
}
