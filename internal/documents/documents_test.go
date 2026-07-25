package documents

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/richhaase/c2/internal/paths"
)

func TestReadDistinguishesMissingAndPresentDocuments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan.md")
	content, ok, err := Read(path)
	if err != nil || ok || content != "" {
		t.Fatalf("missing = %q, %v, %v", content, ok, err)
	}
	if err := os.WriteFile(path, []byte("# Plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	content, ok, err = Read(path)
	if err != nil || !ok || content != "# Plan\n" {
		t.Fatalf("present = %q, %v, %v", content, ok, err)
	}
}

func TestListNarrativesFiltersValidDatesAndSorts(t *testing.T) {
	p := paths.For(t.TempDir())
	if err := os.MkdirAll(p.ReportsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"2026-07-20.md", "2026-01-02.md", "bad.md", "2026-02-30.md", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(p.ReportsDir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dates, err := ListNarratives(p)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"2026-01-02", "2026-07-20"}; !slices.Equal(dates, want) {
		t.Fatalf("dates = %v, want %v", dates, want)
	}
}

func TestListNarrativesReturnsOperationalErrors(t *testing.T) {
	p := paths.For(t.TempDir())
	if err := os.WriteFile(p.ReportsDir, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ListNarratives(p); err == nil {
		t.Fatal("ListNarratives succeeded")
	}
}
