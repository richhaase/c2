package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/richhaase/c2/internal/notes"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/store"
)

func savedNote(t *testing.T, raw string) notes.Record {
	t.Helper()
	var env struct {
		Data notes.Record `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		t.Fatal(err)
	}
	return env.Data
}

func TestNoteCorrectionWorkflowAndTransfer(t *testing.T) {
	home := testHome(t)
	p := paths.For(seedWorkouts(t, home))
	created := savedNote(t, mustRun(t, "note", "add", "--body", "original", "--date", "2025-12-20", "--author", "coach", "--type", "lesson", "--tags", "pace,technique", "--workout", "last", "--json").stdout)
	if created.ID == "" || !strings.Contains(mustRun(t, "note", "list").stdout, created.ID) {
		t.Fatal("note ID is not discoverable")
	}
	text := "  revised < & >\n\n"
	got := runWithStdin(t, text, "note", "edit", created.ID, "--file", "-", "--json")
	if got.failed || !strings.Contains(got.stdout, `"changed": true`) {
		t.Fatalf("edit = %+v", got)
	}
	expected := created
	expected.Body = text
	if actual := savedNote(t, mustRun(t, "note", "show", created.ID, "--json").stdout); !reflect.DeepEqual(actual, expected) {
		t.Fatalf("saved = %+v, want %+v", actual, expected)
	}
	if got := mustRun(t, "note", "edit", created.ID, "--body", text, "--json"); !strings.Contains(got.stdout, `"changed": false`) {
		t.Fatal(got.stdout)
	}
	if _, err := notes.Compact(p, time.Now()); err != nil {
		t.Fatal(err)
	}
	mustRun(t, "note", "edit", created.ID, "--date", "2026-01-02", "--tags", "", "--workout", "")
	corrected := savedNote(t, mustRun(t, "note", "show", created.ID, "--json").stdout)
	if corrected.ID != created.ID || corrected.Body != text || corrected.WorkoutID != nil || len(corrected.Tags) != 0 || corrected.Author != "coach" || corrected.Type != "lesson" || !strings.HasPrefix(corrected.Date, "2026-01-02") {
		t.Fatalf("metadata correction = %+v", corrected)
	}
	if _, err := notes.Compact(p, time.Now()); err != nil {
		t.Fatal(err)
	}
	mustRun(t, "data", "doctor")
	mustRun(t, "data", "prepare", "--timezone", "UTC")
	target := paths.For(filepath.Join(t.TempDir(), "copied"))
	if _, err := store.Copy(p, target, nil); err != nil {
		t.Fatal(err)
	}
	testHome(t)
	mustRun(t, "data", "use", target.Root)
	if actual := savedNote(t, mustRun(t, "note", "show", created.ID, "--json").stdout); !reflect.DeepEqual(actual, corrected) {
		t.Fatalf("transferred = %+v", actual)
	}
	mustRun(t, "data", "doctor")
}

func TestNoteEditInputAndFailureLeaveRecordIntact(t *testing.T) {
	testHome(t)
	id := strings.TrimSpace(mustRun(t, "note", "add", "original").stdout)
	before := savedNote(t, mustRun(t, "note", "show", id, "--json").stdout)
	for _, flags := range [][]string{
		{}, {"--body", ""}, {"--file", ""}, {"--file", "missing-file"},
		{"--body", "x", "--file", "-"}, {"positional", "--body", "explicit"},
		{"--date", "2026-02-30"}, {"--type", "invalid"}, {"--author", "invalid"}, {"--workout", "999"},
	} {
		args := append([]string{"note", "edit", id}, flags...)
		if !run(t, args...).failed {
			t.Fatalf("accepted invalid edit %v", args)
		}
		if actual := savedNote(t, mustRun(t, "note", "show", id, "--json").stdout); !reflect.DeepEqual(actual, before) {
			t.Fatalf("failed edit changed record: %+v", actual)
		}
	}
	file := filepath.Join(t.TempDir(), "correction.md")
	if err := os.WriteFile(file, []byte("file correction\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mustRun(t, "note", "edit", id, "--file", file, "--type", "lesson", "--author", "coach")
	if n := savedNote(t, mustRun(t, "note", "show", id, "--json").stdout); n.Body != "file correction\n" || n.Type != "lesson" || n.Author != "coach" {
		t.Fatalf("file correction = %+v", n)
	}
}
