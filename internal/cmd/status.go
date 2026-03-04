package cmd

import (
	"fmt"
	"time"

	"github.com/richhaase/c2cli/internal/config"
	"github.com/richhaase/c2cli/internal/display"
	"github.com/richhaase/c2cli/internal/storage"
)

func RunStatus() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	workouts, err := storage.ReadWorkouts()
	if err != nil {
		return err
	}

	target := cfg.Goal.TargetMeters
	start, err := config.ParseGoalDate(cfg.Goal.StartDate)
	if err != nil {
		return fmt.Errorf("invalid goal.start_date: %w", err)
	}
	end, err := config.ParseGoalDate(cfg.Goal.EndDate)
	if err != nil {
		return fmt.Errorf("invalid goal.end_date: %w", err)
	}
	today := time.Now()

	// Total meters in the goal period
	var totalMeters int64
	for _, w := range workouts {
		t, err := w.ParsedDate()
		if err != nil {
			continue
		}
		if !t.Before(start) && !t.After(end) {
			totalMeters += w.Distance
		}
	}

	progress := float64(totalMeters) / float64(target)
	totalDays := end.Sub(start).Hours() / 24
	totalWeeks := int64(totalDays/7 + 0.99)
	var weeksElapsed int64
	if today.After(start) {
		weeksElapsed = int64(today.Sub(start).Hours() / 24 / 7)
	}

	remainingMeters := target - totalMeters
	if remainingMeters < 0 {
		remainingMeters = 0
	}
	remainingWeeks := totalWeeks - weeksElapsed
	if remainingWeeks < 1 {
		remainingWeeks = 1
	}
	requiredPace := remainingMeters / remainingWeeks

	fmt.Printf("Goal: %sm\n", display.FormatMeters(target))
	fmt.Printf("Season start: %s\n", start.Format("2006-01-02"))
	fmt.Printf("Progress: %s / %s (%s)\n",
		display.FormatMeters(totalMeters),
		display.FormatMeters(target),
		display.FormatPercent(progress))
	fmt.Printf("Weeks elapsed: %d / %d\n", weeksElapsed, totalWeeks)
	fmt.Printf("Required pace: %s\n", display.FormatMetersPerWeek(requiredPace))
	fmt.Println()

	// Last 4 weeks breakdown
	fmt.Println("Last 4 weeks:")
	for i := 0; i < 4; i++ {
		weekEnd := today.AddDate(0, 0, -i*7)
		weekStart := weekEnd.AddDate(0, 0, -6)
		// Align to Monday
		daysSinceMonday := int(weekStart.Weekday()+6) % 7
		weekStartAligned := weekStart.AddDate(0, 0, -daysSinceMonday)
		weekEndAligned := weekStartAligned.AddDate(0, 0, 7)

		var meters int64
		sessions := 0
		for _, w := range workouts {
			t, err := w.ParsedDate()
			if err != nil {
				continue
			}
			if !t.Before(weekStartAligned) && t.Before(weekEndAligned) {
				meters += w.Distance
				sessions++
			}
		}

		fmt.Printf("  Week of %s: %s (%d sessions)\n",
			weekStartAligned.Format("01/02"),
			display.FormatMeters(meters),
			sessions)
	}
	fmt.Println()

	// Weekly average and on-pace check
	if weeksElapsed > 0 {
		avg := totalMeters / weeksElapsed
		targetWeekly := float64(target) / float64(totalWeeks)
		onPace := float64(avg) >= targetWeekly
		indicator := "behind pace ✗"
		if onPace {
			indicator = "on pace ✓"
		}
		fmt.Printf("Current avg: %s — %s\n", display.FormatMetersPerWeek(avg), indicator)
	}

	return nil
}
