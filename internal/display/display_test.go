package display

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/richhaase/c2/internal/models"
)

func intPtr(v int) *int {
	return &v
}

func makeWorkout() models.Workout {
	return models.Workout{
		ID:            1,
		UserID:        1,
		Date:          "2026-04-09 07:00:00",
		Distance:      5000,
		Type:          "rower",
		Time:          17155,
		TimeFormatted: "28:35.4",
		WorkoutType:   "FixedDistanceSplits",
		StrokeRate:    intPtr(24),
		HeartRate:     &models.HeartRate{Average: intPtr(112)},
		DragFactor:    intPtr(107),
	}
}

func TestFormatMeters(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{500, "500"},
		{1000, "1,000"},
		{12345, "12,345"},
		{1000000, "1,000,000"},
		{100, "100"},
		{999, "999"},
		{1234567, "1,234,567"},
		{-1234, "-1,234"},
	}
	for _, c := range cases {
		if got := FormatMeters(c.in); got != c.want {
			t.Errorf("FormatMeters(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatPercent(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "0.0%"},
		{0.5, "50.0%"},
		{1.0, "100.0%"},
		{0.1234, "12.3%"},
		{0.131, "13.1%"},
		{0.1225, "12.3%"},
		{0.0625, "6.3%"},
		{0.7055, "70.6%"},
	}
	for _, c := range cases {
		if got := FormatPercent(c.in); got != c.want {
			t.Errorf("FormatPercent(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatMetersPerWeek(t *testing.T) {
	if got := FormatMetersPerWeek(20212); got != "20,212m/week" {
		t.Errorf("FormatMetersPerWeek(20212) = %q, want %q", got, "20,212m/week")
	}
}

func TestFormatDate(t *testing.T) {
	d := time.Date(2026, time.April, 9, 7, 0, 0, 0, time.Local)
	if got := FormatDate(d, "%Y-%m-%d"); got != "2026-04-09" {
		t.Errorf("FormatDate(%%Y-%%m-%%d) = %q, want %q", got, "2026-04-09")
	}
	if got := FormatDate(d, "%m/%d"); got != "04/09" {
		t.Errorf("FormatDate(%%m/%%d) = %q, want %q", got, "04/09")
	}
	if got := FormatDate(d, "anything else"); got != "04/09" {
		t.Errorf("FormatDate(fallback) = %q, want %q", got, "04/09")
	}
}

func TestSparkBar(t *testing.T) {
	if got := SparkBar(100, 0); got != "" {
		t.Errorf("SparkBar(100, 0) = %q, want empty string", got)
	}

	want := strings.Repeat("█", 20)
	if got := SparkBar(100, 100); got != want {
		t.Errorf("SparkBar(100, 100) = %q, want %q", got, want)
	}

	half := SparkBar(50, 100)
	if n := utf8.RuneCountInString(half); n != 20 {
		t.Errorf("SparkBar(50, 100) has %d runes, want 20", n)
	}
	wantHalf := strings.Repeat("█", 10) + strings.Repeat("░", 10)
	if half != wantHalf {
		t.Errorf("SparkBar(50, 100) = %q, want %q", half, wantHalf)
	}
}

func TestTrendArrow(t *testing.T) {
	cases := []struct {
		prev, curr float64
		want       string
		desc       string
	}{
		{0, 100, " ", "space when prev is 0"},
		{100, 110, "↑", "up arrow for increase"},
		{100, 90, "↓", "down arrow for decrease"},
		{100, 101, "→", "right arrow for stable"},
	}
	for _, c := range cases {
		if got := TrendArrow(c.prev, c.curr); got != c.want {
			t.Errorf("TrendArrow(%v, %v) = %q, want %q (%s)", c.prev, c.curr, got, c.want, c.desc)
		}
	}
}

func TestPaceArrow(t *testing.T) {
	if got := PaceArrow(180, 170); got != "↑" {
		t.Errorf("PaceArrow(180, 170) = %q, want %q (lower pace is an improvement)", got, "↑")
	}
	if got := PaceArrow(170, 180); got != "↓" {
		t.Errorf("PaceArrow(170, 180) = %q, want %q", got, "↓")
	}
	if got := PaceArrow(0, 170); got != " " {
		t.Errorf("PaceArrow(0, 170) = %q, want a single space", got)
	}
}

func TestFormatIntervalTag(t *testing.T) {
	t.Run("continuous piece", func(t *testing.T) {
		w := makeWorkout()
		w.WorkoutType = "FixedDistanceSplits"
		if got := FormatIntervalTag(w); got != "" {
			t.Errorf("FormatIntervalTag = %q, want empty string", got)
		}
	})

	t.Run("interval with rest duration", func(t *testing.T) {
		w := makeWorkout()
		w.WorkoutType = "FixedDistanceInterval"
		w.RestTime = intPtr(3600)
		if got := FormatIntervalTag(w); got != "[IVL rest 6:00.0]" {
			t.Errorf("FormatIntervalTag = %q, want %q", got, "[IVL rest 6:00.0]")
		}
	})

	t.Run("interval type without rest_time", func(t *testing.T) {
		w := makeWorkout()
		w.WorkoutType = "FixedDistanceInterval"
		if got := FormatIntervalTag(w); got != "[IVL]" {
			t.Errorf("FormatIntervalTag = %q, want %q", got, "[IVL]")
		}
	})

	t.Run("interval detected via rest_distance alone", func(t *testing.T) {
		w := makeWorkout()
		w.WorkoutType = ""
		w.RestDistance = intPtr(660)
		if got := FormatIntervalTag(w); got != "[IVL]" {
			t.Errorf("FormatIntervalTag = %q, want %q", got, "[IVL]")
		}
	})
}

func TestFormatWorkoutLine(t *testing.T) {
	t.Run("continuous piece without an interval tag", func(t *testing.T) {
		w := makeWorkout()
		line := FormatWorkoutLine(w, "%m/%d")

		want := "04/09   5,000m   28:35.4   2:51.6/500m  24spm  112bpm  107df"
		if line != want {
			t.Errorf("FormatWorkoutLine =\n%q\nwant\n%q", line, want)
		}
		for _, part := range []string{"04/09", "5,000m", "28:35.4", "2:51.6/500m", "24spm", "112bpm", "107df"} {
			if !strings.Contains(line, part) {
				t.Errorf("line %q does not contain %q", line, part)
			}
		}
		if strings.Contains(line, "[IVL") {
			t.Errorf("line %q should not contain an interval tag", line)
		}
	})

	t.Run("ISO date format", func(t *testing.T) {
		w := makeWorkout()
		line := FormatWorkoutLine(w, "%Y-%m-%d")
		want := "2026-04-09   5,000m   28:35.4   2:51.6/500m  24spm  112bpm  107df"
		if line != want {
			t.Errorf("FormatWorkoutLine =\n%q\nwant\n%q", line, want)
		}
	})

	t.Run("appends interval tag", func(t *testing.T) {
		w := makeWorkout()
		w.Date = "2026-04-11 09:14:00"
		w.Distance = 3000
		w.Time = 8626
		w.TimeFormatted = "20:22.6"
		w.WorkoutType = "FixedDistanceInterval"
		w.RestTime = intPtr(3600)
		w.RestDistance = intPtr(660)
		w.StrokeRate = intPtr(30)
		w.HeartRate = &models.HeartRate{Average: intPtr(152)}
		w.DragFactor = intPtr(108)

		line := FormatWorkoutLine(w, "%m/%d")
		want := "04/11   3,000m   20:22.6   2:23.8/500m  30spm  152bpm  108df  [IVL rest 6:00.0]"
		if line != want {
			t.Errorf("FormatWorkoutLine =\n%q\nwant\n%q", line, want)
		}
		if !strings.Contains(line, "[IVL rest 6:00.0]") {
			t.Errorf("line %q missing interval tag", line)
		}
		if !strings.Contains(line, "2:23.8/500m") {
			t.Errorf("line %q missing pace", line)
		}
	})

	t.Run("missing stroke_rate, heart_rate and drag_factor", func(t *testing.T) {
		w := makeWorkout()
		w.StrokeRate = nil
		w.HeartRate = nil
		w.DragFactor = nil

		line := FormatWorkoutLine(w, "%m/%d")
		want := "04/09   5,000m   28:35.4   2:51.6/500m      -       -     -"
		if line != want {
			t.Errorf("FormatWorkoutLine =\n%q\nwant\n%q", line, want)
		}
		if !strings.Contains(line, "    -") {
			t.Errorf("line %q missing padded placeholder", line)
		}
	})

	t.Run("zero heart rate average renders as placeholder", func(t *testing.T) {
		w := makeWorkout()
		w.HeartRate = &models.HeartRate{Average: intPtr(0)}
		line := FormatWorkoutLine(w, "%m/%d")
		want := "04/09   5,000m   28:35.4   2:51.6/500m  24spm       -  107df"
		if line != want {
			t.Errorf("FormatWorkoutLine =\n%q\nwant\n%q", line, want)
		}
	})

	t.Run("overflowing fields are not truncated", func(t *testing.T) {
		w := makeWorkout()
		w.Distance = 1234567
		w.TimeFormatted = "1:28:35.4"
		line := FormatWorkoutLine(w, "%m/%d")
		want := "04/09  1,234,567m  1:28:35.4   0:00.7/500m  24spm  112bpm  107df"
		if line != want {
			t.Errorf("FormatWorkoutLine =\n%q\nwant\n%q", line, want)
		}
	})
}

func TestWorkoutJSON(t *testing.T) {
	cases := []struct {
		name    string
		workout func() models.Workout
		want    string
	}{
		{
			name:    "continuous piece",
			workout: makeWorkout,
			want:    `{"id":1,"date":"2026-04-09 07:00:00","distance":5000,"time_tenths":17155,"time_formatted":"28:35.4","pace_500m":"2:51.6","pace_500m_seconds":171.6,"stroke_rate":24,"hr_avg":112,"drag_factor":107,"workout_type":"FixedDistanceSplits","interval":false,"rest_seconds":null,"comments":null,"has_stroke_data":false}`,
		},
		{
			name: "interval workout",
			workout: func() models.Workout {
				w := makeWorkout()
				w.Date = "2026-04-11 09:14:00"
				w.Distance = 3000
				w.Time = 8626
				w.TimeFormatted = "20:22.6"
				w.WorkoutType = "FixedDistanceInterval"
				w.RestTime = intPtr(3600)
				w.RestDistance = intPtr(660)
				w.StrokeRate = intPtr(30)
				w.HeartRate = &models.HeartRate{Average: intPtr(152)}
				w.DragFactor = intPtr(108)
				return w
			},
			want: `{"id":1,"date":"2026-04-11 09:14:00","distance":3000,"time_tenths":8626,"time_formatted":"20:22.6","pace_500m":"2:23.8","pace_500m_seconds":143.8,"stroke_rate":30,"hr_avg":152,"drag_factor":108,"workout_type":"FixedDistanceInterval","interval":true,"rest_seconds":360,"comments":null,"has_stroke_data":false}`,
		},
		{
			name: "empty workout nulls every optional field",
			workout: func() models.Workout {
				return models.Workout{
					ID:            9,
					UserID:        1,
					Date:          "2026-01-02 06:00:00",
					Distance:      0,
					Type:          "rower",
					Time:          0,
					TimeFormatted: "0:00.0",
				}
			},
			want: `{"id":9,"date":"2026-01-02 06:00:00","distance":0,"time_tenths":0,"time_formatted":"0:00.0","pace_500m":null,"pace_500m_seconds":null,"stroke_rate":null,"hr_avg":null,"drag_factor":null,"workout_type":null,"interval":false,"rest_seconds":null,"comments":null,"has_stroke_data":false}`,
		},
		{
			name: "comments and stroke data",
			workout: func() models.Workout {
				w := makeWorkout()
				w.Comments = "felt strong"
				w.StrokeData = true
				return w
			},
			want: `{"id":1,"date":"2026-04-09 07:00:00","distance":5000,"time_tenths":17155,"time_formatted":"28:35.4","pace_500m":"2:51.6","pace_500m_seconds":171.6,"stroke_rate":24,"hr_avg":112,"drag_factor":107,"workout_type":"FixedDistanceSplits","interval":false,"rest_seconds":null,"comments":"felt strong","has_stroke_data":true}`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			encoded, err := json.Marshal(WorkoutJSON(c.workout()))
			if err != nil {
				t.Fatalf("json.Marshal: %v", err)
			}
			if string(encoded) != c.want {
				t.Errorf("WorkoutJSON =\n%s\nwant\n%s", encoded, c.want)
			}
		})
	}
}
