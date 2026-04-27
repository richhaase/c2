package stats

import (
	"math"
	"time"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/model"
)

type GoalProgress struct {
	Target          int
	TotalMeters     int
	Progress        float64
	WeeksElapsed    int
	TotalWeeks      int
	RemainingMeters int
	RemainingWeeks  int
	RequiredPace    int
	CurrentAvgPace  int
	OnPace          bool
}

func ComputeGoalProgress(workouts []model.Workout, cfg config.GoalConfig, now time.Time) GoalProgress {
	target := cfg.TargetMeters
	start, startErr := config.ParseGoalDate(cfg.StartDate)
	end, endErr := config.ParseGoalDate(cfg.EndDate)
	if startErr != nil || endErr != nil || target == 0 {
		return GoalProgress{Target: target}
	}

	totalMeters := 0
	endExclusive := end.AddDate(0, 0, 1)
	for _, workout := range workouts {
		parsed, err := model.ParsedDate(workout)
		if err != nil {
			continue
		}
		if !parsed.Before(start) && parsed.Before(endExclusive) {
			totalMeters += workout.Distance
		}
	}

	progress := float64(totalMeters) / float64(target)
	totalDays := end.Sub(start).Hours() / 24
	totalWeeks := int(math.Ceil(totalDays / 7))

	weeksElapsed := 0
	if now.After(start) {
		weeksElapsed = int(math.Floor(now.Sub(start).Hours() / (24 * 7)))
	}

	remainingMeters := target - totalMeters
	if remainingMeters < 0 {
		remainingMeters = 0
	}

	remainingWeeks := totalWeeks - weeksElapsed
	if remainingWeeks < 1 {
		remainingWeeks = 1
	}

	requiredPace := int(math.Floor(float64(remainingMeters) / float64(remainingWeeks)))
	currentAvgPace := 0
	if weeksElapsed > 0 {
		currentAvgPace = int(math.Floor(float64(totalMeters) / float64(weeksElapsed)))
	}

	targetWeekly := float64(target) / float64(totalWeeks)
	onPace := float64(currentAvgPace) >= targetWeekly

	return GoalProgress{
		Target:          target,
		TotalMeters:     totalMeters,
		Progress:        progress,
		WeeksElapsed:    weeksElapsed,
		TotalWeeks:      totalWeeks,
		RemainingMeters: remainingMeters,
		RemainingWeeks:  remainingWeeks,
		RequiredPace:    requiredPace,
		CurrentAvgPace:  currentAvgPace,
		OnPace:          onPace,
	}
}
