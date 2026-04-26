package model

import "testing"

func TestParsedDate(t *testing.T) {
	w := Workout{Date: "2026-03-07 09:21:00"}
	d, err := ParsedDate(w)
	if err != nil {
		t.Fatalf("ParsedDate() error = %v", err)
	}
	if d.Year() != 2026 || d.Month() != 3 || d.Day() != 7 || d.Hour() != 9 || d.Minute() != 21 {
		t.Fatalf("ParsedDate() = %v", d)
	}
}

func TestCalendarDay(t *testing.T) {
	w := Workout{Date: "2026-03-02 17:41:00"}
	if got := CalendarDay(w); got != "2026-03-02" {
		t.Fatalf("CalendarDay() = %q", got)
	}
}

func TestCalendarDayShortDate(t *testing.T) {
	w := Workout{Date: "bad"}
	if got := CalendarDay(w); got != "bad" {
		t.Fatalf("CalendarDay() = %q", got)
	}
}

func TestIsIntervalWorkout(t *testing.T) {
	rest := 600
	restDistance := 12
	tests := []struct {
		name string
		w    Workout
		want bool
	}{
		{name: "continuous", w: Workout{WorkoutType: "FixedDistance"}, want: false},
		{name: "type interval", w: Workout{WorkoutType: "FixedDistanceInterval"}, want: true},
		{name: "time interval type", w: Workout{WorkoutType: "FixedTimeInterval"}, want: true},
		{name: "rest time", w: Workout{RestTime: &rest}, want: true},
		{name: "rest distance", w: Workout{RestDistance: &restDistance}, want: true},
		{name: "explicit zero rest", w: Workout{WorkoutType: "FixedDistanceSplits", RestTime: ptr(0), RestDistance: ptr(0)}, want: false},
		{name: "missing workout type and no rest", w: Workout{}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsIntervalWorkout(tt.w); got != tt.want {
				t.Fatalf("IsIntervalWorkout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func ptr(v int) *int {
	return &v
}
