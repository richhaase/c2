package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/storage"
)

var testNow = time.Date(2026, 7, 5, 12, 0, 0, 0, time.Local)

const workoutLine = `{"id":1,"user_id":1,"date":"2026-07-01 08:00:00","distance":8000,"type":"rower","time":12000,"time_formatted":"20:00.0"}`

func inspect(t *testing.T, p paths.DataPaths) Inspection {
	t.Helper()
	got, err := Inspect(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestInspectMissingDirectory(t *testing.T) {
	p := paths.For(filepath.Join(t.TempDir(), "nope"))
	got := inspect(t, p)
	if got.State != StateMissing || !got.Writable {
		t.Fatalf("got %+v", got)
	}
}

func TestInspectEmptyStoreAndForeign(t *testing.T) {
	base := t.TempDir()

	empty := paths.For(filepath.Join(base, "empty"))
	if err := os.MkdirAll(empty.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := inspect(t, empty); got.State != StateEmpty {
		t.Fatalf("empty: got %v", got.State)
	}

	initialized := paths.For(filepath.Join(base, "store"))
	if err := os.MkdirAll(initialized.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Init(initialized, testNow, nil); err != nil {
		t.Fatal(err)
	}
	if got := inspect(t, initialized); got.State != StateStore {
		t.Fatalf("store: got %v", got.State)
	}

	foreign := paths.For(filepath.Join(base, "foreign"))
	if err := os.MkdirAll(foreign.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(foreign.Root, "novel.docx"), []byte("chapter one"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := inspect(t, foreign); got.State != StateForeign {
		t.Fatalf("foreign: got %v", got.State)
	}
}

func TestInitCreatesDirectoriesAndMetaOnce(t *testing.T) {
	p := paths.For(filepath.Join(t.TempDir(), "init"))
	if err := os.MkdirAll(p.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Init(p, testNow, nil); err != nil {
		t.Fatal(err)
	}
	meta := storage.ReadMeta(p, nil)
	if meta == nil || meta.SchemaVersion == nil || *meta.SchemaVersion != 1 {
		t.Fatalf("got %+v", meta)
	}
	created := meta.Created

	if err := Init(p, time.Date(2027, 1, 1, 0, 0, 0, 0, time.Local), nil); err != nil {
		t.Fatal(err)
	}
	again := storage.ReadMeta(p, nil)
	if again.Created != created {
		t.Fatalf("created changed: %q -> %q", created, again.Created)
	}
}

func TestInitRejectsUnsupportedSchemaVersion(t *testing.T) {
	p := paths.For(t.TempDir())
	version := storage.SchemaVersion + 1
	if err := storage.WriteMeta(p, storage.StoreMeta{
		SchemaVersion: &version,
		Created:       "2026-01-01T00:00:00.000Z",
	}); err != nil {
		t.Fatal(err)
	}
	if err := Init(p, testNow, nil); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("Init error = %v", err)
	}
	if _, err := os.Stat(p.StrokesDir); !os.IsNotExist(err) {
		t.Fatalf("strokes stat error = %v", err)
	}
}

func TestSummarizeCountsContents(t *testing.T) {
	p := paths.For(filepath.Join(t.TempDir(), "sum"))
	if err := os.MkdirAll(p.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Init(p, testNow, nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Workouts, []byte(workoutLine+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.StrokeFile(1), []byte(`{"t":1}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	note := `{"id":"01A","date":"2026-07-01T08:00:00-06:00","type":"observation","body":"counted","author":"athlete"}`
	if err := os.WriteFile(filepath.Join(p.NotesDir, "01A.json"), []byte(note), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Summarize(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Workouts != 1 || got.FirstDate != "2026-07-01" || got.StrokeFiles != 1 || got.Notes != 1 {
		t.Fatalf("got %+v", got)
	}
	if got.SchemaVersion == nil || *got.SchemaVersion != 1 {
		t.Fatalf("schema version %+v", got.SchemaVersion)
	}
}

func TestSummarizePropagatesStrokeDirectoryErrors(t *testing.T) {
	p := paths.For(t.TempDir())
	if err := os.WriteFile(p.StrokesDir, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Summarize(p, nil); err == nil {
		t.Fatal("Summarize error = nil")
	}
}

func TestMissingNestedPathsAreCreatable(t *testing.T) {
	nested := paths.For(filepath.Join(t.TempDir(), "a", "b", "c"))
	got := inspect(t, nested)
	if got.State != StateMissing || !got.Writable {
		t.Fatalf("got %+v", got)
	}
	if err := Init(nested, testNow, nil); err != nil {
		t.Fatal(err)
	}
	if got := inspect(t, nested); got.State != StateStore {
		t.Fatalf("after init: got %v", got.State)
	}
}

func TestGenericFolderNamesAreNotAdopted(t *testing.T) {
	base := t.TempDir()

	notesOnly := paths.For(filepath.Join(base, "notes-only"))
	if err := os.MkdirAll(notesOnly.NotesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := inspect(t, notesOnly); got.State != StateForeign {
		t.Fatalf("notes only: got %v", got.State)
	}

	planOnly := paths.For(filepath.Join(base, "plan-only"))
	if err := os.MkdirAll(planOnly.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planOnly.Plan, []byte("# someone else's plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := inspect(t, planOnly); got.State != StateForeign {
		t.Fatalf("plan only: got %v", got.State)
	}
}

func TestForeignMetaJSONIsNotAdopted(t *testing.T) {
	p := paths.For(filepath.Join(t.TempDir(), "other-tool"))
	if err := os.MkdirAll(p.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Meta, []byte(`{"name":"some other tool"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := inspect(t, p); got.State != StateForeign {
		t.Fatalf("got %v", got.State)
	}
}

func TestStoreMissingMetaButCurrentLayoutIsRecognized(t *testing.T) {
	p := paths.For(filepath.Join(t.TempDir(), "meta-lost"))
	if err := os.MkdirAll(p.StrokesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(p.ReportsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Workouts, []byte(workoutLine+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Plan, []byte("# plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := inspect(t, p); got.State != StateStore {
		t.Fatalf("got %v", got.State)
	}
}

func TestFileInPathYieldsForeign(t *testing.T) {
	base := t.TempDir()
	filePath := filepath.Join(base, "regular-file")
	if err := os.WriteFile(filePath, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := paths.For(filepath.Join(filePath, "sub"))
	got := inspect(t, nested)
	if got.State != StateForeign || got.Writable {
		t.Fatalf("got %+v", got)
	}
}

func TestCorruptMetaIsToleratedNotFatal(t *testing.T) {
	p := paths.For(filepath.Join(t.TempDir(), "corrupt"))
	if err := os.MkdirAll(p.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Init(p, testNow, nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Meta, []byte("{ truncated"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := inspect(t, p); got.State != StateStore {
		t.Fatalf("corrupt meta must stay usable, got %v", got.State)
	}
	if got := storage.ReadMeta(p, nil); got != nil {
		t.Fatalf("got %+v", got)
	}
	summary, err := Summarize(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if summary.SchemaVersion != nil {
		t.Fatalf("schema version should be nil, got %d", *summary.SchemaVersion)
	}
}

func TestMoveCopiesVerifiesAndRefusesNonEmptyTargets(t *testing.T) {
	base := t.TempDir()
	from := paths.For(filepath.Join(base, "src"))
	if err := os.MkdirAll(from.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Init(from, testNow, nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(from.Workouts, []byte(workoutLine+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(from.StrokeFile(1), []byte(`{"t":1}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	to := paths.For(filepath.Join(base, "dst"))
	copied, err := Copy(from, to, nil)
	if err != nil {
		t.Fatal(err)
	}
	if copied.Files < 3 {
		t.Fatalf("copied %d files", copied.Files)
	}
	moved, err := storage.ReadWorkouts(to)
	if err != nil {
		t.Fatal(err)
	}
	if len(moved) != 1 {
		t.Fatalf("moved %d workouts", len(moved))
	}

	occupied := paths.For(filepath.Join(base, "occupied"))
	if err := os.MkdirAll(occupied.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(occupied.Root, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Copy(from, occupied, nil); err == nil {
		t.Fatal("expected refusal for non-empty target")
	}
}

func TestMoveNeverCopiesDotfilesAndPreservesTargetVCSMetadata(t *testing.T) {
	base := t.TempDir()
	from := paths.For(filepath.Join(base, "src"))
	if err := os.MkdirAll(from.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Init(from, testNow, nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(from.Workouts, []byte(workoutLine+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(from.Root, ".DS_Store"), []byte("source junk"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(from.Root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(from.Root, ".git", "HEAD"), []byte("ref: refs/heads/source\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	to := paths.For(filepath.Join(base, "synced"))
	if err := os.MkdirAll(filepath.Join(to.Root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(to.Root, ".git", "HEAD"), []byte("ref: refs/heads/target\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(to.Root, ".DS_Store"), []byte("different pre-existing junk!"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Copy(from, to, nil); err != nil {
		t.Fatal(err)
	}
	head, err := os.ReadFile(filepath.Join(to.Root, ".git", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	if string(head) != "ref: refs/heads/target\n" {
		t.Fatalf("target VCS metadata clobbered: %q", head)
	}
	ds, err := os.ReadFile(filepath.Join(to.Root, ".DS_Store"))
	if err != nil {
		t.Fatal(err)
	}
	if string(ds) != "different pre-existing junk!" {
		t.Fatalf("target dotfile clobbered: %q", ds)
	}
}

func TestMoveRejectsSymbolicLinks(t *testing.T) {
	root := t.TempDir()
	from := paths.For(filepath.Join(root, "from"))
	to := paths.For(filepath.Join(root, "to"))
	if err := Init(from, time.Now(), nil); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside.json")
	if err := os.WriteFile(outside, []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(from.Root, "linked.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := Copy(from, to, nil); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("Copy error = %v", err)
	}
}

func TestVerifyCopyRejectsSameSizeContentChanges(t *testing.T) {
	root := t.TempDir()
	from := filepath.Join(root, "from")
	to := filepath.Join(root, "to")
	if err := os.MkdirAll(from, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(to, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(from, "data"), []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(to, "data"), []byte("bravo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyCopy(from, to); err == nil || !strings.Contains(err.Error(), "content differs") {
		t.Fatalf("verifyCopy error = %v", err)
	}
}
