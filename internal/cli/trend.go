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
				_, err := fmt.Fprintln(out, "No workouts found. Run `c2 sync` first.")
				return err
			}
			summaries := stats.BuildWeekSummaries(workouts, weeks, deps.Now())
			if err := printVolumeTrend(out, summaries); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
			if err := printPaceTrend(out, summaries); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
			if err := printSPMTrend(out, summaries); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
			return printHRTrend(out, summaries)
		},
	}
	cmd.Flags().IntVarP(&weeks, "weeks", "w", 8, "number of weeks to display")
	return cmd
}

func printVolumeTrend(out interface{ Write([]byte) (int, error) }, summaries []stats.WeekSummary) error {
	if _, err := fmt.Fprintln(out, "Volume (meters/week):"); err != nil {
		return err
	}
	prevMeters := 0
	max := maxMeters(summaries)
	for _, summary := range summaries {
		if _, err := fmt.Fprintf(out, "  %s  %s %7s  %s  (%d sessions)\n",
			summary.WeekStart.Format("01/02"),
			display.TrendArrow(prevMeters, summary.Meters),
			display.FormatMeters(summary.Meters),
			display.SparkBar(summary.Meters, max),
			summary.Sessions,
		); err != nil {
			return err
		}
		prevMeters = summary.Meters
	}
	return nil
}

func printPaceTrend(out interface{ Write([]byte) (int, error) }, summaries []stats.WeekSummary) error {
	if _, err := fmt.Fprintln(out, "Avg Pace (/500m):"); err != nil {
		return err
	}
	prevPace := 0.0
	for _, summary := range summaries {
		if summary.PaceCount == 0 {
			if _, err := fmt.Fprintf(out, "  %s    -\n", summary.WeekStart.Format("01/02")); err != nil {
				return err
			}
			continue
		}
		avg := summary.PaceSum / float64(summary.PaceCount)
		if _, err := fmt.Fprintf(out, "  %s  %s %s\n", summary.WeekStart.Format("01/02"), display.PaceArrow(prevPace, avg), model.FormatSeconds(avg)); err != nil {
			return err
		}
		prevPace = avg
	}
	return nil
}

func printSPMTrend(out interface{ Write([]byte) (int, error) }, summaries []stats.WeekSummary) error {
	if _, err := fmt.Fprintln(out, "Avg Stroke Rate (spm):"); err != nil {
		return err
	}
	prevSPM := 0
	for _, summary := range summaries {
		if summary.SPMCount == 0 {
			if _, err := fmt.Fprintf(out, "  %s    -\n", summary.WeekStart.Format("01/02")); err != nil {
				return err
			}
			continue
		}
		avg := float64(summary.SPMSum) / float64(summary.SPMCount)
		if _, err := fmt.Fprintf(out, "  %s  %s %4.1f\n", summary.WeekStart.Format("01/02"), display.TrendArrow(prevSPM, int(avg)), avg); err != nil {
			return err
		}
		prevSPM = int(avg)
	}
	return nil
}

func printHRTrend(out interface{ Write([]byte) (int, error) }, summaries []stats.WeekSummary) error {
	if _, err := fmt.Fprintln(out, "Avg Heart Rate (bpm):"); err != nil {
		return err
	}
	hasAny := false
	prevHR := 0
	for _, summary := range summaries {
		if summary.HRCount == 0 {
			if _, err := fmt.Fprintf(out, "  %s    -\n", summary.WeekStart.Format("01/02")); err != nil {
				return err
			}
			continue
		}
		hasAny = true
		avg := float64(summary.HRSum) / float64(summary.HRCount)
		if _, err := fmt.Fprintf(out, "  %s  %s %5.1f\n", summary.WeekStart.Format("01/02"), display.TrendArrow(prevHR, int(avg)), avg); err != nil {
			return err
		}
		prevHR = int(avg)
	}
	if !hasAny {
		if _, err := fmt.Fprintln(out, "  No heart rate data available."); err != nil {
			return err
		}
	}
	return nil
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
