package models

import (
	"encoding/json"
	"testing"
)

func TestWorkoutRoundtrip(t *testing.T) {
	w := Workout{
		ID:            339,
		UserID:        1,
		Date:          "2026-03-02 17:41:00",
		Distance:      5500,
		MachineType:   "rower",
		Time:          19122,
		TimeFormatted: "31:52.2",
		StrokeRate:    26,
		DragFactor:    83,
		HeartRate:     &HeartRate{Average: 118},
	}

	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed Workout
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if parsed.ID != 339 {
		t.Errorf("ID = %d, want 339", parsed.ID)
	}
	if parsed.Distance != 5500 {
		t.Errorf("Distance = %d, want 5500", parsed.Distance)
	}
	if parsed.HeartRate == nil || parsed.HeartRate.Average != 118 {
		t.Error("HeartRate not preserved")
	}
}

func TestParsedDate(t *testing.T) {
	w := Workout{Date: "2026-03-02 17:41:00"}
	dt, err := w.ParsedDate()
	if err != nil {
		t.Fatalf("ParsedDate: %v", err)
	}
	if dt.Year() != 2026 || dt.Month() != 3 || dt.Day() != 2 {
		t.Errorf("ParsedDate = %v, want 2026-03-02", dt)
	}
}

func TestPace500m(t *testing.T) {
	w := Workout{Time: 19122, Distance: 5500}
	pace := w.Pace500m()
	if pace != "2:53.8" {
		t.Errorf("Pace500m() = %q, want %q", pace, "2:53.8")
	}
}

func TestPace500mZero(t *testing.T) {
	w := Workout{Time: 0, Distance: 0}
	if w.Pace500m() != "-" {
		t.Errorf("Pace500m() with zero values should return '-'")
	}
}
