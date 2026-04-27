package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/richhaase/c2/internal/model"
)

func TestWriteAndReadStrokeData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "strokes", "101.jsonl")
	strokes := []model.StrokeData{
		{T: floatPtr(1.0), D: floatPtr(5.2), P: floatPtr(120.5), SPM: floatPtr(27), HR: floatPtr(145)},
		{T: floatPtr(2.0), D: floatPtr(10.4), P: floatPtr(121.5), SPM: floatPtr(28), HR: floatPtr(146)},
	}

	if err := writeStrokeDataPath(path, strokes); err != nil {
		t.Fatal(err)
	}

	got, err := readStrokeDataPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(strokes) {
		t.Fatalf("readStrokeDataPath() returned %d strokes, want %d", len(got), len(strokes))
	}
	for i := range strokes {
		if *got[i].T != *strokes[i].T || *got[i].D != *strokes[i].D || *got[i].P != *strokes[i].P || *got[i].SPM != *strokes[i].SPM || *got[i].HR != *strokes[i].HR {
			t.Fatalf("stroke %d = %#v, want %#v", i, got[i], strokes[i])
		}
	}
}

func TestReadStrokeDataMissingFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "strokes", "missing.jsonl")

	strokes, err := readStrokeDataPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(strokes) != 0 {
		t.Fatalf("readStrokeDataPath() returned %d strokes, want 0", len(strokes))
	}
}

func TestReadStrokeDataInvalidJSONIncludesPathAndLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "strokes", "101.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{\"t\":1}\nnot-json\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := readStrokeDataPath(path)
	if err == nil {
		t.Fatal("readStrokeDataPath() returned nil error")
	}
	message := err.Error()
	if !strings.Contains(message, path) || !strings.Contains(message, "line 2") {
		t.Fatalf("readStrokeDataPath() error = %q, want path and line number", message)
	}
}

func TestWriteStrokeDataCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "strokes", "101.jsonl")

	if err := writeStrokeDataPath(path, []model.StrokeData{{T: floatPtr(1)}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestHasStrokeData(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	strokesDir := filepath.Join(dir, ".config", "c2", "data", "strokes")
	if err := os.MkdirAll(strokesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	has, err := HasStrokeData(101)
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Fatal("HasStrokeData() missing file = true, want false")
	}

	if err := os.WriteFile(filepath.Join(strokesDir, "101.jsonl"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	has, err = HasStrokeData(101)
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("HasStrokeData() existing file = false, want true")
	}

	if err := os.Mkdir(filepath.Join(strokesDir, "102.jsonl"), 0o755); err != nil {
		t.Fatal(err)
	}
	has, err = HasStrokeData(102)
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Fatal("HasStrokeData() directory = true, want false")
	}
}

func floatPtr(v float64) *float64 {
	return &v
}
