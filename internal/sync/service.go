package sync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/richhaase/c2/internal/api"
	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/model"
	"github.com/richhaase/c2/internal/storage"
)

type API interface {
	GetAllResults(ctx context.Context, from string, to string) ([]model.Workout, error)
	GetStrokes(ctx context.Context, workoutID int) ([]model.StrokeData, error)
}

type Store interface {
	AppendWorkouts(workouts []model.Workout) (int, error)
	HasStrokeData(workoutID int) (bool, error)
	WriteStrokeData(workoutID int, strokes []model.StrokeData) error
	WorkoutCount() (int, error)
}

type Service struct {
	Config     config.Config
	API        API
	Store      Store
	SaveConfig func(config.Config) error
	EnsureDirs func() error
	Now        func() time.Time
}

type Result struct {
	FetchedWorkouts int
	NewWorkouts     int
	StrokeCount     int
	TotalWorkouts   int
	Warnings        []string
}

var ErrMissingToken = errors.New("no API token configured; run `c2 setup` first")

func NewService(cfg config.Config, client *api.Client) Service {
	return Service{
		Config:     cfg,
		API:        client,
		Store:      fileStore{},
		SaveConfig: config.Save,
		EnsureDirs: config.EnsureDirs,
		Now:        time.Now,
	}
}

func (s Service) Sync(ctx context.Context) (Result, error) {
	if s.Config.API.Token == "" {
		return Result{}, ErrMissingToken
	}
	if s.EnsureDirs != nil {
		if err := s.EnsureDirs(); err != nil {
			return Result{}, err
		}
	}
	if s.API == nil {
		return Result{}, errors.New("sync API is not configured")
	}
	store := s.Store
	if store == nil {
		store = fileStore{}
	}

	result := Result{}
	workouts, err := s.API.GetAllResults(ctx, s.Config.Sync.LastSync, "")
	if err != nil {
		return Result{}, err
	}
	result.FetchedWorkouts = len(workouts)

	written, err := store.AppendWorkouts(workouts)
	if err != nil {
		return Result{}, err
	}
	result.NewWorkouts = written

	strokeCount, warnings, err := s.syncStrokes(ctx, store, workouts)
	if err != nil {
		return Result{}, err
	}
	result.StrokeCount = strokeCount
	result.Warnings = warnings

	s.Config.Sync.LastSync = s.now().UTC().Format(time.RFC3339)
	if s.SaveConfig != nil {
		if err := s.SaveConfig(s.Config); err != nil {
			return Result{}, err
		}
	}

	total, err := store.WorkoutCount()
	if err != nil {
		return Result{}, err
	}
	result.TotalWorkouts = total
	return result, nil
}

func (s Service) syncStrokes(ctx context.Context, store Store, workouts []model.Workout) (int, []string, error) {
	count := 0
	failures := 0
	var warnings []string
	for _, workout := range workouts {
		if !workout.StrokeData {
			continue
		}
		hasStrokes, err := store.HasStrokeData(workout.ID)
		if err != nil {
			return 0, nil, err
		}
		if hasStrokes {
			continue
		}

		strokes, err := s.API.GetStrokes(ctx, workout.ID)
		if err != nil {
			failures++
			warnings = append(warnings, fmt.Sprintf("Warning: failed to fetch strokes for workout %d: %v", workout.ID, err))
			if failures >= 3 {
				warnings = append(warnings, "Too many failures, skipping remaining stroke data.")
				break
			}
			continue
		}
		if len(strokes) == 0 {
			continue
		}
		if err := store.WriteStrokeData(workout.ID, strokes); err != nil {
			return 0, nil, err
		}
		count++
	}
	return count, warnings, nil
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

type fileStore struct{}

func (fileStore) AppendWorkouts(workouts []model.Workout) (int, error) {
	return storage.AppendWorkouts(workouts)
}

func (fileStore) HasStrokeData(workoutID int) (bool, error) {
	return storage.HasStrokeData(workoutID)
}

func (fileStore) WriteStrokeData(workoutID int, strokes []model.StrokeData) error {
	return storage.WriteStrokeData(workoutID, strokes)
}

func (fileStore) WorkoutCount() (int, error) {
	return storage.WorkoutCount()
}
