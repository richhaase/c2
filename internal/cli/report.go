package cli

import (
	"fmt"
	"path/filepath"

	"github.com/richhaase/c2/internal/report"
	"github.com/spf13/cobra"
)

func newReportCommand(deps Dependencies) *cobra.Command {
	var output string
	var open bool
	var weeks int
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate HTML progress report",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := positiveIntFlag(weeks, "weeks"); err != nil {
				return err
			}
			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}
			if cfg.Goal.StartDate == "" || cfg.Goal.EndDate == "" {
				return fmt.Errorf("goal dates not configured. Run `c2 setup` to set start and end dates")
			}
			workouts, err := deps.ReadWorkouts()
			if err != nil {
				return err
			}
			if len(workouts) == 0 {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "No workouts found. Run `c2 sync` first.")
				return err
			}

			html, err := report.HTML(workouts, cfg.Goal, weeks, deps.Now())
			if err != nil {
				return err
			}
			outPath := output
			if outPath == "" {
				dir, err := deps.TempDir("", "c2-report-")
				if err != nil {
					return err
				}
				outPath = filepath.Join(dir, "report.html")
			}
			absPath, err := filepath.Abs(outPath)
			if err != nil {
				return err
			}
			if err := deps.WriteFile(absPath, []byte(html), 0o644); err != nil {
				return err
			}
			if open {
				if err := deps.OpenFile(absPath); err != nil {
					return fmt.Errorf("could not open report: %w", err)
				}
				return nil
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Report written to: %s\n", absPath)
			return err
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "save to a specific file instead of a temp file")
	cmd.Flags().BoolVar(&open, "open", false, "open report in browser")
	cmd.Flags().IntVarP(&weeks, "weeks", "w", 12, "weeks of history to show")
	return cmd
}
