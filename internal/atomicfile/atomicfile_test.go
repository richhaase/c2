package atomicfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteCreatesAndReplacesAtomically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	if err := Write(path, []byte("first\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Write(path, []byte("second\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "second\n" {
		t.Fatalf("data = %q", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("mode = %o", got)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".data.json.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files remain: %v", matches)
	}
}

func TestWriteFailureLeavesExistingFileUntouched(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	if err := os.WriteFile(path, []byte("original\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	missingPath := filepath.Join(dir, "missing", "data.json")
	if err := Write(missingPath, []byte("replacement\n"), 0o600); err == nil {
		t.Fatal("Write succeeded")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original\n" {
		t.Fatalf("data = %q", data)
	}
}
