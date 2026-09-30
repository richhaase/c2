package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/display"
	"github.com/richhaase/c2/internal/envelope"
	"github.com/richhaase/c2/internal/goals"
	"github.com/richhaase/c2/internal/report"
	"github.com/richhaase/c2/internal/stats"
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
	var legacy bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show progress toward your distance goal",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, p, workouts, err := loadWorkouts(cmd)
			if err != nil {
				return err
			}
			if !legacy {
				o, err := report.BuildOverview(cfg, p, workouts, cfg.Now(), 4)
				if err != nil {
					return err
				}
				if asJSON {
					return envelope.Print(cmd.OutOrStdout(), "c2.status.v2", statusOverview{Period: o.Period, Summary: o.Summary, Freshness: o.Freshness, Goals: o.Goals})
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Activity %s through %s (%s): %d m, %d workouts, %d training days\n", o.Period.From, o.Period.To, o.Period.Timezone, o.Summary.Meters, o.Summary.Workouts, o.Summary.TrainingDays)
				fmt.Fprintf(cmd.OutOrStdout(), "Last successful sync: %s; latest workout: %s\n", orUnknown(o.Freshness.LastSync), orUnknown(o.Freshness.LatestWorkout))
				printGoals(cmd, o.Goals)
				return nil
			}
			cfg, err = legacyGoalConfig(cfg, p)
			if err != nil {
				return err
			}
			now := cfg.Now()
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
			fmt.Fprintf(out, "Goal start: %s\n", cfg.Goal.StartDate)
			fmt.Fprintf(out, "Progress: %s / %s (%s)\n",
				display.FormatMeters(goal.TotalMeters),
				display.FormatMeters(goal.Target),
				display.FormatPercent(goal.Progress))
			fmt.Fprintf(out, "Weeks elapsed: %d / %d\n", goal.WeeksElapsed, goal.TotalWeeks)
			if goal.RemainingWeeks == 0 {
				fmt.Fprintln(out, "Required pace: goal window ended")
			} else {
				fmt.Fprintf(out, "Required pace: %s\n", display.FormatMetersPerWeek(goal.RequiredPace))
			}
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
	cmd.Flags().BoolVar(&legacy, "legacy", false, "use the original single-goal c2.status.v1 output")
	return cmd
}

type statusOverview struct {
	Period    report.ActivityPeriod  `json:"period"`
	Summary   report.ActivitySummary `json:"summary"`
	Freshness report.Freshness       `json:"freshness"`
	Goals     []goals.Progress       `json:"goals"`
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}
