package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/richhaase/c2/internal/model"
)

func TestReadWorkoutsMissingFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "workouts.jsonl")

	workouts, err := ReadWorkoutsPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(workouts) != 0 {
		t.Fatalf("ReadWorkoutsPath() returned %d workouts, want 0", len(workouts))
	}
}

func TestAppendWorkoutsAddsNewlineAfterExistingFinalRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "workouts.jsonl")
	existing := model.Workout{ID: 101, Date: "2026-04-01 07:30:00", Distance: 2000, Type: "rower", Time: 4820, TimeFormatted: "8:02.0"}
	data, err := json.Marshal(existing)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	written, err := AppendWorkoutsPath(path, []model.Workout{
		{ID: 102, Date: "2026-04-02 07:30:00", Distance: 5000, Type: "rower", Time: 12500, TimeFormatted: "20:50.0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if written != 1 {
		t.Fatalf("AppendWorkoutsPath() wrote %d workouts, want 1", written)
	}

	workouts, err := ReadWorkoutsPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(workouts) != 2 {
		t.Fatalf("ReadWorkoutsPath() returned %d workouts, want 2", len(workouts))
	}
}

func TestReadWorkoutsInvalidJSONIncludesPathAndLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "workouts.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{\"id\":101}\nnot-json\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadWorkoutsPath(path)
	if err == nil {
		t.Fatal("ReadWorkoutsPath() returned nil error")
	}
	message := err.Error()
	if !strings.Contains(message, path) || !strings.Contains(message, "line 2") {
		t.Fatalf("ReadWorkoutsPath() error = %q, want path and line number", message)
	}
}

func TestReadWorkoutsSupportsLongLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "workouts.jsonl")
	workout := model.Workout{
		ID:            101,
		Date:          "2026-04-01 07:30:00",
		Distance:      2000,
		Type:          "rower",
		Time:          4820,
		TimeFormatted: "8:02.0",
		Comments:      strings.Repeat("x", 70*1024),
	}
	data, err := json.Marshal(workout)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	workouts, err := ReadWorkoutsPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(workouts) != 1 {
		t.Fatalf("ReadWorkoutsPath() returned %d workouts, want 1", len(workouts))
	}
	if workouts[0].Comments != workout.Comments {
		t.Fatalf("Comments length = %d, want %d", len(workouts[0].Comments), len(workout.Comments))
	}
}

func TestAppendWorkoutsSkipsDuplicateIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "workouts.jsonl")
	initial := []model.Workout{
		{ID: 101, Date: "2026-04-01 07:30:00", Distance: 2000, Type: "rower", Time: 4820, TimeFormatted: "8:02.0"},
		{ID: 102, Date: "2026-04-02 07:30:00", Distance: 5000, Type: "rower", Time: 12500, TimeFormatted: "20:50.0"},
	}

	written, err := AppendWorkoutsPath(path, initial)
	if err != nil {
		t.Fatal(err)
	}
	if written != 2 {
		t.Fatalf("first AppendWorkoutsPath() wrote %d workouts, want 2", written)
	}

	written, err = AppendWorkoutsPath(path, []model.Workout{
		{ID: 101, Date: "2026-04-01 07:30:00", Distance: 2000, Type: "rower", Time: 4820, TimeFormatted: "8:02.0"},
		{ID: 103, Date: "2026-04-03 07:30:00", Distance: 1000, Type: "rower", Time: 2450, TimeFormatted: "4:05.0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if written != 1 {
		t.Fatalf("second AppendWorkoutsPath() wrote %d workouts, want 1", written)
	}

	workouts, err := ReadWorkoutsPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(workouts) != 3 {
		t.Fatalf("ReadWorkoutsPath() returned %d workouts, want 3", len(workouts))
	}
	for i, wantID := range []int{101, 102, 103} {
		if workouts[i].ID != wantID {
			t.Fatalf("workouts[%d].ID = %d, want %d", i, workouts[i].ID, wantID)
		}
	}
}
