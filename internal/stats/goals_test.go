package stats

import (
	"math"
	"testing"
	"time"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/model"
)

func makeGoalConfig(overrides config.GoalConfig) config.GoalConfig {
	cfg := config.GoalConfig{
		StartDate:    "2026-01-01",
		EndDate:      "2026-12-31",
		TargetMeters: 1000000,
	}
	if overrides.StartDate != "" {
		cfg.StartDate = overrides.StartDate
	}
	if overrides.EndDate != "" {
		cfg.EndDate = overrides.EndDate
	}
	if overrides.TargetMeters != 0 {
		cfg.TargetMeters = overrides.TargetMeters
	}
	return cfg
}

func TestComputeGoalProgressComputesMidSeasonProgress(t *testing.T) {
	cfg := makeGoalConfig(config.GoalConfig{})
	workouts := []model.Workout{
		makeStatsWorkout(1, "2026-03-01 10:00:00", 100000),
		makeStatsWorkout(2, "2026-02-01 10:00:00", 100000),
	}
	now := time.Date(2026, 3, 7, 0, 0, 0, 0, time.Local)
	goal := ComputeGoalProgress(workouts, cfg, now)
	if goal.TotalMeters != 200000 {
		t.Fatalf("TotalMeters = %d, want 200000", goal.TotalMeters)
	}
	if goal.Target != 1000000 {
		t.Fatalf("Target = %d, want 1000000", goal.Target)
	}
	if math.Abs(goal.Progress-0.2) > 0.005 {
		t.Fatalf("Progress = %v, want close to 0.2", goal.Progress)
	}
	if goal.RemainingMeters != 800000 {
		t.Fatalf("RemainingMeters = %d, want 800000", goal.RemainingMeters)
	}
	if goal.WeeksElapsed <= 0 {
		t.Fatalf("WeeksElapsed = %d, want > 0", goal.WeeksElapsed)
	}
}

func TestComputeGoalProgressClampsRemainingMetersToZeroWhenGoalExceeded(t *testing.T) {
	cfg := makeGoalConfig(config.GoalConfig{TargetMeters: 100000})
	workouts := []model.Workout{makeStatsWorkout(1, "2026-03-01 10:00:00", 150000)}
	now := time.Date(2026, 3, 7, 0, 0, 0, 0, time.Local)
	goal := ComputeGoalProgress(workouts, cfg, now)
	if goal.RemainingMeters != 0 {
		t.Fatalf("RemainingMeters = %d, want 0", goal.RemainingMeters)
	}
	if goal.Progress <= 1 {
		t.Fatalf("Progress = %v, want > 1", goal.Progress)
	}
}

func TestComputeGoalProgressExcludesWorkoutsOutsideGoalDateRange(t *testing.T) {
	cfg := makeGoalConfig(config.GoalConfig{})
	workouts := []model.Workout{
		makeStatsWorkout(1, "2025-12-01 10:00:00", 50000),
		makeStatsWorkout(2, "2026-03-01 10:00:00", 100000),
		makeStatsWorkout(3, "2027-02-01 10:00:00", 50000),
	}
	now := time.Date(2026, 3, 7, 0, 0, 0, 0, time.Local)
	goal := ComputeGoalProgress(workouts, cfg, now)
	if goal.TotalMeters != 100000 {
		t.Fatalf("TotalMeters = %d, want 100000", goal.TotalMeters)
	}
}

func TestComputeGoalProgressIncludesWorkoutsDuringEndDate(t *testing.T) {
	cfg := makeGoalConfig(config.GoalConfig{})
	workouts := []model.Workout{
		makeStatsWorkout(1, "2026-12-31 10:00:00", 50000),
		makeStatsWorkout(2, "2027-01-01 00:00:00", 50000),
	}
	now := time.Date(2026, 12, 31, 12, 0, 0, 0, time.Local)
	goal := ComputeGoalProgress(workouts, cfg, now)
	if goal.TotalMeters != 50000 {
		t.Fatalf("TotalMeters = %d, want 50000", goal.TotalMeters)
	}
}

func TestComputeGoalProgressBeforeStartDateHasNoElapsedAverage(t *testing.T) {
	cfg := makeGoalConfig(config.GoalConfig{StartDate: "2026-06-01"})
	now := time.Date(2026, 3, 7, 0, 0, 0, 0, time.Local)
	goal := ComputeGoalProgress(nil, cfg, now)
	if goal.WeeksElapsed != 0 {
		t.Fatalf("WeeksElapsed = %d, want 0", goal.WeeksElapsed)
	}
	if goal.CurrentAvgPace != 0 {
		t.Fatalf("CurrentAvgPace = %d, want 0", goal.CurrentAvgPace)
	}
}
