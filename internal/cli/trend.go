package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/display"
	"github.com/richhaase/c2/internal/envelope"
	"github.com/richhaase/c2/internal/stats"
)

func shortWeek(t time.Time) string {
	return fmt.Sprintf("%02d/%02d", int(t.Month()), t.Day())
}

func maxMeters(summaries []stats.WeekSummary) float64 {
	max := 0.0
	for _, ws := range summaries {
		if float64(ws.Meters) > max {
			max = float64(ws.Meters)
		}
	}
	return max
}

func printVolumeTrend(out io.Writer, summaries []stats.WeekSummary) {
	fmt.Fprintln(out, "Volume (meters/week):")
	prev := 0.0
	for _, ws := range summaries {
		arrow := display.TrendArrow(prev, float64(ws.Meters))
		bar := display.SparkBar(float64(ws.Meters), maxMeters(summaries))
		fmt.Fprintf(out, "  %s  %s %7s  %s  (%d sessions)\n",
			shortWeek(ws.WeekStart), arrow, display.FormatMeters(ws.Meters), bar, ws.Sessions)
		prev = float64(ws.Meters)
	}
}

func printPaceTrend(out io.Writer, summaries []stats.WeekSummary) {
	fmt.Fprintln(out, "Avg Pace (/500m):")
	prev := 0.0
	for _, ws := range summaries {
		if ws.PaceCount == 0 {
			fmt.Fprintf(out, "  %s    -\n", shortWeek(ws.WeekStart))
			continue
		}
		avg := ws.PaceSum / float64(ws.PaceCount)
		arrow := display.PaceArrow(prev, avg)
		mins := int(avg / 60)
		secs := avg - float64(mins)*60
		fmt.Fprintf(out, "  %s  %s %d:%s\n",
			shortWeek(ws.WeekStart), arrow, mins, padStart(display.ToFixed(secs, 1), 4, "0"))
		prev = avg
	}
}

func printSPMTrend(out io.Writer, summaries []stats.WeekSummary) {
	fmt.Fprintln(out, "Avg Stroke Rate (spm):")
	prev := 0.0
	for _, ws := range summaries {
		if ws.SPMCount == 0 {
			fmt.Fprintf(out, "  %s    -\n", shortWeek(ws.WeekStart))
			continue
		}
		avg := float64(ws.SPMSum) / float64(ws.SPMCount)
		fmt.Fprintf(out, "  %s  %s %4s\n",
			shortWeek(ws.WeekStart), display.TrendArrow(prev, avg), display.ToFixed(avg, 1))
		prev = avg
	}
}

func printHRTrend(out io.Writer, summaries []stats.WeekSummary) {
	fmt.Fprintln(out, "Avg Heart Rate (bpm):")
	hasAny := false
	prev := 0.0
	for _, ws := range summaries {
		if ws.HRCount == 0 {
			fmt.Fprintf(out, "  %s    -\n", shortWeek(ws.WeekStart))
			continue
		}
		hasAny = true
		avg := float64(ws.HRSum) / float64(ws.HRCount)
		fmt.Fprintf(out, "  %s  %s %5s\n",
			shortWeek(ws.WeekStart), display.TrendArrow(prev, avg), display.ToFixed(avg, 1))
		prev = avg
	}
	if !hasAny {
		fmt.Fprintln(out, "  No heart rate data available.")
	}
}

func padStart(s string, width int, pad string) string {
	for len(s) < width {
		s = pad + s
	}
	return s
}

type trendPayload struct {
	Weeks []stats.WeekSummaryData `json:"weeks"`
}

func newTrendCmd() *cobra.Command {
	var (
		weeksFlag string
		asJSON    bool
	)

	cmd := &cobra.Command{
		Use:   "trend",
		Short: "Show training trends over time",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, workouts, err := loadWorkouts(cmd)
			if err != nil {
				return err
			}
			weeks, err := weekCount(cmd, weeksFlag)
			if err != nil {
				return err
			}
			summaries := stats.BuildWeekSummaries(workouts, time.Now(), weeks)

			out := cmd.OutOrStdout()
			if asJSON {
				payload := make([]stats.WeekSummaryData, 0, len(summaries))
				for _, ws := range summaries {
					payload = append(payload, stats.WeekSummaryDataOf(ws))
				}
				return envelope.Print(out, "c2.trend.v1", trendPayload{Weeks: payload})
			}

			if len(workouts) == 0 {
				fmt.Fprintln(out, "No workouts found. Run `c2 sync` first.")
				return nil
			}

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

	cmd.Flags().StringVarP(&weeksFlag, "weeks", "w", "8", "number of weeks to display")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}
