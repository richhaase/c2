package syncer

import (
	"context"
	"fmt"
	"time"

	"github.com/richhaase/c2/internal/api"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/notes"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/storage"
)

const maxStrokeFailuresPerSync = 3

type Client interface {
	GetAllResults(context.Context, api.ResultsFilter) ([]models.Workout, error)
	GetStrokes(context.Context, int64) ([]models.StrokeData, error)
}

type StrokeFailure struct {
	WorkoutID int64
	Err       error
}

type Result struct {
	Since          string
	Fetched        int
	Workouts       storage.UpsertResult
	Strokes        int
	StrokeFailures []StrokeFailure
	Compacted      notes.CompactResult
	TotalWorkouts  int
}

func Run(
	ctx context.Context,
	p paths.DataPaths,
	client Client,
	startedAt time.Time,
	warn func(string),
) (Result, error) {
	meta := storage.ReadMeta(p, warn)
	result := Result{}
	filter := api.ResultsFilter{}
	if meta != nil {
		if meta.LastSync != "" {
			updatedAfter, ok := updatedAfter(meta.LastSync)
			if ok {
				result.Since = meta.LastSync
				filter.UpdatedAfter = updatedAfter
			} else if warn != nil {
				warn(fmt.Sprintf("Warning: meta.json has invalid last_sync %q; performing a full sync.", meta.LastSync))
			}
		}
	}

	workouts, err := client.GetAllResults(ctx, filter)
	if err != nil {
		return Result{}, err
	}
	result.Fetched = len(workouts)
	result.Workouts, err = storage.UpsertWorkouts(p, workouts)
	if err != nil {
		return Result{}, err
	}

	allWorkouts, err := storage.ReadWorkouts(p)
	if err != nil {
		return Result{}, err
	}
	result.Strokes, result.StrokeFailures, err = syncStrokes(ctx, client, p, allWorkouts)
	if err != nil {
		return Result{}, err
	}

	newMeta := storage.StoreMeta{
		SchemaVersion: new(storage.SchemaVersion),
		Created:       startedAt.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		LastSync:      syncTimestamp(startedAt),
	}
	if meta != nil {
		if meta.SchemaVersion != nil {
			newMeta.SchemaVersion = meta.SchemaVersion
		}
		if meta.Created != "" {
			newMeta.Created = meta.Created
		}
	}
	if err := storage.WriteMeta(p, newMeta); err != nil {
		return Result{}, err
	}

	result.Compacted, err = notes.Compact(p, startedAt)
	if err != nil {
		return Result{}, err
	}
	result.TotalWorkouts, err = storage.WorkoutCount(p)
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func syncStrokes(
	ctx context.Context,
	client Client,
	p paths.DataPaths,
	workouts []models.Workout,
) (int, []StrokeFailure, error) {
	count := 0
	var failures []StrokeFailure
	for _, workout := range workouts {
		if !workout.StrokeData {
			continue
		}
		has, err := storage.HasStrokeData(p, workout.ID)
		if err != nil {
			return count, failures, err
		}
		if has {
			continue
		}
		strokes, err := client.GetStrokes(ctx, workout.ID)
		if err != nil {
			failures = append(failures, StrokeFailure{WorkoutID: workout.ID, Err: err})
			if len(failures) >= maxStrokeFailuresPerSync {
				break
			}
			continue
		}
		if len(strokes) == 0 {
			failures = append(failures, StrokeFailure{
				WorkoutID: workout.ID,
				Err:       fmt.Errorf("API returned no stroke samples"),
			})
			if len(failures) >= maxStrokeFailuresPerSync {
				break
			}
			continue
		}
		if err := storage.WriteStrokeData(p, workout.ID, strokes); err != nil {
			return count, failures, err
		}
		count++
	}
	return count, failures, nil
}

func syncTimestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func updatedAfter(stored string) (string, bool) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05"} {
		at, err := time.Parse(layout, stored)
		if err == nil {
			return at.UTC().Format("2006-01-02 15:04:05"), true
		}
	}
	return "", false
}
