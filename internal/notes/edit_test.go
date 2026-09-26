package notes

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCorrectionPreservesIdentityMetadataAndSurvivesCompaction(t *testing.T) {
	p := tempStore(t)
	before := record("A", "2026-06-01T08:00:00-06:00", "original")
	before.Tags = []string{"technique"}
	before.WorkoutID = new(int64(7))
	mustWrite(t, p, before)
	after := before
	after.Body = "  correction < & >\n\n"
	if changed, err := Update(p, before, after); err != nil || !changed {
		t.Fatalf("update = %v, %v", changed, err)
	}
	if got := readAll(t, p); len(got) != 1 || !reflect.DeepEqual(got[0], after) {
		t.Fatalf("saved = %+v", got)
	}
	if _, err := Compact(p, testNow); err != nil {
		t.Fatal(err)
	}
	if got := readAll(t, p); len(got) != 1 || !reflect.DeepEqual(got[0], after) {
		t.Fatalf("archived = %+v", got)
	}
}

func TestArchivedCorrectionKeepsOtherRecordsAndReordersDates(t *testing.T) {
	p := tempStore(t)
	before := record("A", "2026-06-01T08:00:00Z", "original")
	other := record("B", "2026-06-02T08:00:00Z", "untouched")
	raw := strings.TrimSuffix(serialize(t, other), "}") + `,"future_field":"keep me"}`
	path := p.ArchiveFile(2026)
	if err := os.WriteFile(path, []byte(serialize(t, before)+"\n"+raw+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	after := before
	after.Body, after.Date = "corrected", "2027-01-01T08:00:00Z"
	if changed, err := Update(p, before, after); err != nil || !changed {
		t.Fatalf("update = %v, %v", changed, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != raw+"\n"+serialize(t, after)+"\n" || len(looseFiles(t, p)) != 0 {
		t.Fatalf("archive = %s, loose = %v", data, looseFiles(t, p))
	}
	if changed, err := Update(p, after, after); err != nil || changed {
		t.Fatalf("no-op = %v, %v", changed, err)
	}
	unchanged, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, unchanged) {
		t.Fatalf("no-op rewrote archive: %v", err)
	}
}

func TestCorrectionRefusesStaleRecordAndInvalidChanges(t *testing.T) {
	p := tempStore(t)
	before := record("A", "2026-06-01T08:00:00Z", "original")
	mustWrite(t, p, before)
	after := before
	after.Body = "first correction"
	if _, err := Update(p, before, after); err != nil {
		t.Fatal(err)
	}
	stale := before
	stale.Body = "stale correction"
	if _, err := Update(p, before, stale); err == nil {
		t.Fatal("stale correction accepted")
	}
	for _, mutate := range []func(*Record){
		func(n *Record) { n.ID = "different" },
		func(n *Record) { n.Body = " \n" },
		func(n *Record) { n.Type = "invalid" },
		func(n *Record) { n.Date = "2026-02-30" },
	} {
		bad := after
		mutate(&bad)
		if _, err := Update(p, after, bad); err == nil {
			t.Fatalf("invalid correction accepted: %+v", bad)
		}
	}
	if got := readAll(t, p); !reflect.DeepEqual(got, []Record{after}) {
		t.Fatalf("failed edits changed store: %+v", got)
	}
}

func TestCorrectionRefusesConflictsAndMalformedStorage(t *testing.T) {
	for _, scenario := range []string{"duplicate archive", "duplicate loose", "corrupt loose", "corrupt archive", "unknown target field"} {
		t.Run(scenario, func(t *testing.T) {
			p := tempStore(t)
			before := record("A", "2026-06-01T08:00:00Z", "original")
			mustWrite(t, p, before)
			path := filepath.Join(p.NotesDir, "A.json")
			extraPath, extra := "", ""
			switch scenario {
			case "duplicate archive":
				extraPath, extra = p.ArchiveFile(2026), serialize(t, before)+"\n"
			case "duplicate loose":
				extraPath, extra = filepath.Join(p.NotesDir, "conflict.json"), serialize(t, before)
			case "corrupt loose":
				extraPath, extra = filepath.Join(p.NotesDir, "corrupt.json"), "{broken"
			case "corrupt archive":
				extraPath, extra = p.ArchiveFile(2026), "{broken\n"
			case "unknown target field":
				extraPath, extra = path, strings.TrimSuffix(serialize(t, before), "}")+`,"future":true}`
			}
			if err := os.WriteFile(extraPath, []byte(extra), 0o644); err != nil {
				t.Fatal(err)
			}
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ReadForEdit(p, before.ID); err == nil {
				t.Fatal("unsafe source opened for editing")
			}
			after := before
			after.Body = "corrected"
			if _, err := Update(p, before, after); err == nil {
				t.Fatal("unsafe source overwritten")
			}
			got, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(got, original) {
				t.Fatalf("changed on failure: %s, %v", got, err)
			}
		})
	}
}
