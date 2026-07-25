package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/analysis"
	"github.com/richhaase/c2/internal/display"
	"github.com/richhaase/c2/internal/envelope"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/stats"
	"github.com/richhaase/c2/internal/storage"
)

func jsNumber(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func orDash[T any](v *T, format func(T) string) string {
	if v == nil {
		return "-"
	}
	return format(*v)
}

type weeklyPayload struct {
	Weeks []stats.WeekSummaryData `json:"weeks"`
}

type goalPayload struct {
	Goal       stats.GoalProgress   `json:"goal"`
	Projection stats.GoalProjection `json:"projection"`
	ThisWeek   weekPayload          `json:"this_week"`
}

type splitsPayload struct {
	WorkoutID  int64               `json:"workout_id"`
	Date       string              `json:"date"`
	Distance   int                 `json:"distance"`
	SplitShape analysis.Shape      `json:"split_shape"`
	Splits     []analysis.SplitRow `json:"splits"`
}

type hrPacePayload struct {
	Weeks int                   `json:"weeks"`
	Bands []analysis.HRPaceBand `json:"bands"`
}

func newStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Derived training statistics",
	}
	cmd.AddCommand(
		newStatsWeeklyCmd(),
		newStatsGoalCmd(),
		newStatsSplitsCmd(),
		newStatsHRPaceCmd(),
	)
	return cmd
}

func newStatsWeeklyCmd() *cobra.Command {
	var (
		weeksFlag string
		asJSON    bool
	)
	cmd := &cobra.Command{
		Use:   "weekly",
		Short: "Weekly volume, sessions, pace, SPM, and HR",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			weeks, err := positiveInt(cmd, weeksFlag, "Error: --weeks must be a positive integer.")
			if err != nil {
				return err
			}
			_, p, err := loadStore()
			if err != nil {
				return err
			}
			workouts, err := storage.ReadWorkouts(p)
			if err != nil {
				return err
			}
			summaries := make([]stats.WeekSummaryData, 0, weeks)
			for _, ws := range stats.BuildWeekSummaries(workouts, time.Now(), weeks) {
				summaries = append(summaries, stats.WeekSummaryDataOf(ws))
			}

			out := cmd.OutOrStdout()
			if asJSON {
				return envelope.Print(out, "c2.stats.weekly.v1", weeklyPayload{Weeks: summaries})
			}
			fmt.Fprintln(out, "week        meters  sess  pace/500m   spm    hr")
			for _, s := range summaries {
				fmt.Fprintf(out, "%s  %8s  %4d  %9s  %4s  %4s\n",
					s.WeekStart,
					display.FormatMeters(s.Meters),
					s.Sessions,
					orDash(s.AvgPace500m, func(v string) string { return v }),
					orDash(s.AvgSPM, jsNumber),
					orDash(s.AvgHR, strconv.Itoa))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&weeksFlag, "weeks", "w", "12", "number of weeks")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}

func newStatsGoalCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "goal",
		Short: "Goal trajectory and projection",
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
			end, err := configGoalEnd(cfg.Goal.EndDate)
			if err != nil {
				return err
			}
			projection := stats.ProjectGoal(goal, end, now)
			weeks := stats.RecentWeeks(workouts, now, 4)
			thisWeek := weeks[0]

			out := cmd.OutOrStdout()
			if asJSON {
				return envelope.Print(out, "c2.stats.goal.v1", goalPayload{
					Goal:       goal,
					Projection: projection,
					ThisWeek:   newWeekPayload(thisWeek.WeekStart, thisWeek.Meters, thisWeek.Sessions),
				})
			}

			fmt.Fprintf(out, "Progress: %s / %s (%s%%)\n",
				display.FormatMeters(goal.TotalMeters),
				display.FormatMeters(goal.Target),
				display.ToFixed(goal.Progress*100, 1))
			fmt.Fprintf(out, "Required pace: %s m/wk\n", display.FormatMeters(goal.RequiredPace))
			fmt.Fprintf(out, "Recent average: %s m/wk\n", display.FormatMeters(goal.CurrentAvgPace))
			fmt.Fprintf(out, "Projection at current pace: %s m (%s%%)\n",
				display.FormatMeters(projection.ProjectedTotalMeters), jsNumber(projection.ProjectedPct))
			if projection.ShortfallMeters > 0 {
				fmt.Fprintf(out, "Projected shortfall: %s m\n", display.FormatMeters(projection.ShortfallMeters))
			} else {
				fmt.Fprintln(out, "On track to exceed goal.")
			}
			fmt.Fprintf(out, "This week so far: %s m (%d sessions)\n",
				display.FormatMeters(thisWeek.Meters), thisWeek.Sessions)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}

func newStatsSplitsCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "splits <id>",
		Short: "Split analysis for one workout (id or 'last')",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref := args[0]
			_, p, err := loadStore()
			if err != nil {
				return err
			}
			workouts, err := storage.ReadWorkouts(p)
			if err != nil {
				return err
			}
			w := models.ResolveWorkout(workouts, ref)
			if w == nil {
				if ref == "last" {
					return reportf(cmd, "No workouts found. Run `c2 sync` first.")
				}
				return reportf(cmd, "No workout with id %s.", ref)
			}
			rows := analysis.SplitTable(*w)
			shape := analysis.SplitShape(rows)

			out := cmd.OutOrStdout()
			if asJSON {
				return envelope.Print(out, "c2.stats.splits.v1", splitsPayload{
					WorkoutID:  w.ID,
					Date:       w.Date,
					Distance:   w.Distance,
					SplitShape: shape,
					Splits:     rows,
				})
			}
			if len(rows) == 0 {
				fmt.Fprintf(out, "Workout %d has no split data.\n", w.ID)
				return nil
			}
			fmt.Fprintf(out, "Workout %d — %s — %sm — %s splits\n",
				w.ID, w.Date, display.FormatMeters(w.Distance), shape)
			for _, s := range rows {
				distance := "-"
				if s.Distance != nil {
					distance = display.FormatMeters(*s.Distance) + "m"
				}
				fmt.Fprintf(out, "  %d: %s  %s/500m  %sspm  HR %s\n",
					s.Index,
					distance,
					orDash(s.Pace500m, func(v string) string { return v }),
					orDash(s.StrokeRate, strconv.Itoa),
					orDash(s.HRAvg, strconv.Itoa))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}

func newStatsHRPaceCmd() *cobra.Command {
	var (
		weeksFlag string
		asJSON    bool
	)
	cmd := &cobra.Command{
		Use:   "hr-pace",
		Short: "Average heart rate by steady pace band (fitness proxy)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			weeks, err := positiveInt(cmd, weeksFlag, "Error: --weeks must be a positive integer.")
			if err != nil {
				return err
			}
			_, p, err := loadStore()
			if err != nil {
				return err
			}
			workouts, err := storage.ReadWorkouts(p)
			if err != nil {
				return err
			}
			bands := analysis.HRAtPace(workouts, time.Now(), weeks)

			out := cmd.OutOrStdout()
			if asJSON {
				if bands == nil {
					bands = []analysis.HRPaceBand{}
				}
				return envelope.Print(out, "c2.stats.hr-pace.v1", hrPacePayload{Weeks: weeks, Bands: bands})
			}
			if len(bands) == 0 {
				fmt.Fprintln(out, "No steady workouts with heart rate data in the window.")
				return nil
			}
			fmt.Fprintf(out, "Steady pace bands over the last %d weeks (HR avg, early→late half):\n", weeks)
			for _, b := range bands {
				trend := ""
				if b.HRDelta != nil {
					switch {
					case *b.HRDelta < 0:
						trend = fmt.Sprintf("  ↓%d (improving)", -*b.HRDelta)
					case *b.HRDelta > 0:
						trend = fmt.Sprintf("  ↑%d (watch)", *b.HRDelta)
					default:
						trend = "  → flat"
					}
				}
				halves := ""
				if b.EarlyAvgHR != nil && b.LateAvgHR != nil {
					halves = fmt.Sprintf(" (%d→%d)", *b.EarlyAvgHR, *b.LateAvgHR)
				}
				plural := "s"
				if b.Workouts == 1 {
					plural = ""
				}
				fmt.Fprintf(out, "  %s/500m: HR %d%s across %d workout%s%s\n",
					b.Band, b.AvgHR, halves, b.Workouts, plural, trend)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&weeksFlag, "weeks", "w", "8", "window in weeks")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}
