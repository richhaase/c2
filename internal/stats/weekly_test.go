package stats

import (
	"testing"
	"time"

	"github.com/richhaase/c2/internal/model"
)

func makeStatsWorkout(id int, date string, distance int, workoutTime ...int) model.Workout {
	tenths := int(float64(distance)*3.5 + 0.5)
	if len(workoutTime) > 0 {
		tenths = workoutTime[0]
	}
	return model.Workout{
		ID:            id,
		UserID:        1,
		Date:          date,
		Distance:      distance,
		Type:          "rower",
		Time:          tenths,
		TimeFormatted: "0:00.0",
	}
}

func TestMondayOf(t *testing.T) {
	tests := []struct {
		name string
		day  time.Time
		want time.Time
	}{
		{name: "monday returns same date", day: time.Date(2026, 3, 2, 15, 4, 5, 0, time.Local), want: time.Date(2026, 3, 2, 0, 0, 0, 0, time.Local)},
		{name: "wednesday returns previous monday", day: time.Date(2026, 3, 4, 15, 4, 5, 0, time.Local), want: time.Date(2026, 3, 2, 0, 0, 0, 0, time.Local)},
		{name: "sunday returns previous monday", day: time.Date(2026, 3, 8, 15, 4, 5, 0, time.Local), want: time.Date(2026, 3, 2, 0, 0, 0, 0, time.Local)},
		{name: "handles month boundary", day: time.Date(2026, 3, 1, 15, 4, 5, 0, time.Local), want: time.Date(2026, 2, 23, 0, 0, 0, 0, time.Local)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MondayOf(tt.day)
			if !got.Equal(tt.want) {
				t.Fatalf("MondayOf() = %v, want %v", got, tt.want)
			}
			if got.Weekday() != time.Monday {
				t.Fatalf("MondayOf() weekday = %v, want Monday", got.Weekday())
			}
		})
	}
}

func TestWorkoutsInRangeFiltersByHalfOpenDateRange(t *testing.T) {
	workouts := []model.Workout{
		makeStatsWorkout(1, "2026-01-15 10:00:00", 5000),
		makeStatsWorkout(2, "2026-02-15 10:00:00", 5000),
		makeStatsWorkout(3, "2026-03-15 10:00:00", 5000),
	}
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, 3, 1, 0, 0, 0, 0, time.Local)
	result := WorkoutsInRange(workouts, from, to)
	if len(result) != 1 {
		t.Fatalf("len(WorkoutsInRange()) = %d, want 1", len(result))
	}
	if result[0].ID != 2 {
		t.Fatalf("result[0].ID = %d, want 2", result[0].ID)
	}
}

func TestWorkoutsInRangeReturnsEmptyForNoMatches(t *testing.T) {
	workouts := []model.Workout{makeStatsWorkout(1, "2026-06-15 10:00:00", 5000)}
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, 2, 1, 0, 0, 0, 0, time.Local)
	if got := WorkoutsInRange(workouts, from, to); len(got) != 0 {
		t.Fatalf("len(WorkoutsInRange()) = %d, want 0", len(got))
	}
}

func TestBuildWeekSummariesBucketsWorkoutsIntoCorrectWeeks(t *testing.T) {
	now := time.Date(2026, 3, 7, 0, 0, 0, 0, time.Local)
	workouts := []model.Workout{
		makeStatsWorkout(1, "2026-03-02 10:00:00", 5000),
		makeStatsWorkout(2, "2026-02-23 10:00:00", 6000),
	}
	summaries := BuildWeekSummaries(workouts, 2, now)
	if len(summaries) != 2 {
		t.Fatalf("len(BuildWeekSummaries()) = %d, want 2", len(summaries))
	}
	if summaries[0].Meters != 6000 || summaries[1].Meters != 5000 {
		t.Fatalf("meters = %d, %d; want 6000, 5000", summaries[0].Meters, summaries[1].Meters)
	}
}

func TestBuildWeekSummariesCountsSessionsAsUniqueDays(t *testing.T) {
	now := time.Date(2026, 3, 7, 0, 0, 0, 0, time.Local)
	workouts := []model.Workout{
		makeStatsWorkout(1, "2026-03-02 09:00:00", 1000),
		makeStatsWorkout(2, "2026-03-02 10:00:00", 2000),
		makeStatsWorkout(3, "2026-03-04 10:00:00", 3000),
	}
	summaries := BuildWeekSummaries(workouts, 1, now)
	if summaries[0].Meters != 6000 {
		t.Fatalf("meters = %d, want 6000", summaries[0].Meters)
	}
	if summaries[0].Sessions != 2 {
		t.Fatalf("sessions = %d, want 2", summaries[0].Sessions)
	}
}

func TestBuildWeekSummariesReturnsEmptySummariesForNoWorkouts(t *testing.T) {
	now := time.Date(2026, 3, 7, 0, 0, 0, 0, time.Local)
	summaries := BuildWeekSummaries(nil, 4, now)
	if len(summaries) != 4 {
		t.Fatalf("len(BuildWeekSummaries()) = %d, want 4", len(summaries))
	}
	for i, summary := range summaries {
		if summary.Meters != 0 {
			t.Fatalf("summary[%d].Meters = %d, want 0", i, summary.Meters)
		}
	}
}

func TestBuildWeekSummariesAggregatesPaceStrokeRateAndHeartRate(t *testing.T) {
	strokeRate := 24
	heartRate := 112
	now := time.Date(2026, 3, 7, 0, 0, 0, 0, time.Local)
	workouts := []model.Workout{{
		ID:            1,
		UserID:        1,
		Date:          "2026-03-02 09:00:00",
		Distance:      5000,
		Type:          "rower",
		Time:          12000,
		TimeFormatted: "20:00.0",
		StrokeRate:    &strokeRate,
		HeartRate:     &model.HeartRate{Average: &heartRate},
	}}
	summaries := BuildWeekSummaries(workouts, 1, now)
	if summaries[0].PaceSum != 120 || summaries[0].PaceCount != 1 {
		t.Fatalf("pace = sum %v count %d, want sum 120 count 1", summaries[0].PaceSum, summaries[0].PaceCount)
	}
	if summaries[0].SPMSum != 24 || summaries[0].SPMCount != 1 {
		t.Fatalf("spm = sum %d count %d, want sum 24 count 1", summaries[0].SPMSum, summaries[0].SPMCount)
	}
	if summaries[0].HRSum != 112 || summaries[0].HRCount != 1 {
		t.Fatalf("hr = sum %d count %d, want sum 112 count 1", summaries[0].HRSum, summaries[0].HRCount)
	}
}
