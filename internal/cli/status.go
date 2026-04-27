package cli

import (
	"fmt"

	"github.com/richhaase/c2/internal/display"
	"github.com/richhaase/c2/internal/model"
	"github.com/richhaase/c2/internal/stats"
	"github.com/spf13/cobra"
)

func newStatusCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show progress toward your distance goal",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
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
			out := cmd.OutOrStdout()
			if len(workouts) == 0 {
				_, err := fmt.Fprintln(out, "No workouts found. Run `c2 sync` first.")
				return err
			}

			now := deps.Now()
			goal := stats.ComputeGoalProgress(workouts, cfg.Goal, now)
			if _, err := fmt.Fprintf(out, "Goal: %sm\n", display.FormatMeters(goal.Target)); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "Season start: %s\n", cfg.Goal.StartDate); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "Progress: %s / %s (%s)\n", display.FormatMeters(goal.TotalMeters), display.FormatMeters(goal.Target), display.FormatPercent(goal.Progress)); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "Weeks elapsed: %d / %d\n", goal.WeeksElapsed, goal.TotalWeeks); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "Required pace: %s\n", display.FormatMetersPerWeek(goal.RequiredPace)); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(out, "Last 4 weeks:"); err != nil {
				return err
			}

			for i := 0; i < 4; i++ {
				weekStart := stats.MondayOf(now).AddDate(0, 0, -i*7)
				weekEnd := weekStart.AddDate(0, 0, 7)
				weekWorkouts := stats.WorkoutsInRange(workouts, weekStart, weekEnd)
				meters := 0
				days := map[string]struct{}{}
				for _, workout := range weekWorkouts {
					meters += workout.Distance
					days[model.CalendarDay(workout)] = struct{}{}
				}
				if _, err := fmt.Fprintf(out, "  Week of %s: %s (%d sessions)\n", weekStart.Format("01/02"), display.FormatMeters(meters), len(days)); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}

			if goal.WeeksElapsed > 0 {
				indicator := "behind pace x"
				if goal.OnPace {
					indicator = "on pace"
				}
				if _, err := fmt.Fprintf(out, "Current avg: %s - %s\n", display.FormatMetersPerWeek(goal.CurrentAvgPace), indicator); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
