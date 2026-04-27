package tui

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/model"
)

func TestReportGeneratorDoesNotWriteWhenContextCanceledBeforeWrite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	writerCalled := false
	generator := NewReportGenerator(
		func() ([]model.Workout, error) {
			return []model.Workout{testWorkout(1)}, nil
		},
		func() (config.Config, error) {
			cancel()
			cfg := config.Default()
			cfg.Goal.StartDate = "2026-01-01"
			cfg.Goal.EndDate = "2026-12-31"
			return cfg, nil
		},
		func(string, []byte, os.FileMode) error {
			writerCalled = true
			return nil
		},
		func() time.Time { return time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC) },
	)

	err := generator.GenerateReport(ctx, "report.html")

	if err == nil {
		t.Fatal("GenerateReport() error = nil, want context cancellation")
	}
	if writerCalled {
		t.Fatal("GenerateReport() called writer after context cancellation")
	}
}

func TestExporterDoesNotWriteWhenContextCanceledBeforeWrite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	writerCalled := false
	exporter := NewExporter(
		func() ([]model.Workout, error) {
			return manyWorkouts(50000), nil
		},
		func(string, []byte, os.FileMode) error {
			writerCalled = true
			return nil
		},
	)

	go func() {
		time.Sleep(time.Millisecond)
		cancel()
	}()
	err := exporter.Export(ctx, "csv", "workouts.csv")

	if err == nil {
		t.Fatal("Export() error = nil, want context cancellation")
	}
	if writerCalled {
		t.Fatal("Export() called writer after context cancellation")
	}
}

func testWorkout(id int) model.Workout {
	return model.Workout{
		ID:            id,
		Date:          "2026-04-20 07:00:00",
		Distance:      5000,
		Time:          12000,
		TimeFormatted: "20:00.0",
		Type:          "rower",
		Comments:      strings.Repeat("steady ", 20),
	}
}

func manyWorkouts(count int) []model.Workout {
	workouts := make([]model.Workout, 0, count)
	for i := 0; i < count; i++ {
		workouts = append(workouts, testWorkout(i+1))
	}
	return workouts
}
