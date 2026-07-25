package store

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/notes"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/storage"
)

type DirState string

const (
	StateMissing DirState = "missing"
	StateStore   DirState = "store"
	StateEmpty   DirState = "empty"
	StateForeign DirState = "foreign"
)

type Inspection struct {
	Path     string
	State    DirState
	Writable bool
}

var storeMarkers = []string{
	"workouts.jsonl",
	"strokes",
	"notes",
	"reports",
	"plan.md",
	"playbook.md",
}

type Summary struct {
	Workouts      int
	FirstDate     string
	LastDate      string
	StrokeFiles   int
	Notes         int
	SchemaVersion *int
	LastSync      string
}

func Inspect(p paths.DataPaths, warn func(string)) (Inspection, error) {
	info, err := os.Stat(p.Root)
	if err != nil {
		if isNotDir(err) {
			return Inspection{Path: p.Root, State: StateForeign, Writable: false}, nil
		}
		if !os.IsNotExist(err) {
			return Inspection{}, err
		}
		return Inspection{Path: p.Root, State: StateMissing, Writable: parentWritable(p.Root)}, nil
	}
	if !info.IsDir() {
		return Inspection{Path: p.Root, State: StateForeign, Writable: false}, nil
	}

	meta := storage.ReadMeta(p, warn)
	entries, err := visibleEntries(p.Root)
	if err != nil {
		return Inspection{}, err
	}

	var state DirState
	switch {
	case meta != nil && meta.SchemaVersion != nil:
		state = StateStore
	case len(entries) == 0:
		state = StateEmpty
	default:
		strong := slices.Contains(entries, "workouts.jsonl") || slices.Contains(entries, "strokes")
		known := true
		for _, e := range entries {
			if e != "meta.json" && !slices.Contains(storeMarkers, e) {
				known = false
				break
			}
		}
		if strong && known {
			state = StateStore
		} else {
			state = StateForeign
		}
	}
	return Inspection{Path: p.Root, State: state, Writable: probeWritable(p.Root)}, nil
}

func isNotDir(err error) bool {
	return errors.Is(err, syscall.ENOTDIR)
}

func visibleEntries(dir string) ([]string, error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, item := range items {
		if strings.HasPrefix(item.Name(), ".") {
			continue
		}
		out = append(out, item.Name())
	}
	return out, nil
}

func parentWritable(path string) bool {
	current := path
	for {
		parent := filepath.Dir(current)
		if parent == current {
			return false
		}
		if _, err := os.Stat(parent); err == nil {
			return probeWritable(parent)
		} else if !os.IsNotExist(err) {
			return false
		}
		current = parent
	}
}

func probeWritable(dir string) bool {
	probe := filepath.Join(dir, fmt.Sprintf(".c2-probe-%d", os.Getpid()))
	if err := os.WriteFile(probe, nil, 0o644); err != nil {
		return false
	}
	os.Remove(probe)
	return true
}

func RejectForeign(p paths.DataPaths, warn func(string)) error {
	inspection, err := Inspect(p, warn)
	if err != nil {
		return err
	}
	if inspection.State == StateForeign {
		return fmt.Errorf("%s exists but is not a c2 data store. Fix data_dir via `c2 setup`.", p.Root)
	}
	return nil
}

func EnsureForWrite(p paths.DataPaths, now time.Time, warn func(string)) error {
	inspection, err := Inspect(p, warn)
	if err != nil {
		return err
	}
	if inspection.State == StateForeign {
		return fmt.Errorf("%s exists but is not a c2 data store. Fix data_dir via `c2 setup`.", p.Root)
	}
	if !inspection.Writable {
		return fmt.Errorf("Cannot write to %s.", p.Root)
	}
	return Init(p, now, warn)
}

func Init(p paths.DataPaths, now time.Time, warn func(string)) error {
	for _, dir := range []string{p.StrokesDir, p.ArchiveDir, p.ReportsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if storage.ReadMeta(p, warn) == nil {
		return storage.WriteMeta(p, storage.StoreMeta{
			SchemaVersion: new(storage.SchemaVersion),
			Created:       now.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		})
	}
	return nil
}

func listIfPresent(dir string) []string {
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Name())
	}
	return out
}

func Summarize(p paths.DataPaths, warn func(string)) (Summary, error) {
	workouts, err := storage.ReadWorkouts(p)
	if err != nil {
		return Summary{}, err
	}
	days := make([]string, 0, len(workouts))
	for _, w := range workouts {
		days = append(days, models.CalendarDay(w))
	}
	sort.Strings(days)

	strokeFiles := 0
	for _, f := range listIfPresent(p.StrokesDir) {
		if strings.HasSuffix(f, ".jsonl") {
			strokeFiles++
		}
	}

	summary := Summary{
		Workouts:    len(workouts),
		StrokeFiles: strokeFiles,
		Notes:       len(notes.ReadAll(p)),
	}
	if len(days) > 0 {
		summary.FirstDate = days[0]
		summary.LastDate = days[len(days)-1]
	}
	if meta := storage.ReadMeta(p, warn); meta != nil {
		summary.SchemaVersion = meta.SchemaVersion
		summary.LastSync = meta.LastSync
	}
	return summary, nil
}

type CopyStats struct {
	Files int
	Bytes int64
}

func Move(from, to paths.DataPaths, warn func(string)) (CopyStats, error) {
	target, err := Inspect(to, warn)
	if err != nil {
		return CopyStats{}, err
	}
	if target.State == StateStore || target.State == StateForeign {
		return CopyStats{}, fmt.Errorf("target %s is not empty", to.Root)
	}
	if !target.Writable {
		return CopyStats{}, fmt.Errorf("target %s is not writable", to.Root)
	}
	if err := os.MkdirAll(to.Root, 0o755); err != nil {
		return CopyStats{}, err
	}
	if err := copyTree(from.Root, to.Root); err != nil {
		return CopyStats{}, err
	}
	return verifyCopy(from.Root, to.Root)
}

func copyTree(src, dst string) error {
	items, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, item := range items {
		if strings.HasPrefix(item.Name(), ".") {
			continue
		}
		srcPath := filepath.Join(src, item.Name())
		dstPath := filepath.Join(dst, item.Name())
		if item.IsDir() {
			if err := os.MkdirAll(dstPath, 0o755); err != nil {
				return err
			}
			if err := copyTree(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func verifyCopy(fromDir, toDir string) (CopyStats, error) {
	var stats CopyStats
	items, err := os.ReadDir(fromDir)
	if err != nil {
		return stats, err
	}
	for _, item := range items {
		if strings.HasPrefix(item.Name(), ".") {
			continue
		}
		src := filepath.Join(fromDir, item.Name())
		dst := filepath.Join(toDir, item.Name())
		if item.IsDir() {
			sub, err := verifyCopy(src, dst)
			if err != nil {
				return stats, err
			}
			stats.Files += sub.Files
			stats.Bytes += sub.Bytes
			continue
		}
		srcInfo, err := os.Stat(src)
		if err != nil {
			return stats, err
		}
		dstInfo, err := os.Stat(dst)
		if err != nil {
			return stats, fmt.Errorf("copy verification failed: %s is missing", dst)
		}
		if dstInfo.Size() != srcInfo.Size() {
			return stats, fmt.Errorf("copy verification failed: %s has %d bytes, expected %d", dst, dstInfo.Size(), srcInfo.Size())
		}
		stats.Files++
		stats.Bytes += srcInfo.Size()
	}
	return stats, nil
}
