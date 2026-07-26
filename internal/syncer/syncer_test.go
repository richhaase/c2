package syncer

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/richhaase/c2/internal/api"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/storage"
	"github.com/richhaase/c2/internal/store"
)

type fakeClient struct {
	results     []models.Workout
	resultsErr  error
	filter      api.ResultsFilter
	strokes     map[int64][]models.StrokeData
	strokeErrs  map[int64]error
	strokeCalls []int64
}

func (f *fakeClient) GetAllResults(_ context.Context, filter api.ResultsFilter) ([]models.Workout, error) {
	f.filter = filter
	return f.results, f.resultsErr
}

func (f *fakeClient) GetStrokes(_ context.Context, id int64) ([]models.StrokeData, error) {
	f.strokeCalls = append(f.strokeCalls, id)
	if err := f.strokeErrs[id]; err != nil {
		return nil, err
	}
	return f.strokes[id], nil
}

func syncFixture(t *testing.T) (paths.DataPaths, time.Time) {
	t.Helper()
	p := paths.For(t.TempDir())
	startedAt := time.Date(2026, 7, 20, 18, 34, 56, 789, time.UTC)
	if err := store.Init(p, startedAt.AddDate(0, 0, -10), nil); err != nil {
		t.Fatal(err)
	}
	meta := storage.ReadMeta(p, nil)
	meta.LastSync = "2026-07-19T17:00:00Z"
	if err := storage.WriteMeta(p, *meta); err != nil {
		t.Fatal(err)
	}
	return p, startedAt
}

func TestRunUsesUpdateWatermarkUpsertsAndRetriesStrokes(t *testing.T) {
	p, startedAt := syncFixture(t)
	existing := models.Workout{
		ID:         1,
		Date:       "2026-07-01 08:00:00",
		Distance:   1000,
		StrokeData: true,
	}
	if _, err := storage.AppendWorkouts(p, []models.Workout{existing}); err != nil {
		t.Fatal(err)
	}
	pace := 1750.0
	client := &fakeClient{
		results: []models.Workout{
			{ID: 1, Date: existing.Date, Distance: 1500, StrokeData: true},
			{ID: 2, Date: "2026-07-02 08:00:00", Distance: 2000, StrokeData: true},
		},
		strokes: map[int64][]models.StrokeData{
			2: {{P: &pace}},
		},
		strokeErrs: map[int64]error{
			1: errors.New("temporary"),
		},
	}
	result, err := Run(context.Background(), p, client, startedAt, nil)
	if err != nil {
		t.Fatal(err)
	}
	if client.filter.From != "" || client.filter.UpdatedAfter != "2026-07-19 17:00:00" {
		t.Fatalf("filter = %#v", client.filter)
	}
	if result.Workouts.Added != 1 || result.Workouts.Updated != 1 {
		t.Fatalf("workouts = %#v", result.Workouts)
	}
	if result.Strokes != 1 || len(result.StrokeFailures) != 1 || result.StrokeFailures[0].WorkoutID != 1 {
		t.Fatalf("stroke result = %#v", result)
	}
	workouts, err := storage.ReadWorkouts(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(workouts) != 2 || workouts[0].Distance != 1500 {
		t.Fatalf("stored workouts = %#v", workouts)
	}
	meta := storage.ReadMeta(p, nil)
	if meta == nil || meta.LastSync != "2026-07-20T18:34:56Z" || meta.StrokeCursor != 2 {
		t.Fatalf("meta = %#v", meta)
	}

	client.results = nil
	client.strokeErrs = nil
	client.strokes[1] = []models.StrokeData{{P: &pace}}
	client.strokeCalls = nil
	second, err := Run(context.Background(), p, client, startedAt.Add(time.Hour), nil)
	if err != nil {
		t.Fatal(err)
	}
	if second.Strokes != 1 || len(client.strokeCalls) != 1 || client.strokeCalls[0] != 1 {
		t.Fatalf("second sync = %#v, calls = %v", second, client.strokeCalls)
	}
}

func TestRunFallsBackToFullSyncForInvalidWatermark(t *testing.T) {
	p, startedAt := syncFixture(t)
	meta := storage.ReadMeta(p, nil)
	meta.LastSync = "not-a-time"
	if err := storage.WriteMeta(p, *meta); err != nil {
		t.Fatal(err)
	}
	client := &fakeClient{}
	var warnings []string
	if _, err := Run(context.Background(), p, client, startedAt, func(s string) {
		warnings = append(warnings, s)
	}); err != nil {
		t.Fatal(err)
	}
	if client.filter.UpdatedAfter != "" {
		t.Fatalf("filter = %#v", client.filter)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "full sync") {
		t.Fatalf("warnings = %v", warnings)
	}
}

func TestRunDoesNotAdvanceWatermarkWhenResultsFail(t *testing.T) {
	p, startedAt := syncFixture(t)
	client := &fakeClient{resultsErr: errors.New("offline")}
	if _, err := Run(context.Background(), p, client, startedAt, nil); err == nil {
		t.Fatal("Run succeeded")
	}
	meta := storage.ReadMeta(p, nil)
	if meta == nil || meta.LastSync != "2026-07-19T17:00:00Z" {
		t.Fatalf("meta = %#v", meta)
	}
	if _, err := os.Stat(p.Workouts); !os.IsNotExist(err) {
		t.Fatalf("workouts stat error = %v", err)
	}
}

func TestSyncStrokesStopsAfterRepeatedFailures(t *testing.T) {
	p := paths.For(t.TempDir())
	if err := os.MkdirAll(p.StrokesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	workouts := make([]models.Workout, maxStrokeFailuresPerSync+2)
	client := &fakeClient{strokeErrs: map[int64]error{}}
	for i := range workouts {
		id := int64(i + 1)
		workouts[i] = models.Workout{ID: id, StrokeData: true}
		client.strokeErrs[id] = errors.New("offline")
	}

	count, failures, cursor, err := syncStrokes(context.Background(), client, p, workouts, 0)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 || len(failures) != maxStrokeFailuresPerSync {
		t.Fatalf("count = %d, failures = %d", count, len(failures))
	}
	if len(client.strokeCalls) != maxStrokeFailuresPerSync {
		t.Fatalf("stroke calls = %v", client.strokeCalls)
	}
	if cursor != int64(maxStrokeFailuresPerSync) {
		t.Fatalf("cursor = %d", cursor)
	}

	pace := 1750.0
	client.strokeCalls = nil
	client.strokes = map[int64][]models.StrokeData{
		4: {{P: &pace}},
		5: {{P: &pace}},
	}
	delete(client.strokeErrs, 4)
	delete(client.strokeErrs, 5)
	count, failures, cursor, err = syncStrokes(context.Background(), client, p, workouts, cursor)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 || len(failures) != maxStrokeFailuresPerSync {
		t.Fatalf("count = %d, failures = %d", count, len(failures))
	}
	if len(client.strokeCalls) != len(workouts) {
		t.Fatalf("stroke calls = %v", client.strokeCalls)
	}
	if client.strokeCalls[0] != 4 || client.strokeCalls[1] != 5 {
		t.Fatalf("stroke calls = %v", client.strokeCalls)
	}
	if cursor != 3 {
		t.Fatalf("cursor = %d", cursor)
	}
}
