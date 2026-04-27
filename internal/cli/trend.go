package cli

import (
	"fmt"

	"github.com/richhaase/c2/internal/display"
	"github.com/richhaase/c2/internal/model"
	"github.com/richhaase/c2/internal/stats"
	"github.com/spf13/cobra"
)

func newTrendCommand(deps Dependencies) *cobra.Command {
	var weeks int
	cmd := &cobra.Command{
		Use:   "trend",
		Short: "Show training trends over time",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := positiveIntFlag(weeks, "weeks"); err != nil {
				return err
			}
			workouts, err := deps.ReadWorkouts()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(workouts) == 0 {
				fmt.Fprintln(out, "No workouts found. Run `c2 sync` first.")
				return nil
			}
			summaries := stats.BuildWeekSummaries(workouts, weeks, deps.Now())
			printVolumeTrend(out, summaries)
			fmt.Fprintln(out)
			printPaceTrend(out, summaries)
			fmt.Fprintln(out)
			printSPMTrend(out, summaries)
			fmt.Fprintln(out)
			printHRTrend(out, summaries)
			return nil
		},
	}
	cmd.Flags().IntVarP(&weeks, "weeks", "w", 8, "number of weeks to display")
	return cmd
}

func printVolumeTrend(out interface{ Write([]byte) (int, error) }, summaries []stats.WeekSummary) {
	fmt.Fprintln(out, "Volume (meters/week):")
	prevMeters := 0
	max := maxMeters(summaries)
	for _, summary := range summaries {
		fmt.Fprintf(out, "  %s  %s %7s  %s  (%d sessions)\n",
			summary.WeekStart.Format("01/02"),
			display.TrendArrow(prevMeters, summary.Meters),
			display.FormatMeters(summary.Meters),
			display.SparkBar(summary.Meters, max),
			summary.Sessions,
		)
		prevMeters = summary.Meters
	}
}

func printPaceTrend(out interface{ Write([]byte) (int, error) }, summaries []stats.WeekSummary) {
	fmt.Fprintln(out, "Avg Pace (/500m):")
	prevPace := 0.0
	for _, summary := range summaries {
		if summary.PaceCount == 0 {
			fmt.Fprintf(out, "  %s    -\n", summary.WeekStart.Format("01/02"))
			continue
		}
		avg := summary.PaceSum / float64(summary.PaceCount)
		fmt.Fprintf(out, "  %s  %s %s\n", summary.WeekStart.Format("01/02"), display.PaceArrow(prevPace, avg), model.FormatSeconds(avg))
		prevPace = avg
	}
}

func printSPMTrend(out interface{ Write([]byte) (int, error) }, summaries []stats.WeekSummary) {
	fmt.Fprintln(out, "Avg Stroke Rate (spm):")
	prevSPM := 0
	for _, summary := range summaries {
		if summary.SPMCount == 0 {
			fmt.Fprintf(out, "  %s    -\n", summary.WeekStart.Format("01/02"))
			continue
		}
		avg := float64(summary.SPMSum) / float64(summary.SPMCount)
		fmt.Fprintf(out, "  %s  %s %4.1f\n", summary.WeekStart.Format("01/02"), display.TrendArrow(prevSPM, int(avg)), avg)
		prevSPM = int(avg)
	}
}

func printHRTrend(out interface{ Write([]byte) (int, error) }, summaries []stats.WeekSummary) {
	fmt.Fprintln(out, "Avg Heart Rate (bpm):")
	hasAny := false
	prevHR := 0
	for _, summary := range summaries {
		if summary.HRCount == 0 {
			fmt.Fprintf(out, "  %s    -\n", summary.WeekStart.Format("01/02"))
			continue
		}
		hasAny = true
		avg := float64(summary.HRSum) / float64(summary.HRCount)
		fmt.Fprintf(out, "  %s  %s %5.1f\n", summary.WeekStart.Format("01/02"), display.TrendArrow(prevHR, int(avg)), avg)
		prevHR = int(avg)
	}
	if !hasAny {
		fmt.Fprintln(out, "  No heart rate data available.")
	}
}

func maxMeters(summaries []stats.WeekSummary) int {
	max := 0
	for _, summary := range summaries {
		if summary.Meters > max {
			max = summary.Meters
		}
	}
	return max
}
