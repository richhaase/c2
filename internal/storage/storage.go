package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/richhaase/c2cli/internal/config"
	"github.com/richhaase/c2cli/internal/models"
)

func workoutsPath() (string, error) {
	dir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "workouts.jsonl"), nil
}

func strokesPath(workoutID int64) (string, error) {
	dir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "strokes", fmt.Sprintf("%d.jsonl", workoutID)), nil
}

// ReadWorkouts reads all workouts from the JSONL file.
func ReadWorkouts() ([]models.Workout, error) {
	path, err := workoutsPath()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()

	var workouts []models.Workout
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var w models.Workout
		if err := json.Unmarshal([]byte(line), &w); err != nil {
			return nil, fmt.Errorf("failed to parse workout: %w", err)
		}
		workouts = append(workouts, w)
	}
	return workouts, scanner.Err()
}

// AppendWorkouts appends new workouts, skipping any with IDs already present.
// Returns the number of workouts written.
func AppendWorkouts(newWorkouts []models.Workout) (int, error) {
	existing, err := ReadWorkouts()
	if err != nil {
		return 0, err
	}

	ids := make(map[int64]bool)
	for _, w := range existing {
		ids[w.ID] = true
	}

	path, err := workoutsPath()
	if err != nil {
		return 0, err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, fmt.Errorf("failed to open %s for appending: %w", path, err)
	}
	defer f.Close()

	count := 0
	for _, w := range newWorkouts {
		if ids[w.ID] {
			continue
		}
		data, err := json.Marshal(w)
		if err != nil {
			return count, fmt.Errorf("failed to serialize workout: %w", err)
		}
		if _, err := fmt.Fprintf(f, "%s\n", data); err != nil {
			return count, fmt.Errorf("failed to write workout: %w", err)
		}
		count++
	}
	return count, nil
}

// HasStrokeData checks if stroke data exists for a workout.
func HasStrokeData(workoutID int64) bool {
	path, err := strokesPath(workoutID)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// WriteStrokeData writes stroke data for a workout.
func WriteStrokeData(workoutID int64, strokes []models.StrokeData) error {
	path, err := strokesPath(workoutID)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	defer f.Close()

	for _, s := range strokes {
		data, err := json.Marshal(s)
		if err != nil {
			return fmt.Errorf("failed to serialize stroke: %w", err)
		}
		fmt.Fprintf(f, "%s\n", data)
	}
	return nil
}

// WorkoutCount returns the number of stored workouts.
func WorkoutCount() (int, error) {
	path, err := workoutsPath()
	if err != nil {
		return 0, err
	}

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if scanner.Text() != "" {
			count++
		}
	}
	return count, scanner.Err()
}
