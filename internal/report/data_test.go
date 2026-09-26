package report

import (
	"testing"

	"github.com/richhaase/c2/internal/models"
)

func TestLatestSplitsPreferLongestThenMostRecentOnLatestDay(t *testing.T) {
	splits := &models.WorkoutDetail{Splits: []models.WorkoutSplit{{Time: 15000, Distance: new(5000)}}}
	workouts := []models.Workout{
		{ID: 1, Date: "2026-09-25 08:00:00", Distance: 10000, Workout: splits},
		{ID: 2, Date: "2026-09-26 08:00:00", Distance: 7000, Workout: splits},
		{ID: 3, Date: "2026-09-26 09:00:00", Distance: 7000, Workout: splits},
		{ID: 4, Date: "2026-09-26 10:00:00", Distance: 5000, Workout: splits},
		{ID: 5, Date: "2026-09-26 11:00:00", Distance: 10000},
	}
	got := buildActivityData(workouts, workouts, nil)
	if got.latestSplits == nil || got.latestSplits.WorkoutID != 3 {
		t.Fatalf("latest splits = %+v", got.latestSplits)
	}
	workouts = append(workouts, models.Workout{ID: 6, Date: "2026-09-27 08:00:00", Distance: 5000})
	if got := buildActivityData(workouts, workouts, nil); got.latestSplits != nil {
		t.Fatal("used splits from an older day")
	}
}
