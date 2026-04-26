package model

import "testing"

func TestPace500mSeconds(t *testing.T) {
	w := Workout{Distance: 2000, Time: 4800}
	if got := Pace500mSeconds(w); got != 120 {
		t.Fatalf("Pace500mSeconds() = %v, want 120", got)
	}
}

func TestPace500mSecondsZeroValues(t *testing.T) {
	tests := []struct {
		name string
		w    Workout
	}{
		{name: "zero distance", w: Workout{Distance: 0, Time: 4800}},
		{name: "zero time", w: Workout{Distance: 2000, Time: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Pace500mSeconds(tt.w); got != 0 {
				t.Fatalf("Pace500mSeconds() = %v, want 0", got)
			}
		})
	}
}

func TestPace500m(t *testing.T) {
	tests := []struct {
		name string
		w    Workout
		want string
	}{
		{name: "formats pace", w: Workout{Distance: 5500, Time: 19122}, want: "2:53.8"},
		{name: "zero values", w: Workout{Distance: 0, Time: 0}, want: "-"},
		{name: "three minute pace", w: Workout{Distance: 1000, Time: 3706}, want: "3:05.3"},
		{name: "uses work time for interval pace", w: Workout{Distance: 3000, Time: 8626, TimeFormatted: "20:22.6", WorkoutType: "FixedDistanceInterval", RestTime: ptr(3600), RestDistance: ptr(660)}, want: "2:23.8"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Pace500m(tt.w); got != tt.want {
				t.Fatalf("Pace500m() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRestSeconds(t *testing.T) {
	tests := []struct {
		name string
		w    Workout
		want float64
	}{
		{name: "missing", w: Workout{}, want: 0},
		{name: "converts tenths", w: Workout{RestTime: ptr(3600)}, want: 360},
		{name: "explicit zero", w: Workout{RestTime: ptr(0)}, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RestSeconds(tt.w); got != tt.want {
				t.Fatalf("RestSeconds() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorkSeconds(t *testing.T) {
	w := Workout{Time: 8626}
	if got := WorkSeconds(w); got != 862.6 {
		t.Fatalf("WorkSeconds() = %v, want 862.6", got)
	}
}

func TestFormatSeconds(t *testing.T) {
	tests := map[float64]string{
		-5:    "0:00.0",
		0:     "0:00.0",
		5.5:   "0:05.5",
		65.3:  "1:05.3",
		360:   "6:00.0",
		862.6: "14:22.6",
	}
	for input, want := range tests {
		if got := FormatSeconds(input); got != want {
			t.Fatalf("FormatSeconds(%v) = %q, want %q", input, got, want)
		}
	}
}
