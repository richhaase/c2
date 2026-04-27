package storage

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/model"
)

func ReadWorkouts() ([]model.Workout, error) {
	return ReadWorkoutsPath(config.WorkoutsPath())
}

func ReadWorkoutsPath(path string) ([]model.Workout, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []model.Workout{}, nil
		}
		return nil, err
	}
	var workouts []model.Workout
	reader := bufio.NewReader(file)
	lineNumber := 0
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			lineNumber++
			workout, err := parseWorkoutLine(path, lineNumber, line)
			if err != nil {
				_ = file.Close()
				return nil, err
			}
			if workout != nil {
				workouts = append(workouts, *workout)
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			_ = file.Close()
			return nil, err
		}
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	return workouts, nil
}

func parseWorkoutLine(path string, lineNumber int, line []byte) (*model.Workout, error) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return nil, nil
	}
	var workout model.Workout
	if err := json.Unmarshal(line, &workout); err != nil {
		return nil, fmt.Errorf("%s line %d: %w", path, lineNumber, err)
	}
	return &workout, nil
}

func AppendWorkouts(workouts []model.Workout) (int, error) {
	return AppendWorkoutsPath(config.WorkoutsPath(), workouts)
}

func AppendWorkoutsPath(path string, workouts []model.Workout) (int, error) {
	existing, err := ReadWorkoutsPath(path)
	if err != nil {
		return 0, err
	}

	seen := make(map[int]bool, len(existing)+len(workouts))
	for _, workout := range existing {
		seen[workout.ID] = true
	}

	var toWrite []model.Workout
	for _, workout := range workouts {
		if seen[workout.ID] {
			continue
		}
		seen[workout.ID] = true
		toWrite = append(toWrite, workout)
	}
	if len(toWrite) == 0 {
		return 0, nil
	}

	needsLeadingNewline, err := needsAppendNewline(path)
	if err != nil {
		return 0, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}

	writer := bufio.NewWriter(file)
	if needsLeadingNewline {
		if err := writer.WriteByte('\n'); err != nil {
			_ = file.Close()
			return 0, err
		}
	}
	for _, workout := range toWrite {
		data, err := json.Marshal(workout)
		if err != nil {
			_ = file.Close()
			return 0, err
		}
		if _, err := writer.Write(data); err != nil {
			_ = file.Close()
			return 0, err
		}
		if err := writer.WriteByte('\n'); err != nil {
			_ = file.Close()
			return 0, err
		}
	}
	if err := writer.Flush(); err != nil {
		_ = file.Close()
		return 0, err
	}
	if err := file.Close(); err != nil {
		return 0, err
	}
	return len(toWrite), nil
}

func needsAppendNewline(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return false, err
	}
	if info.Size() == 0 {
		return false, file.Close()
	}
	if _, err := file.Seek(-1, io.SeekEnd); err != nil {
		_ = file.Close()
		return false, err
	}
	last := make([]byte, 1)
	if _, err := file.Read(last); err != nil {
		_ = file.Close()
		return false, err
	}
	needsNewline := last[0] != '\n'
	if err := file.Close(); err != nil {
		return false, err
	}
	return needsNewline, nil
}

func WorkoutCount() (int, error) {
	workouts, err := ReadWorkouts()
	if err != nil {
		return 0, err
	}
	return len(workouts), nil
}
