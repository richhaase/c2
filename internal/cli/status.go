package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/display"
	"github.com/richhaase/c2/internal/envelope"
	"github.com/richhaase/c2/internal/stats"
	"github.com/richhaase/c2/internal/storage"
)

type weekPayload struct {
	WeekStart string `json:"week_start"`
	Meters    int    `json:"meters"`
	Sessions  int    `json:"sessions"`
}

func newWeekPayload(weekStart time.Time, meters, sessions int) weekPayload {
	return weekPayload{
		WeekStart: stats.LocalYMD(weekStart),
		Meters:    meters,
		Sessions:  sessions,
	}
}

type statusPayload struct {
	Goal        stats.GoalProgress `json:"goal"`
	ThisWeek    weekPayload        `json:"this_week"`
	RecentWeeks []weekPayload      `json:"recent_weeks"`
}

func newStatusCmd() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show progress toward your distance goal",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, p, err := loadStore()
			if err != nil {
				return err
			}
			if cfg.Goal.StartDate == "" || cfg.Goal.EndDate == "" {
				return reportf(cmd, "Goal dates not configured. Run `c2 setup` to set start and end dates.")
			}

			workouts, err := storage.ReadWorkouts(p)
			if err != nil {
				return err
			}
			now := time.Now()
			goal, err := stats.ComputeGoalProgress(workouts, cfg, now)
			if err != nil {
				return err
			}
			weeks := stats.RecentWeeks(workouts, now, 4)
			thisWeek := weeks[0]

			out := cmd.OutOrStdout()
			if !asJSON && len(workouts) == 0 {
				fmt.Fprintln(out, "No workouts found. Run `c2 sync` first.")
				return nil
			}

			if asJSON {
				recent := make([]weekPayload, 0, len(weeks))
				for _, w := range weeks {
					recent = append(recent, newWeekPayload(w.WeekStart, w.Meters, w.Sessions))
				}
				return envelope.Print(out, "c2.status.v1", statusPayload{
					Goal:        goal,
					ThisWeek:    newWeekPayload(thisWeek.WeekStart, thisWeek.Meters, thisWeek.Sessions),
					RecentWeeks: recent,
				})
			}

			fmt.Fprintf(out, "Goal: %sm\n", display.FormatMeters(goal.Target))
			fmt.Fprintf(out, "Season start: %s\n", cfg.Goal.StartDate)
			fmt.Fprintf(out, "Progress: %s / %s (%s)\n",
				display.FormatMeters(goal.TotalMeters),
				display.FormatMeters(goal.Target),
				display.FormatPercent(goal.Progress))
			fmt.Fprintf(out, "Weeks elapsed: %d / %d\n", goal.WeeksElapsed, goal.TotalWeeks)
			fmt.Fprintf(out, "Required pace: %s\n", display.FormatMetersPerWeek(goal.RequiredPace))
			fmt.Fprintf(out, "This week so far: %s (%d sessions)\n",
				display.FormatMeters(thisWeek.Meters), thisWeek.Sessions)
			fmt.Fprintln(out)

			fmt.Fprintln(out, "Last 4 weeks:")
			for _, w := range weeks {
				fmt.Fprintf(out, "  Week of %02d/%02d: %s (%d sessions)\n",
					int(w.WeekStart.Month()), w.WeekStart.Day(),
					display.FormatMeters(w.Meters), w.Sessions)
			}
			fmt.Fprintln(out)

			if goal.WeeksElapsed > 0 {
				indicator := "behind pace ✗"
				if goal.OnPace {
					indicator = "on pace ✓"
				}
				fmt.Fprintf(out, "Current avg: %s — %s\n",
					display.FormatMetersPerWeek(goal.CurrentAvgPace), indicator)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}
