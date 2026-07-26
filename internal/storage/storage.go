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

	"github.com/richhaase/c2/internal/atomicfile"
	"github.com/richhaase/c2/internal/jsonx"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/paths"
)

const SchemaVersion = 1

type StoreMeta struct {
	SchemaVersion *int   `json:"schema_version"`
	Created       string `json:"created"`
	LastSync      string `json:"last_sync,omitempty"`
	StrokeCursor  int64  `json:"stroke_cursor,omitempty"`
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

type UpsertResult struct {
	Added   int
	Updated int
}

func UpsertWorkouts(p paths.DataPaths, incoming []models.Workout) (UpsertResult, error) {
	existing, err := ReadWorkouts(p)
	if err != nil {
		return UpsertResult{}, err
	}
	originalLen := len(existing)
	indexByID := make(map[int64]int, len(existing)+len(incoming))
	encoded := make([][]byte, len(existing))
	for i, w := range existing {
		indexByID[w.ID] = i
		line, err := jsonx.Compact(w)
		if err != nil {
			return UpsertResult{}, err
		}
		encoded[i] = line
	}

	addedIDs := make(map[int64]bool)
	updatedIDs := make(map[int64]bool)
	for _, w := range incoming {
		line, err := jsonx.Compact(w)
		if err != nil {
			return UpsertResult{}, err
		}
		if index, ok := indexByID[w.ID]; ok {
			if bytes.Equal(encoded[index], line) {
				continue
			}
			existing[index] = w
			encoded[index] = line
			if !addedIDs[w.ID] {
				updatedIDs[w.ID] = true
			}
			continue
		}
		indexByID[w.ID] = len(existing)
		existing = append(existing, w)
		encoded = append(encoded, line)
		addedIDs[w.ID] = true
	}

	result := UpsertResult{Added: len(addedIDs), Updated: len(updatedIDs)}
	if result.Added == 0 && result.Updated == 0 {
		return result, nil
	}
	if result.Updated == 0 {
		var buf bytes.Buffer
		for _, line := range encoded[originalLen:] {
			buf.Write(line)
			buf.WriteByte('\n')
		}
		file, err := os.OpenFile(p.Workouts, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return UpsertResult{}, err
		}
		if _, err := file.Write(buf.Bytes()); err != nil {
			_ = file.Close()
			return UpsertResult{}, err
		}
		if err := file.Close(); err != nil {
			return UpsertResult{}, err
		}
		return result, nil
	}

	var buf bytes.Buffer
	for _, line := range encoded {
		buf.Write(line)
		buf.WriteByte('\n')
	}
	if err := atomicfile.Write(p.Workouts, buf.Bytes(), 0o644); err != nil {
		return UpsertResult{}, err
	}
	return result, nil
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
	data, err := os.ReadFile(p.StrokeFile(workoutID))
	if err != nil {
		if IsMissing(err) {
			return false, nil
		}
		return false, err
	}
	found := false
	for _, line := range nonEmptyLines(data) {
		if _, ok := parseStroke(line); !ok {
			return false, nil
		}
		found = true
	}
	return found, nil
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
	return atomicfile.Write(p.StrokeFile(workoutID), buf.Bytes(), 0o644)
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
		s, ok := parseStroke(line)
		if !ok {
			continue
		}
		strokes = append(strokes, s)
	}
	return strokes, nil
}

func parseStroke(line []byte) (models.StrokeData, bool) {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) < 2 || trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' {
		return models.StrokeData{}, false
	}
	var stroke models.StrokeData
	if err := json.Unmarshal(trimmed, &stroke); err != nil {
		return models.StrokeData{}, false
	}
	return stroke, true
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
	return atomicfile.Write(p.Meta, append(data, '\n'), 0o644)
}
