package analysis

import (
	"math"
	"testing"
	"time"

	"github.com/richhaase/c2/internal/models"
)

func intPtr(v int) *int { return &v }

func floatPtr(v float64) *float64 { return &v }

func split(timeTenths float64, distance, spm, hr int) models.WorkoutSplit {
	return models.WorkoutSplit{
		Type:       "distance",
		Time:       timeTenths,
		Distance:   intPtr(distance),
		StrokeRate: intPtr(spm),
		HeartRate:  &models.HeartRate{Average: intPtr(hr), Max: intPtr(hr + 8)},
	}
}

func workoutWithSplits(splits []models.WorkoutSplit) models.Workout {
	distance := 0
	totalTime := 0.0
	for _, s := range splits {
		if s.Distance != nil {
			distance += *s.Distance
		}
		totalTime += s.Time
	}
	return models.Workout{
		ID:            1,
		UserID:        1,
		Date:          "2026-07-03 12:00:00",
		Distance:      distance,
		Type:          "rower",
		Time:          int(totalTime),
		TimeFormatted: "34:39.1",
		Workout: &models.WorkoutDetail{
			Targets: &models.WorkoutTargets{Pace: floatPtr(1750)},
			Splits:  splits,
		},
	}
}

func negativeSplitWorkout() models.Workout {
	return workoutWithSplits([]models.WorkoutSplit{
		split(4280, 1200, 24, 93),
		split(4210, 1200, 25, 107),
		split(4140, 1200, 25, 115),
		split(4076, 1200, 26, 120),
		split(4088, 1200, 26, 122),
	})
}

func TestSplitTableComputesPerSplitPaceFromTenths(t *testing.T) {
	rows := SplitTable(negativeSplitWorkout())
	if len(rows) != 5 {
		t.Fatalf("rows length = %d, want 5", len(rows))
	}
	if got := *rows[0].Pace500mSeconds; got != 178.3 {
		t.Errorf("rows[0].pace_500m_seconds = %v, want 178.3", got)
	}
	if got := *rows[0].Pace500m; got != "2:58.3" {
		t.Errorf("rows[0].pace_500m = %q, want %q", got, "2:58.3")
	}
	if got := *rows[3].Pace500m; got != "2:49.8" {
		t.Errorf("rows[3].pace_500m = %q, want %q", got, "2:49.8")
	}
	if got := *rows[0].HRAvg; got != 93 {
		t.Errorf("rows[0].hr_avg = %d, want 93", got)
	}
	if got := *rows[0].HRMax; got != 101 {
		t.Errorf("rows[0].hr_max = %d, want 101", got)
	}
}

func TestSplitShapeDetectsNegativeSplit(t *testing.T) {
	if got := SplitShape(SplitTable(negativeSplitWorkout())); got != ShapeNegative {
		t.Errorf("SplitShape = %q, want %q", got, ShapeNegative)
	}
}

func TestSplitShapeDetectsEvenPositiveVariableAndUnknown(t *testing.T) {
	even := workoutWithSplits([]models.WorkoutSplit{
		split(4200, 1200, 25, 110),
		split(4210, 1200, 25, 112),
		split(4195, 1200, 25, 113),
		split(4205, 1200, 25, 114),
	})
	if got := SplitShape(SplitTable(even)); got != ShapeEven {
		t.Errorf("even SplitShape = %q, want %q", got, ShapeEven)
	}

	positive := workoutWithSplits([]models.WorkoutSplit{
		split(4000, 1200, 26, 118),
		split(4100, 1200, 25, 120),
		split(4250, 1200, 24, 121),
		split(4350, 1200, 23, 122),
	})
	if got := SplitShape(SplitTable(positive)); got != ShapePositive {
		t.Errorf("positive SplitShape = %q, want %q", got, ShapePositive)
	}

	variable := workoutWithSplits([]models.WorkoutSplit{
		split(4000, 1200, 26, 118),
		split(4400, 1200, 22, 110),
		split(3950, 1200, 27, 125),
		split(4380, 1200, 22, 112),
	})
	if got := SplitShape(SplitTable(variable)); got != ShapeVariable {
		t.Errorf("variable SplitShape = %q, want %q", got, ShapeVariable)
	}

	single := workoutWithSplits([]models.WorkoutSplit{split(4200, 1200, 25, 110)})
	if got := SplitShape(SplitTable(single)); got != ShapeUnknown {
		t.Errorf("single-split SplitShape = %q, want %q", got, ShapeUnknown)
	}
	if got := SplitShape(nil); got != ShapeUnknown {
		t.Errorf("empty SplitShape = %q, want %q", got, ShapeUnknown)
	}
}

func TestStrokeSummaryAggregatesSamples(t *testing.T) {
	strokes := []models.StrokeData{
		{T: floatPtr(100), D: floatPtr(500), P: floatPtr(1750), SPM: floatPtr(24), HR: floatPtr(110)},
		{T: floatPtr(200), D: floatPtr(1000), P: floatPtr(1730), SPM: floatPtr(25), HR: floatPtr(118)},
		{T: floatPtr(300), D: floatPtr(1500), P: floatPtr(1710), SPM: floatPtr(26), HR: floatPtr(126)},
		{T: floatPtr(400), D: floatPtr(2000), P: floatPtr(0), SPM: floatPtr(0), HR: floatPtr(0)},
	}
	s := StrokeSummary(strokes)
	if s.Samples != 4 {
		t.Errorf("samples = %d, want 4", s.Samples)
	}
	if got := *s.AvgPace500mSeconds; got != 173 {
		t.Errorf("avg_pace_500m_seconds = %v, want 173", got)
	}
	if got := *s.AvgPace500m; got != "2:53.0" {
		t.Errorf("avg_pace_500m = %q, want %q", got, "2:53.0")
	}
	if got := *s.AvgSPM; got != 25 {
		t.Errorf("avg_spm = %v, want 25", got)
	}
	if got := *s.AvgHR; got != 118 {
		t.Errorf("avg_hr = %d, want 118", got)
	}
	if got := *s.MaxHR; got != 126 {
		t.Errorf("max_hr = %v, want 126", got)
	}
}

func steadyWorkout(id int64, date string, paceSecs float64, hr int) models.Workout {
	distance := 6000
	return models.Workout{
		ID:            id,
		UserID:        1,
		Date:          date,
		Distance:      distance,
		Type:          "rower",
		Time:          int(math.Round(paceSecs * (float64(distance) / 500) * 10)),
		TimeFormatted: "x",
		HeartRate:     &models.HeartRate{Average: intPtr(hr)},
	}
}

func TestHRAtPaceBucketsSteadyWorkAndReportsDrift(t *testing.T) {
	now := time.Date(2026, time.July, 5, 12, 0, 0, 0, time.Local)

	interval := steadyWorkout(6, "2026-07-02 08:00:00", 150, 140)
	interval.RestTime = intPtr(600)

	workouts := []models.Workout{
		steadyWorkout(1, "2026-05-20 08:00:00", 173, 120),
		steadyWorkout(2, "2026-05-27 08:00:00", 174, 118),
		steadyWorkout(3, "2026-06-24 08:00:00", 172, 112),
		steadyWorkout(4, "2026-07-01 08:00:00", 171, 110),
		steadyWorkout(5, "2026-07-03 08:00:00", 168, 115),
		interval,
		steadyWorkout(7, "2026-01-01 08:00:00", 173, 130),
	}
	bands := HRAtPace(workouts, now, 8)

	if len(bands) != 2 {
		t.Fatalf("bands length = %d, want 2", len(bands))
	}

	band170 := findBand(t, bands, 170)
	if band170.Workouts != 4 {
		t.Errorf("band170.workouts = %d, want 4", band170.Workouts)
	}
	if got := *band170.EarlyAvgHR; got != 119 {
		t.Errorf("band170.early_avg_hr = %d, want 119", got)
	}
	if got := *band170.LateAvgHR; got != 111 {
		t.Errorf("band170.late_avg_hr = %d, want 111", got)
	}
	if got := *band170.HRDelta; got != -8 {
		t.Errorf("band170.hr_delta = %d, want -8", got)
	}
	if band170.Band != "2:50.0–2:55.0" {
		t.Errorf("band170.band = %q, want %q", band170.Band, "2:50.0–2:55.0")
	}

	band165 := findBand(t, bands, 165)
	if band165.Workouts != 1 {
		t.Errorf("band165.workouts = %d, want 1", band165.Workouts)
	}
	if band165.HRDelta != nil {
		t.Errorf("band165.hr_delta = %v, want nil", *band165.HRDelta)
	}
}

func findBand(t *testing.T, bands []HRPaceBand, start int) HRPaceBand {
	t.Helper()
	for _, b := range bands {
		if b.BandStartSeconds == start {
			return b
		}
	}
	t.Fatalf("no band with band_start_seconds = %d", start)
	return HRPaceBand{}
}
