package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/paths"
)

const workoutLine = `{"id":1,"user_id":1,"date":"2026-07-01 08:00:00","distance":8000,"type":"rower","time":12000,"time_formatted":"20:00.0"}`

func tempPaths(t *testing.T) paths.DataPaths {
	t.Helper()
	p := paths.For(t.TempDir())
	if err := os.MkdirAll(p.StrokesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestReadWorkoutsMissingFile(t *testing.T) {
	p := paths.For(filepath.Join(t.TempDir(), "nope"))
	got, err := ReadWorkouts(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d", len(got))
	}
}

func TestAppendWorkoutsDedupesByID(t *testing.T) {
	p := tempPaths(t)
	var w models.Workout
	if err := json.Unmarshal([]byte(workoutLine), &w); err != nil {
		t.Fatal(err)
	}

	n, err := AppendWorkouts(p, []models.Workout{w})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("first append wrote %d", n)
	}

	n, err = AppendWorkouts(p, []models.Workout{w})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("duplicate append wrote %d", n)
	}

	count, err := WorkoutCount(p)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count = %d", count)
	}
}

func TestAppendWorkoutsDedupesWithinBatch(t *testing.T) {
	p := tempPaths(t)
	var w models.Workout
	if err := json.Unmarshal([]byte(workoutLine), &w); err != nil {
		t.Fatal(err)
	}
	n, err := AppendWorkouts(p, []models.Workout{w, w})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("wrote %d, want 1", n)
	}
}

func TestAppendWorkoutsPreservesUnknownFields(t *testing.T) {
	p := tempPaths(t)
	line := `{"id":9,"user_id":1,"date":"2026-07-01 08:00:00","distance":8000,"type":"rower","time":12000,"time_formatted":"20:00.0","nickname":"keep me"}`
	var w models.Workout
	if err := json.Unmarshal([]byte(line), &w); err != nil {
		t.Fatal(err)
	}
	if _, err := AppendWorkouts(p, []models.Workout{w}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(p.Workouts)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != line+"\n" {
		t.Fatalf("stored line lost fields:\n got %s\nwant %s", got, line)
	}
}

func TestUpsertWorkoutsAddsAndUpdatesByID(t *testing.T) {
	p := tempPaths(t)
	initial := []models.Workout{
		{ID: 1, Date: "2026-01-01", Distance: 1000},
		{ID: 2, Date: "2026-01-02", Distance: 2000},
	}
	if _, err := AppendWorkouts(p, initial); err != nil {
		t.Fatal(err)
	}
	result, err := UpsertWorkouts(p, []models.Workout{
		{ID: 2, Date: "2026-01-02", Distance: 2500},
		{ID: 3, Date: "2026-01-03", Distance: 3000},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Added != 1 || result.Updated != 1 {
		t.Fatalf("result = %#v", result)
	}
	workouts, err := ReadWorkouts(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(workouts) != 3 || workouts[1].Distance != 2500 || workouts[2].ID != 3 {
		t.Fatalf("workouts = %#v", workouts)
	}
}

func TestUpsertWorkoutsPreservesUnchangedRawRecords(t *testing.T) {
	p := tempPaths(t)
	raw := `{"id":1,"date":"2026-01-01","distance":1000,"future":"kept"}`
	var workout models.Workout
	if err := json.Unmarshal([]byte(raw), &workout); err != nil {
		t.Fatal(err)
	}
	if _, err := AppendWorkouts(p, []models.Workout{workout}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpsertWorkouts(p, []models.Workout{{ID: 2, Date: "2026-01-02", Distance: 2000}}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(p.Workouts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), raw+"\n") {
		t.Fatalf("data = %s", data)
	}
}

func TestStrokeDataRoundTrip(t *testing.T) {
	p := tempPaths(t)
	ok, err := HasStrokeData(p, 1)
	if err != nil || ok {
		t.Fatalf("expected no stroke data, got %v %v", ok, err)
	}
	strokes := []models.StrokeData{
		{T: new(1.0), D: new(500.0), P: new(1750.0), SPM: new(24.0), HR: new(110.0)},
		{T: new(2.0), D: new(1000.0)},
	}
	if err := WriteStrokeData(p, 1, strokes); err != nil {
		t.Fatal(err)
	}
	ok, err = HasStrokeData(p, 1)
	if err != nil || !ok {
		t.Fatalf("expected stroke data, got %v %v", ok, err)
	}
	back, err := ReadStrokeData(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(back) != 2 || back[0].P == nil || *back[0].P != 1750 {
		t.Fatalf("got %+v", back)
	}
	if back[1].P != nil {
		t.Fatalf("absent field should stay nil, got %v", *back[1].P)
	}
}

func TestReadStrokeDataSkipsCorruptLines(t *testing.T) {
	p := tempPaths(t)
	body := "{\"t\":1}\nnot json\n{\"t\":2}\n"
	if err := os.WriteFile(p.StrokeFile(7), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadStrokeData(p, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d", len(got))
	}
}

func TestMetaRoundTrip(t *testing.T) {
	p := tempPaths(t)
	if got := ReadMeta(p, nil); got != nil {
		t.Fatalf("missing meta should be nil, got %+v", got)
	}
	if err := WriteMeta(p, StoreMeta{SchemaVersion: new(SchemaVersion), Created: "2026-07-05T12:00:00.000Z"}); err != nil {
		t.Fatal(err)
	}
	got := ReadMeta(p, nil)
	if got == nil || got.SchemaVersion == nil || *got.SchemaVersion != 1 {
		t.Fatalf("got %+v", got)
	}
	if got.LastSync != "" {
		t.Fatalf("last_sync should be absent, got %q", got.LastSync)
	}
}

func TestReadMetaCorruptWarnsAndReturnsNil(t *testing.T) {
	p := tempPaths(t)
	if err := os.WriteFile(p.Meta, []byte("{ truncated"), 0o644); err != nil {
		t.Fatal(err)
	}
	var warnings []string
	if got := ReadMeta(p, func(s string) { warnings = append(warnings, s) }); got != nil {
		t.Fatalf("got %+v", got)
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v", warnings)
	}
}

func TestReadMetaForeignJSONHasNoSchemaVersion(t *testing.T) {
	p := tempPaths(t)
	if err := os.WriteFile(p.Meta, []byte(`{"name":"some other tool"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got := ReadMeta(p, nil)
	if got == nil {
		t.Fatal("expected parsed meta")
	}
	if got.SchemaVersion != nil {
		t.Fatalf("absent schema_version must stay nil, got %d", *got.SchemaVersion)
	}
}

func TestReadStrokeDataSkipsNullAndNonObjectLines(t *testing.T) {
	p := tempPaths(t)
	body := "{\"t\":1}\nnull\n[1,2]\n\"str\"\n123\n{\"t\":2}\n"
	if err := os.WriteFile(p.StrokeFile(9), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadStrokeData(p, 9)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d records, want 2 (null and non-objects must be skipped)", len(got))
	}
	for i, s := range got {
		if s.T == nil {
			t.Fatalf("record %d has no t; a phantom stroke leaked through", i)
		}
	}
}

func TestHasStrokeDataRequiresACompleteValidFile(t *testing.T) {
	p := tempPaths(t)
	if err := os.MkdirAll(p.StrokesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		body string
		want bool
	}{
		{"", false},
		{"null\n", false},
		{"{\"t\":1}\nnot-json\n", false},
		{"{\"t\":1}\n{\"t\":2}\n", true},
	}
	for i, tc := range cases {
		id := int64(i + 1)
		if err := os.WriteFile(p.StrokeFile(id), []byte(tc.body), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := HasStrokeData(p, id)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("case %d = %v, want %v", i, got, tc.want)
		}
	}
}
