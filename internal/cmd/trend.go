// Copyright (c) 2026 Rich Haase. All rights reserved.
// Use of this source code is governed by the MIT license.

package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2cli/internal/display"
	"github.com/richhaase/c2cli/internal/models"
	"github.com/richhaase/c2cli/internal/storage"
)

func newTrendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trend",
		Short: "Show training trends over time",
		Long:  "Show weekly trends for pace, volume, stroke rate, and heart rate.",
		RunE: func(cmd *cobra.Command, args []string) error {
			weeks, _ := cmd.Flags().GetInt("weeks")
			return runTrend(weeks)
		},
	}
	cmd.Flags().IntP("weeks", "w", 8, "number of weeks to display")
	return cmd
}

type weekSummary struct {
	weekStart time.Time
	meters    int64
	sessions  int
	paceSum   float64 // sum of pace seconds for averaging
	paceCount int
	spmSum    int
	spmCount  int
	hrSum     int
	hrCount   int
}

func runTrend(weeks int) error {
	workouts, err := storage.ReadWorkouts()
	if err != nil {
		return err
	}
	if len(workouts) == 0 {
		fmt.Println("No workouts found. Run `c2cli sync` first.")
		return nil
	}

	now := time.Now()
	summaries := buildWeekSummaries(workouts, now, weeks)

	printVolumeTrend(summaries)
	fmt.Println()
	printPaceTrend(summaries)
	fmt.Println()
	printSPMTrend(summaries)
	fmt.Println()
	printHRTrend(summaries)

	return nil
}

// mondayOf returns the Monday of the week containing t.
func mondayOf(t time.Time) time.Time {
	t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	offset := (int(t.Weekday()) + 6) % 7
	return t.AddDate(0, 0, -offset)
}

func buildWeekSummaries(workouts []models.Workout, now time.Time, weeks int) []weekSummary {
	thisMonday := mondayOf(now)
	cutoff := thisMonday.AddDate(0, 0, -(weeks-1)*7)

	// Build a map keyed by Monday date string
	weekMap := make(map[string]*weekSummary)
	for i := 0; i < weeks; i++ {
		monday := thisMonday.AddDate(0, 0, -i*7)
		key := monday.Format("2006-01-02")
		weekMap[key] = &weekSummary{weekStart: monday}
	}

	for _, w := range workouts {
		t, err := w.ParsedDate()
		if err != nil || t.Before(cutoff) || t.After(now) {
			continue
		}
		monday := mondayOf(t)
		key := monday.Format("2006-01-02")
		ws, ok := weekMap[key]
		if !ok {
			continue
		}
		ws.meters += w.Distance
		ws.sessions++

		if w.Distance > 0 && w.Time > 0 {
			paceSeconds := float64(w.Time) / 10.0 * 500.0 / float64(w.Distance)
			ws.paceSum += paceSeconds
			ws.paceCount++
		}
		if w.StrokeRate > 0 {
			ws.spmSum += w.StrokeRate
			ws.spmCount++
		}
		if w.HeartRate != nil && w.HeartRate.Average > 0 {
			ws.hrSum += w.HeartRate.Average
			ws.hrCount++
		}
	}

	// Sort oldest first
	result := make([]weekSummary, 0, weeks)
	for _, ws := range weekMap {
		result = append(result, *ws)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].weekStart.Before(result[j].weekStart)
	})
	return result
}

func trendArrow(prev, curr float64) string {
	if prev == 0 {
		return " "
	}
	diff := (curr - prev) / prev
	switch {
	case diff > 0.02:
		return "↑"
	case diff < -0.02:
		return "↓"
	default:
		return "→"
	}
}

func printVolumeTrend(summaries []weekSummary) {
	fmt.Println("Volume (meters/week):")
	var prevMeters float64
	for _, ws := range summaries {
		arrow := trendArrow(prevMeters, float64(ws.meters))
		bar := sparkBar(ws.meters, maxMeters(summaries))
		fmt.Printf("  %s  %s %7s  %s  (%d sessions)\n",
			ws.weekStart.Format("01/02"),
			arrow,
			display.FormatMeters(ws.meters),
			bar,
			ws.sessions)
		prevMeters = float64(ws.meters)
	}
}

func printPaceTrend(summaries []weekSummary) {
	fmt.Println("Avg Pace (/500m):")
	var prevPace float64
	for _, ws := range summaries {
		if ws.paceCount == 0 {
			fmt.Printf("  %s    -\n", ws.weekStart.Format("01/02"))
			continue
		}
		avgPace := ws.paceSum / float64(ws.paceCount)
		// For pace, lower is better, so flip the arrow
		arrow := trendArrow(avgPace, prevPace)
		if prevPace == 0 {
			arrow = " "
		}
		mins := int(avgPace) / 60
		secs := avgPace - float64(mins*60)
		fmt.Printf("  %s  %s %d:%04.1f\n",
			ws.weekStart.Format("01/02"),
			arrow,
			mins, secs)
		prevPace = avgPace
	}
}

func printSPMTrend(summaries []weekSummary) {
	fmt.Println("Avg Stroke Rate (spm):")
	var prevSPM float64
	for _, ws := range summaries {
		if ws.spmCount == 0 {
			fmt.Printf("  %s    -\n", ws.weekStart.Format("01/02"))
			continue
		}
		avg := float64(ws.spmSum) / float64(ws.spmCount)
		arrow := trendArrow(prevSPM, avg)
		fmt.Printf("  %s  %s %4.1f\n",
			ws.weekStart.Format("01/02"),
			arrow,
			avg)
		prevSPM = avg
	}
}

func printHRTrend(summaries []weekSummary) {
	fmt.Println("Avg Heart Rate (bpm):")
	hasAny := false
	var prevHR float64
	for _, ws := range summaries {
		if ws.hrCount == 0 {
			fmt.Printf("  %s    -\n", ws.weekStart.Format("01/02"))
			continue
		}
		hasAny = true
		avg := float64(ws.hrSum) / float64(ws.hrCount)
		arrow := trendArrow(prevHR, avg)
		fmt.Printf("  %s  %s %5.1f\n",
			ws.weekStart.Format("01/02"),
			arrow,
			avg)
		prevHR = avg
	}
	if !hasAny {
		fmt.Println("  No heart rate data available.")
	}
}

func maxMeters(summaries []weekSummary) int64 {
	var max int64
	for _, ws := range summaries {
		if ws.meters > max {
			max = ws.meters
		}
	}
	return max
}

func sparkBar(value, max int64) string {
	if max == 0 {
		return ""
	}
	const barWidth = 20
	filled := int(float64(value) / float64(max) * barWidth)
	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	return bar
}
