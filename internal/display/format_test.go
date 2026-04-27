package display

import (
	"strings"
	"testing"

	"github.com/richhaase/c2/internal/model"
)

func makeDisplayWorkout(overrides model.Workout) model.Workout {
	strokeRate := 24
	heartRate := 112
	dragFactor := 107
	w := model.Workout{
		ID:            1,
		UserID:        1,
		Date:          "2026-04-09 07:00:00",
		Distance:      5000,
		Type:          "rower",
		Time:          17155,
		TimeFormatted: "28:35.4",
		WorkoutType:   "FixedDistanceSplits",
		StrokeRate:    &strokeRate,
		HeartRate:     &model.HeartRate{Average: &heartRate},
		DragFactor:    &dragFactor,
	}
	if overrides.Date != "" {
		w.Date = overrides.Date
	}
	if overrides.Distance != 0 {
		w.Distance = overrides.Distance
	}
	if overrides.Time != 0 {
		w.Time = overrides.Time
	}
	if overrides.TimeFormatted != "" {
		w.TimeFormatted = overrides.TimeFormatted
	}
	if overrides.WorkoutType != "" {
		w.WorkoutType = overrides.WorkoutType
	}
	if overrides.StrokeRate != nil {
		w.StrokeRate = overrides.StrokeRate
	}
	if overrides.HeartRate != nil {
		w.HeartRate = overrides.HeartRate
	}
	if overrides.DragFactor != nil {
		w.DragFactor = overrides.DragFactor
	}
	if overrides.RestTime != nil {
		w.RestTime = overrides.RestTime
	}
	if overrides.RestDistance != nil {
		w.RestDistance = overrides.RestDistance
	}
	return w
}

func intPtr(v int) *int {
	return &v
}

func TestFormatMeters(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{input: 0, want: "0"},
		{input: 500, want: "500"},
		{input: 1000, want: "1,000"},
		{input: 12345, want: "12,345"},
		{input: 1000000, want: "1,000,000"},
	}
	for _, tt := range tests {
		if got := FormatMeters(tt.input); got != tt.want {
			t.Fatalf("FormatMeters(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatPercent(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{input: 0, want: "0.0%"},
		{input: 0.5, want: "50.0%"},
		{input: 1.0, want: "100.0%"},
		{input: 0.1234, want: "12.3%"},
		{input: 0.131, want: "13.1%"},
	}
	for _, tt := range tests {
		if got := FormatPercent(tt.input); got != tt.want {
			t.Fatalf("FormatPercent(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatMetersPerWeek(t *testing.T) {
	if got := FormatMetersPerWeek(20212); got != "20,212m/week" {
		t.Fatalf("FormatMetersPerWeek() = %q, want 20,212m/week", got)
	}
}

func TestFormatIntervalTag(t *testing.T) {
	tests := []struct {
		name string
		w    model.Workout
		want string
	}{
		{name: "continuous piece", w: makeDisplayWorkout(model.Workout{WorkoutType: "FixedDistanceSplits"}), want: ""},
		{name: "tag with rest duration", w: makeDisplayWorkout(model.Workout{WorkoutType: "FixedDistanceInterval", RestTime: intPtr(3600)}), want: "[IVL rest 6:00.0]"},
		{name: "bare tag when rest time missing", w: makeDisplayWorkout(model.Workout{WorkoutType: "FixedDistanceInterval"}), want: "[IVL]"},
		{name: "detects interval via rest distance", w: makeDisplayWorkout(model.Workout{RestDistance: intPtr(660)}), want: "[IVL]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatIntervalTag(tt.w); got != tt.want {
				t.Fatalf("FormatIntervalTag() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatWorkoutLineContinuousPiece(t *testing.T) {
	w := makeDisplayWorkout(model.Workout{
		Date:          "2026-04-09 07:00:00",
		Distance:      5000,
		Time:          17155,
		TimeFormatted: "28:35.4",
		WorkoutType:   "FixedDistanceSplits",
		StrokeRate:    intPtr(24),
		HeartRate:     &model.HeartRate{Average: intPtr(112)},
		DragFactor:    intPtr(107),
	})
	line := FormatWorkoutLine(w, "01/02")
	for _, want := range []string{"04/09", "5,000m", "28:35.4", "2:51.6/500m", "24spm", "112bpm", "107df"} {
		if !strings.Contains(line, want) {
			t.Fatalf("FormatWorkoutLine() = %q, want to contain %q", line, want)
		}
	}
	if strings.Contains(line, "[IVL") {
		t.Fatalf("FormatWorkoutLine() = %q, want no interval tag", line)
	}
}

func TestFormatWorkoutLineAppendsIntervalRestTag(t *testing.T) {
	w := makeDisplayWorkout(model.Workout{
		Date:          "2026-04-11 09:14:00",
		Distance:      3000,
		Time:          8626,
		TimeFormatted: "20:22.6",
		WorkoutType:   "FixedDistanceInterval",
		RestTime:      intPtr(3600),
		RestDistance:  intPtr(660),
		StrokeRate:    intPtr(30),
		HeartRate:     &model.HeartRate{Average: intPtr(152)},
		DragFactor:    intPtr(108),
	})
	line := FormatWorkoutLine(w, "01/02")
	if !strings.Contains(line, "[IVL rest 6:00.0]") {
		t.Fatalf("FormatWorkoutLine() = %q, want interval rest tag", line)
	}
	if !strings.Contains(line, "2:23.8/500m") {
		t.Fatalf("FormatWorkoutLine() = %q, want work-time pace", line)
	}
}

func TestFormatWorkoutLineHandlesMissingOptionalMetrics(t *testing.T) {
	w := makeDisplayWorkout(model.Workout{})
	w.StrokeRate = nil
	w.HeartRate = nil
	w.DragFactor = nil
	line := FormatWorkoutLine(w, "01/02")
	if !strings.Contains(line, "    -") {
		t.Fatalf("FormatWorkoutLine() = %q, want padded placeholder", line)
	}
}

func TestFormatWorkoutLineTreatsZeroOptionalMetricsAsMissing(t *testing.T) {
	w := makeDisplayWorkout(model.Workout{
		StrokeRate: intPtr(0),
		DragFactor: intPtr(0),
	})
	line := FormatWorkoutLine(w, "01/02")
	if strings.Contains(line, "0spm") {
		t.Fatalf("FormatWorkoutLine() = %q, want zero stroke rate formatted as placeholder", line)
	}
	if strings.Contains(line, "0df") {
		t.Fatalf("FormatWorkoutLine() = %q, want zero drag factor formatted as placeholder", line)
	}
	if !strings.Contains(line, "    -") {
		t.Fatalf("FormatWorkoutLine() = %q, want padded placeholder", line)
	}
}
