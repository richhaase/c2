package paths

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type DataPaths struct {
	Root       string
	Meta       string
	Workouts   string
	StrokesDir string
	NotesDir   string
	ArchiveDir string
	Plan       string
	Playbook   string
	ReportsDir string
}

func (p DataPaths) StrokeFile(id int64) string {
	return filepath.Join(p.StrokesDir, strconv.FormatInt(id, 10)+".jsonl")
}

func (p DataPaths) ArchiveFile(year int) string {
	return filepath.Join(p.ArchiveDir, strconv.Itoa(year)+".jsonl")
}

func (p DataPaths) NarrativeFile(date string) string {
	return filepath.Join(p.ReportsDir, date+".md")
}

func ExpandTilde(p string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}

func For(root string) DataPaths {
	abs, err := filepath.Abs(ExpandTilde(root))
	if err != nil {
		abs = ExpandTilde(root)
	}
	return DataPaths{
		Root:       abs,
		Meta:       filepath.Join(abs, "meta.json"),
		Workouts:   filepath.Join(abs, "workouts.jsonl"),
		StrokesDir: filepath.Join(abs, "strokes"),
		NotesDir:   filepath.Join(abs, "notes"),
		ArchiveDir: filepath.Join(abs, "notes", "archive"),
		Plan:       filepath.Join(abs, "plan.md"),
		Playbook:   filepath.Join(abs, "playbook.md"),
		ReportsDir: filepath.Join(abs, "reports"),
	}
}

func CanonicalRoot(p string) string {
	expanded := ExpandTilde(p)
	base, err := filepath.Abs(expanded)
	if err != nil {
		return expanded
	}
	var rest []string
	for {
		real, err := filepath.EvalSymlinks(base)
		if err == nil {
			if len(rest) == 0 {
				return real
			}
			return filepath.Join(append([]string{real}, rest...)...)
		}
		if !os.IsNotExist(err) {
			if len(rest) == 0 {
				return base
			}
			return filepath.Join(append([]string{base}, rest...)...)
		}
		parent := filepath.Dir(base)
		if parent == base {
			abs, absErr := filepath.Abs(expanded)
			if absErr != nil {
				return expanded
			}
			return abs
		}
		rest = append([]string{filepath.Base(base)}, rest...)
		base = parent
	}
}
