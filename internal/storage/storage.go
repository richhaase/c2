package storage

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"syscall"

	"github.com/richhaase/c2/internal/jsonx"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/paths"
)

const SchemaVersion = 1

type StoreMeta struct {
	SchemaVersion *int   `json:"schema_version"`
	Created       string `json:"created"`
	LastSync      string `json:"last_sync,omitempty"`
}

func IsMissing(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR)
}

func nonEmptyLines(data []byte) [][]byte {
	var out [][]byte
	for line := range bytes.SplitSeq(data, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		out = append(out, line)
	}
	return out
}

func ReadWorkouts(p paths.DataPaths) ([]models.Workout, error) {
	data, err := os.ReadFile(p.Workouts)
	if err != nil {
		if IsMissing(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := nonEmptyLines(data)
	workouts := make([]models.Workout, 0, len(lines))
	for i, line := range lines {
		var w models.Workout
		if err := json.Unmarshal(line, &w); err != nil {
			return nil, fmt.Errorf("workouts.jsonl line %d: %w", i+1, err)
		}
		workouts = append(workouts, w)
	}
	return workouts, nil
}

func AppendWorkouts(p paths.DataPaths, incoming []models.Workout) (int, error) {
	existing, err := ReadWorkouts(p)
	if err != nil {
		return 0, err
	}
	seen := make(map[int64]bool, len(existing))
	for _, w := range existing {
		seen[w.ID] = true
	}

	var buf bytes.Buffer
	written := 0
	for _, w := range incoming {
		if seen[w.ID] {
			continue
		}
		seen[w.ID] = true
		line, err := jsonx.Compact(w)
		if err != nil {
			return 0, err
		}
		buf.Write(line)
		buf.WriteByte('\n')
		written++
	}
	if written == 0 {
		return 0, nil
	}

	f, err := os.OpenFile(p.Workouts, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	if _, err := f.Write(buf.Bytes()); err != nil {
		_ = f.Close()
		return 0, err
	}
	if err := f.Close(); err != nil {
		return 0, err
	}
	return written, nil
}

func WorkoutCount(p paths.DataPaths) (int, error) {
	f, err := os.Open(p.Workouts)
	if err != nil {
		if IsMissing(err) {
			return 0, nil
		}
		return 0, err
	}
	defer func() { _ = f.Close() }()
	count := 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			count++
		}
	}
	return count, scanner.Err()
}

func HasStrokeData(p paths.DataPaths, workoutID int64) (bool, error) {
	_, err := os.Stat(p.StrokeFile(workoutID))
	if err != nil {
		if IsMissing(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func WriteStrokeData(p paths.DataPaths, workoutID int64, strokes []models.StrokeData) error {
	var buf bytes.Buffer
	for _, s := range strokes {
		line, err := jsonx.Compact(s)
		if err != nil {
			return err
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	return os.WriteFile(p.StrokeFile(workoutID), buf.Bytes(), 0o644)
}

func ReadStrokeData(p paths.DataPaths, workoutID int64) ([]models.StrokeData, error) {
	data, err := os.ReadFile(p.StrokeFile(workoutID))
	if err != nil {
		if IsMissing(err) {
			return nil, nil
		}
		return nil, err
	}
	var strokes []models.StrokeData
	for _, line := range nonEmptyLines(data) {
		var probe map[string]any
		if err := json.Unmarshal(line, &probe); err != nil || probe == nil {
			continue
		}
		var s models.StrokeData
		if err := json.Unmarshal(line, &s); err != nil {
			continue
		}
		strokes = append(strokes, s)
	}
	return strokes, nil
}

func ReadMeta(p paths.DataPaths, warn func(string)) *StoreMeta {
	data, err := os.ReadFile(p.Meta)
	if err != nil {
		if !IsMissing(err) && warn != nil {
			warn(fmt.Sprintf("Warning: %s is unreadable and will be ignored.", p.Meta))
		}
		return nil
	}
	var meta StoreMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		if warn != nil {
			warn(fmt.Sprintf("Warning: %s is corrupt and will be ignored.", p.Meta))
		}
		return nil
	}
	return &meta
}

func WriteMeta(p paths.DataPaths, meta StoreMeta) error {
	data, err := jsonx.Indent(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(p.Meta, append(data, '\n'), 0o644)
}
