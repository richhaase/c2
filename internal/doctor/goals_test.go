package doctor

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/richhaase/c2/internal/goals"
	"github.com/richhaase/c2/internal/notes"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/store"
)

func TestDoctorDetectsDivergenceAcrossLooseAndArchivedNotes(t *testing.T) {
	p := paths.For(t.TempDir())
	now := time.Now()
	if err := store.Init(p, now, nil); err != nil {
		t.Fatal(err)
	}
	n := notes.Record{ID: "01M3DH249VRJY9VAB7DZPMKYRS", Date: "2026-01-01T12:00:00Z", Type: "subjective", Author: "athlete", Body: "original"}
	body, err := notes.Serialize(n)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.ArchiveFile(2026), []byte(body+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	n.Body = "conflicting revision"
	if err := notes.Write(p, n); err != nil {
		t.Fatal(err)
	}
	r := Run(p)
	if !strings.Contains(strings.Join(r.Issues, "\n"), "divergent loose and archived") {
		t.Fatalf("not detected: %+v", r)
	}
	if err := os.WriteFile(goals.File(p), []byte(`{"version":99}`), 0o644); err != nil {
		t.Fatal(err)
	}
	r = Run(p)
	if !strings.Contains(strings.Join(r.Issues, "\n"), "Unsupported goals.json version") {
		t.Fatal("invalid goals not detected")
	}
}
