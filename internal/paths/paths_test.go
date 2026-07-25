package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandTildeAndFor(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got := ExpandTilde("~"); got != home {
		t.Fatalf("ExpandTilde(~) = %q", got)
	}
	if got := ExpandTilde("~/rowing"); got != filepath.Join(home, "rowing") {
		t.Fatalf("ExpandTilde = %q", got)
	}
	if got := ExpandTilde("~someone/rowing"); got != "~someone/rowing" {
		t.Fatalf("ExpandTilde changed %q", got)
	}
	p := For("~/rowing")
	if p.Root != filepath.Join(home, "rowing") {
		t.Fatalf("Root = %q", p.Root)
	}
	if p.Workouts != filepath.Join(p.Root, "workouts.jsonl") ||
		p.StrokeFile(42) != filepath.Join(p.Root, "strokes", "42.jsonl") ||
		p.ArchiveFile(2026) != filepath.Join(p.Root, "notes", "archive", "2026.jsonl") ||
		p.NarrativeFile("2026-07-20") != filepath.Join(p.Root, "reports", "2026-07-20.md") {
		t.Fatalf("paths = %#v", p)
	}
}

func TestCanonicalRootResolvesExistingSymlinksAndMissingChildren(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	got := CanonicalRoot(filepath.Join(link, "missing", "child"))
	resolved, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(resolved, "missing", "child")
	if got != want {
		t.Fatalf("CanonicalRoot = %q, want %q", got, want)
	}
}
