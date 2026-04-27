package sync

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/model"
)

func TestServiceSyncAppendsNewWorkoutsAndUpdatesLastSync(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 34, 56, 789000000, time.UTC)
	cfg := config.Default()
	cfg.API.Token = "token"
	cfg.Sync.LastSync = "2026-04-01T00:00:00Z"
	api := &fakeAPI{
		workouts: []model.Workout{
			{ID: 101, StrokeData: true},
			{ID: 102, StrokeData: true},
			{ID: 103, StrokeData: false},
		},
		strokes: map[int][]model.StrokeData{
			101: {{T: floatPtr(1)}},
			102: {{T: floatPtr(2)}},
		},
	}
	store := &fakeStore{
		existingStrokes: map[int]bool{102: true},
		newWorkoutCount: 2,
		totalCount:      12,
	}
	var saved config.Config
	service := Service{
		Config: cfg,
		API:    api,
		Store:  store,
		SaveConfig: func(cfg config.Config) error {
			saved = cfg
			return nil
		},
		EnsureDirs: func() error { return nil },
		Now:        func() time.Time { return now },
	}

	result, err := service.Sync(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if api.from != "2026-04-01T00:00:00Z" || api.to != "" {
		t.Fatalf("GetAllResults called with from %q to %q", api.from, api.to)
	}
	if len(store.appended) != 3 {
		t.Fatalf("AppendWorkouts got %d workouts, want 3", len(store.appended))
	}
	if result.FetchedWorkouts != 3 || result.NewWorkouts != 2 || result.StrokeCount != 1 || result.TotalWorkouts != 12 {
		t.Fatalf("result = %#v", result)
	}
	if saved.Sync.LastSync != "2026-04-27T12:34:56Z" {
		t.Fatalf("LastSync = %q, want UTC RFC3339 without milliseconds", saved.Sync.LastSync)
	}
	if len(api.strokeIDs) != 1 || api.strokeIDs[0] != 101 {
		t.Fatalf("stroke fetches = %v, want [101]", api.strokeIDs)
	}
	if len(store.writtenStrokes) != 1 || len(store.writtenStrokes[101]) != 1 {
		t.Fatalf("written strokes = %#v, want strokes for 101", store.writtenStrokes)
	}
}

func TestServiceSyncSkipsStrokesAfterThreeFailures(t *testing.T) {
	cfg := config.Default()
	cfg.API.Token = "token"
	api := &fakeAPI{
		workouts: []model.Workout{
			{ID: 101, StrokeData: true},
			{ID: 102, StrokeData: true},
			{ID: 103, StrokeData: true},
			{ID: 104, StrokeData: true},
		},
		strokeErr: errors.New("strokes unavailable"),
	}
	store := &fakeStore{
		existingStrokes: map[int]bool{},
		totalCount:      4,
	}
	service := Service{
		Config:     cfg,
		API:        api,
		Store:      store,
		SaveConfig: func(config.Config) error { return nil },
		EnsureDirs: func() error { return nil },
		Now:        func() time.Time { return time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC) },
	}

	result, err := service.Sync(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(api.strokeIDs) != 3 {
		t.Fatalf("stroke fetch count = %d, want 3", len(api.strokeIDs))
	}
	for i, want := range []int{101, 102, 103} {
		if api.strokeIDs[i] != want {
			t.Fatalf("strokeIDs[%d] = %d, want %d", i, api.strokeIDs[i], want)
		}
	}
	if result.StrokeCount != 0 {
		t.Fatalf("StrokeCount = %d, want 0", result.StrokeCount)
	}
	if len(result.Warnings) != 4 {
		t.Fatalf("warnings = %#v, want three failures plus stop warning", result.Warnings)
	}
	if !strings.Contains(result.Warnings[0], "workout 101") || !strings.Contains(result.Warnings[3], "Too many failures") {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}

type fakeAPI struct {
	workouts  []model.Workout
	strokes   map[int][]model.StrokeData
	strokeErr error
	from      string
	to        string
	strokeIDs []int
}

func (f *fakeAPI) GetAllResults(_ context.Context, from string, to string) ([]model.Workout, error) {
	f.from = from
	f.to = to
	return f.workouts, nil
}

func (f *fakeAPI) GetStrokes(_ context.Context, workoutID int) ([]model.StrokeData, error) {
	f.strokeIDs = append(f.strokeIDs, workoutID)
	if f.strokeErr != nil {
		return nil, f.strokeErr
	}
	return f.strokes[workoutID], nil
}

type fakeStore struct {
	existingStrokes map[int]bool
	newWorkoutCount int
	totalCount      int
	appended        []model.Workout
	writtenStrokes  map[int][]model.StrokeData
}

func (f *fakeStore) AppendWorkouts(workouts []model.Workout) (int, error) {
	f.appended = append([]model.Workout(nil), workouts...)
	return f.newWorkoutCount, nil
}

func (f *fakeStore) HasStrokeData(workoutID int) (bool, error) {
	return f.existingStrokes[workoutID], nil
}

func (f *fakeStore) WriteStrokeData(workoutID int, strokes []model.StrokeData) error {
	if f.writtenStrokes == nil {
		f.writtenStrokes = make(map[int][]model.StrokeData)
	}
	f.writtenStrokes[workoutID] = strokes
	return nil
}

func (f *fakeStore) WorkoutCount() (int, error) {
	return f.totalCount, nil
}

func floatPtr(v float64) *float64 {
	return &v
}
