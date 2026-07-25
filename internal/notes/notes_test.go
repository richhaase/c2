package notes

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/richhaase/c2/internal/paths"
)

var testNow = time.Date(2026, 7, 6, 12, 0, 0, 0, time.Local)

func tempStore(t *testing.T) paths.DataPaths {
	t.Helper()
	p := paths.For(filepath.Join(t.TempDir(), "store"))
	if err := os.MkdirAll(p.ArchiveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func record(id, date, body string, noteType ...string) Record {
	kind := "observation"
	if len(noteType) > 0 {
		kind = noteType[0]
	}
	return Record{ID: id, Date: date, Type: kind, Body: body, Author: "athlete"}
}

func ids(records []Record) []string {
	out := make([]string, 0, len(records))
	for _, n := range records {
		out = append(out, n.ID)
	}
	return out
}

func readAll(t *testing.T, p paths.DataPaths) []Record {
	t.Helper()
	records, err := ReadAll(p)
	if err != nil {
		t.Fatal(err)
	}
	return records
}

func serialize(t *testing.T, n Record) string {
	t.Helper()
	out, err := Serialize(n)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestULIDIsTwentySixCharsAndTimeOrdered(t *testing.T) {
	a, err := ULID(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	b, err := ULID(time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 26 || len(b) != 26 {
		t.Fatalf("lengths %d %d", len(a), len(b))
	}
	if a >= b {
		t.Fatalf("expected %q < %q", a, b)
	}
}

func TestULIDRejectsDatesOutsideItsTimestampRange(t *testing.T) {
	if _, err := ULID(time.UnixMilli(-1)); err == nil {
		t.Fatal("ULID succeeded")
	}
}

func TestLocalISORendersLocalDayWithOffset(t *testing.T) {
	iso := LocalISO(time.Date(2026, 7, 6, 21, 30, 0, 0, time.Local))
	if !strings.HasPrefix(iso, "2026-07-06T21:30:00") {
		t.Fatalf("got %q", iso)
	}
	if !regexp.MustCompile(`[+-]\d{2}:\d{2}$`).MatchString(iso) {
		t.Fatalf("missing offset: %q", iso)
	}
}

func TestNotesSortByInstantAcrossMixedOffsets(t *testing.T) {
	p := tempStore(t)
	mustWrite(t, p, record("MINUS6", "2026-07-05T23:30:00-06:00", "later instant"))
	mustWrite(t, p, record("PLUS2", "2026-07-06T01:00:00+02:00", "earlier instant"))
	if got := ids(readAll(t, p)); !slices.Equal(got, []string{"PLUS2", "MINUS6"}) {
		t.Fatalf("got %v", got)
	}
}

func TestNotesRoundTripAndSortByDateThenID(t *testing.T) {
	p := tempStore(t)
	mustWrite(t, p, record("B", "2026-07-05T10:00:00-06:00", "second"))
	mustWrite(t, p, record("A", "2026-07-05T10:00:00-06:00", "first"))
	mustWrite(t, p, record("C", "2026-07-01T08:00:00-06:00", "oldest"))
	if got := ids(readAll(t, p)); !slices.Equal(got, []string{"C", "A", "B"}) {
		t.Fatalf("got %v", got)
	}
}

func TestCorruptLooseNotesAreSkipped(t *testing.T) {
	p := tempStore(t)
	mustWrite(t, p, record("GOOD", "2026-07-05T10:00:00-06:00", "fine"))
	writeRaw(t, p, "BAD.json", "{ nope")
	writeRaw(t, p, "SHAPE.json", `{"id":"SHAPE"}`)
	if got := ids(readAll(t, p)); !slices.Equal(got, []string{"GOOD"}) {
		t.Fatalf("got %v", got)
	}
}

func TestReadAllReturnsOperationalErrors(t *testing.T) {
	p := paths.For(t.TempDir())
	if err := os.MkdirAll(p.NotesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.NotesDir, "blocked.json"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(p.NotesDir, "directory.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadAll(p); err == nil {
		t.Fatal("ReadAll succeeded")
	}
}

func TestInvalidAuthorsTagsAndWorkoutIDsAreSkipped(t *testing.T) {
	p := tempStore(t)
	mustWrite(t, p, record("GOOD", "2026-07-05T10:00:00-06:00", "fine"))
	writeRaw(t, p, "LLM.json", `{"id":"LLM","date":"2026-07-05T10:00:00-06:00","type":"observation","body":"x","author":"llm"}`)
	writeRaw(t, p, "BADTAGS.json", `{"id":"BADTAGS","date":"2026-07-05T10:00:00-06:00","type":"observation","tags":"not-an-array","body":"x","author":"athlete"}`)
	writeRaw(t, p, "BADWID.json", `{"id":"BADWID","date":"2026-07-05T10:00:00-06:00","type":"observation","workout_id":"seven","body":"x","author":"athlete"}`)
	if got := ids(readAll(t, p)); !slices.Equal(got, []string{"GOOD"}) {
		t.Fatalf("got %v", got)
	}
}

func TestApplyFiltersByTypeSinceAndWorkout(t *testing.T) {
	withWorkout := record("A", "2026-07-01T08:00:00-06:00", "x", "subjective")
	withWorkout.WorkoutID = new(int64(7))
	records := []Record{
		withWorkout,
		record("B", "2026-07-03T08:00:00-06:00", "y", "lesson"),
		record("C", "2026-07-05T08:00:00-06:00", "z", "observation"),
	}
	if got := ids(Apply(records, Filter{Type: "lesson"})); !slices.Equal(got, []string{"B"}) {
		t.Fatalf("type: got %v", got)
	}
	if got := ids(Apply(records, Filter{Since: "2026-07-03"})); !slices.Equal(got, []string{"B", "C"}) {
		t.Fatalf("since: got %v", got)
	}
	if got := ids(Apply(records, Filter{WorkoutID: new(int64(7))})); !slices.Equal(got, []string{"A"}) {
		t.Fatalf("workout: got %v", got)
	}
}

func TestCompactionArchivesByYearAndKeepsHotSet(t *testing.T) {
	p := tempStore(t)
	mustWrite(t, p, record("OLD1", "2026-06-20T08:00:00-06:00", "old june"))
	mustWrite(t, p, record("OLD2", "2025-12-30T08:00:00-07:00", "old last year"))
	mustWrite(t, p, record("NEW1", "2026-07-05T08:00:00-06:00", "recent"))

	result, err := Compact(p, testNow)
	if err != nil {
		t.Fatal(err)
	}
	if result.Archived != 2 {
		t.Fatalf("archived %d", result.Archived)
	}
	if len(result.Years) != 2 || result.Years[0] != 2025 || result.Years[1] != 2026 {
		t.Fatalf("years %v", result.Years)
	}

	loose := looseFiles(t, p)
	if !slices.Equal(loose, []string{"NEW1.json"}) {
		t.Fatalf("loose %v", loose)
	}
	if got := ids(readAll(t, p)); !slices.Equal(got, []string{"OLD2", "OLD1", "NEW1"}) {
		t.Fatalf("all %v", got)
	}

	again, err := Compact(p, testNow)
	if err != nil {
		t.Fatal(err)
	}
	if again.Archived != 0 {
		t.Fatalf("second compaction archived %d", again.Archived)
	}
}

func TestCompactionIsDeterministicAndMergeIdempotent(t *testing.T) {
	storeA := tempStore(t)
	storeB := tempStore(t)
	batch := []Record{
		record("01A", "2026-06-01T08:00:00-06:00", "one"),
		record("01B", "2026-06-15T08:00:00-06:00", "two"),
		record("01C", "2026-05-20T08:00:00-06:00", "three"),
	}
	for _, store := range []paths.DataPaths{storeA, storeB} {
		for i := len(batch) - 1; i >= 0; i-- {
			mustWrite(t, store, batch[i])
		}
	}
	if _, err := Compact(storeA, testNow); err != nil {
		t.Fatal(err)
	}
	if _, err := Compact(storeB, testNow); err != nil {
		t.Fatal(err)
	}
	a, err := os.ReadFile(storeA.ArchiveFile(2026))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(storeB.ArchiveFile(2026))
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatalf("archives differ:\nA %s\nB %s", a, b)
	}

	mustWrite(t, storeA, record("01D", "2026-06-20T08:00:00-06:00", "late arrival"))
	if _, err := Compact(storeA, testNow); err != nil {
		t.Fatal(err)
	}
	june := 0
	for _, n := range readAll(t, storeA) {
		if strings.HasPrefix(n.Date, "2026-06") {
			june++
		}
	}
	if june != 3 {
		t.Fatalf("june notes = %d", june)
	}
}

func TestSerializeOmitsEmptyOptionalFieldsAndKeepsOrder(t *testing.T) {
	n := record("01A", "2026-07-05T10:00:00-06:00", "plain")
	if got := serialize(t, n); got != `{"id":"01A","date":"2026-07-05T10:00:00-06:00","type":"observation","body":"plain","author":"athlete"}` {
		t.Fatalf("got %s", got)
	}
	n.WorkoutID = new(int64(7))
	n.Tags = []string{"a", "b"}
	if got := serialize(t, n); got != `{"id":"01A","date":"2026-07-05T10:00:00-06:00","type":"observation","workout_id":7,"tags":["a","b"],"body":"plain","author":"athlete"}` {
		t.Fatalf("got %s", got)
	}
}

func TestSerializeDoesNotEscapeHTML(t *testing.T) {
	n := record("01A", "2026-07-05T10:00:00-06:00", "held pace < 2:00 & HR > 150")
	serialized := serialize(t, n)
	if !strings.Contains(serialized, "pace < 2:00 & HR > 150") {
		t.Fatalf("body was escaped: %s", serialized)
	}
}

func mustWrite(t *testing.T, p paths.DataPaths, n Record) {
	t.Helper()
	if err := Write(p, n); err != nil {
		t.Fatal(err)
	}
}

func writeRaw(t *testing.T, p paths.DataPaths, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(p.NotesDir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func looseFiles(t *testing.T, p paths.DataPaths) []string {
	t.Helper()
	entries, err := os.ReadDir(p.NotesDir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			out = append(out, e.Name())
		}
	}
	return out
}
