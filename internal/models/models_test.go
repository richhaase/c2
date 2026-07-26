package models

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/richhaase/c2/internal/jsonx"
)

func makeWorkout(mutate func(*Workout)) Workout {
	w := Workout{
		ID:            1,
		UserID:        1,
		Date:          "2026-03-07 09:21:00",
		Distance:      5500,
		Type:          "rower",
		Time:          19122,
		TimeFormatted: "31:52.2",
	}
	if mutate != nil {
		mutate(&w)
	}
	return w
}

func TestParsedDate(t *testing.T) {
	d := ParsedDate(makeWorkout(nil))
	if d.Year() != 2026 || d.Month() != 3 || d.Day() != 7 || d.Hour() != 9 || d.Minute() != 21 {
		t.Fatalf("got %v", d)
	}
}

func TestCalendarDay(t *testing.T) {
	if got := CalendarDay(makeWorkout(nil)); got != "2026-03-07" {
		t.Fatalf("got %q", got)
	}
}

func TestPace500mSeconds(t *testing.T) {
	if got := Pace500mSeconds(makeWorkout(nil)); math.Abs(got-173.836) > 0.005 {
		t.Fatalf("got %v", got)
	}
	if got := Pace500mSeconds(makeWorkout(func(w *Workout) { w.Distance = 0 })); got != 0 {
		t.Fatalf("zero distance: got %v", got)
	}
	if got := Pace500mSeconds(makeWorkout(func(w *Workout) { w.Time = 0 })); got != 0 {
		t.Fatalf("zero time: got %v", got)
	}
}

func TestPace500m(t *testing.T) {
	if got := Pace500m(makeWorkout(nil)); got != "2:53.8" {
		t.Fatalf("got %q", got)
	}
	dash := makeWorkout(func(w *Workout) { w.Distance = 0; w.Time = 0 })
	if got := Pace500m(dash); got != "-" {
		t.Fatalf("got %q", got)
	}
	slow := makeWorkout(func(w *Workout) { w.Distance = 1000; w.Time = 3706 })
	if got := Pace500m(slow); got != "3:05.3" {
		t.Fatalf("got %q", got)
	}
	interval := makeWorkout(func(w *Workout) {
		w.Distance = 3000
		w.Time = 8626
		w.TimeFormatted = "20:22.6"
		w.WorkoutType = "FixedDistanceInterval"
		w.RestTime = new(3600)
		w.RestDistance = new(660)
	})
	if got := Pace500m(interval); got != "2:23.8" {
		t.Fatalf("interval work pace: got %q", got)
	}
}

func TestIsIntervalWorkout(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Workout)
		want   bool
	}{
		{"continuous piece", func(w *Workout) { w.WorkoutType = "FixedDistanceSplits" }, false},
		{"fixed distance interval", func(w *Workout) { w.WorkoutType = "FixedDistanceInterval" }, true},
		{"fixed time interval", func(w *Workout) { w.WorkoutType = "FixedTimeInterval" }, true},
		{"rest time only", func(w *Workout) { w.RestTime = new(3600) }, true},
		{"rest distance only", func(w *Workout) { w.RestDistance = new(660) }, true},
		{"explicit zero rest", func(w *Workout) {
			w.WorkoutType = "FixedDistanceSplits"
			w.RestTime = new(0)
			w.RestDistance = new(0)
		}, false},
		{"no type and no rest", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsIntervalWorkout(makeWorkout(tc.mutate)); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestRestSeconds(t *testing.T) {
	if got := RestSeconds(makeWorkout(nil)); got != 0 {
		t.Fatalf("got %v", got)
	}
	if got := RestSeconds(makeWorkout(func(w *Workout) { w.RestTime = new(3600) })); got != 360 {
		t.Fatalf("got %v", got)
	}
}

func TestFormatSeconds(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "0:00.0"},
		{-5, "0:00.0"},
		{5.5, "0:05.5"},
		{65.3, "1:05.3"},
		{360, "6:00.0"},
		{862.6, "14:22.6"},
		{59.94, "0:59.9"},
		{59.95, "1:00.0"},
		{59.96, "1:00.0"},
		{119.94, "1:59.9"},
		{119.95, "2:00.0"},
		{119.96, "2:00.0"},
	}
	for _, tc := range cases {
		if got := FormatSeconds(tc.in); got != tc.want {
			t.Errorf("FormatSeconds(%v) = %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsValidYMD(t *testing.T) {
	valid := []string{"2026-03-07", "2024-02-29", "2026-12-31"}
	for _, s := range valid {
		if !IsValidYMD(s) {
			t.Errorf("expected %q valid", s)
		}
	}
	invalid := []string{"2026-3-7", "2026-02-30", "2025-02-29", "not-a-date", "", "2026-13-01"}
	for _, s := range invalid {
		if IsValidYMD(s) {
			t.Errorf("expected %q invalid", s)
		}
	}
}

func TestFilterByDate(t *testing.T) {
	workouts := []Workout{
		{ID: 1, Date: "2026-01-15 10:00:00"},
		{ID: 2, Date: "2026-02-15 10:00:00"},
		{ID: 3, Date: "2026-03-15 10:00:00"},
	}
	if got := FilterByDate(workouts, "", ""); len(got) != 3 {
		t.Fatalf("no bounds: got %d", len(got))
	}
	got := FilterByDate(workouts, "2026-02-01", "2026-02-28")
	if len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("got %+v", got)
	}
	if got := FilterByDate(workouts, "2026-02-15", ""); len(got) != 2 {
		t.Fatalf("inclusive from: got %d", len(got))
	}
	if got := FilterByDate(workouts, "", "2026-02-15"); len(got) != 2 {
		t.Fatalf("inclusive to: got %d", len(got))
	}
}

func TestResolveWorkout(t *testing.T) {
	workouts := []Workout{
		{ID: 1, Date: "2026-01-15 10:00:00"},
		{ID: 3, Date: "2026-03-15 10:00:00"},
		{ID: 2, Date: "2026-02-15 10:00:00"},
	}
	if got := ResolveWorkout(workouts, "last"); got == nil || got.ID != 3 {
		t.Fatalf("last: got %+v", got)
	}
	if got := ResolveWorkout(workouts, "2"); got == nil || got.ID != 2 {
		t.Fatalf("by id: got %+v", got)
	}
	if got := ResolveWorkout(workouts, "999"); got != nil {
		t.Fatalf("missing id: got %+v", got)
	}
	if got := ResolveWorkout(workouts, "abc"); got != nil {
		t.Fatalf("non-numeric: got %+v", got)
	}
	if got := ResolveWorkout(nil, "last"); got != nil {
		t.Fatalf("empty last: got %+v", got)
	}
}

func TestWorkoutPreservesUnknownFieldsOnRoundTrip(t *testing.T) {
	line := `{"id":118212501,"user_id":7,"date":"2026-03-07 09:21:00","distance":5500,"type":"rower","time":19122,"time_formatted":"31:52.2","verified":true,"ranked":false,"nickname":"morning row"}`
	var w Workout
	if err := json.Unmarshal([]byte(line), &w); err != nil {
		t.Fatal(err)
	}
	if w.ID != 118212501 || w.Distance != 5500 {
		t.Fatalf("parsed fields wrong: %+v", w)
	}
	out, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != line {
		t.Fatalf("round trip lost data:\n got %s\nwant %s", out, line)
	}
}

func TestWorkoutMarshalsWithoutRaw(t *testing.T) {
	w := Workout{ID: 5, UserID: 1, Date: "2026-03-07 09:21:00", Distance: 100, Type: "rower", Time: 10, TimeFormatted: "0:01.0"}
	out, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	var back Workout
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatal(err)
	}
	if back.ID != 5 || back.Distance != 100 {
		t.Fatalf("got %+v", back)
	}
}

func TestFormatSecondsRoundsHalvesAwayFromZero(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{174.25, "2:54.3"},
		{174.35, "2:54.3"},
		{54.25, "0:54.3"},
		{125.05, "2:05.0"},
		{0.05, "0:00.1"},
		{2.5, "0:02.5"},
	}
	for _, tc := range cases {
		if got := FormatSeconds(tc.in); got != tc.want {
			t.Errorf("FormatSeconds(%v) = %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestWorkoutWithoutRawDoesNotHTMLEscape(t *testing.T) {
	w := Workout{ID: 1, Date: "2026-01-01 08:00:00", Comments: "pace < 2:00 & HR > 150"}
	out, err := jsonx.Compact(w)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "pace < 2:00 & HR > 150") {
		t.Fatalf("comments were HTML-escaped through jsonx: %s", out)
	}
}

func TestWorkoutWithRawIsUnescapedThroughJSONX(t *testing.T) {
	line := `{"id":1,"user_id":1,"date":"2026-01-01 08:00:00","distance":1,"type":"rower","time":1,"time_formatted":"0:00.1","comments":"pace < 2:00 & HR > 150"}`
	var w Workout
	if err := json.Unmarshal([]byte(line), &w); err != nil {
		t.Fatal(err)
	}
	out, err := jsonx.Compact(w)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != line {
		t.Fatalf("raw round trip changed:\n got %s\nwant %s", out, line)
	}
}

func TestStrokeDataPreservesRawFieldOrderAndUnknownFields(t *testing.T) {
	line := `{"d":17,"p":0,"hr":72,"spm":0,"t":8,"future_field":1}`
	var s StrokeData
	if err := json.Unmarshal([]byte(line), &s); err != nil {
		t.Fatal(err)
	}
	if s.HR == nil || *s.HR != 72 {
		t.Fatalf("parsed fields wrong: %+v", s)
	}
	out, err := jsonx.Compact(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != line {
		t.Fatalf("stroke round trip changed:\n got %s\nwant %s", out, line)
	}
}

func TestToFixedRoundsTheDecimalTheFloatIdentifies(t *testing.T) {
	cases := []struct {
		in     float64
		digits int
		want   string
	}{
		{(11500.0 / 1000000.0) * 100, 1, "1.2"},
		{(449375.0 / 1000000.0) * 100, 1, "44.9"},
		{1.25, 1, "1.3"},
		{2.675, 2, "2.68"},
		{54.25, 1, "54.3"},
		{54.349999999999994, 1, "54.3"},
		{0.05, 1, "0.1"},
	}
	for _, tc := range cases {
		if got := ToFixed(tc.in, tc.digits); got != tc.want {
			t.Errorf("ToFixed(%v, %d) = %q want %q", tc.in, tc.digits, got, tc.want)
		}
	}
}
