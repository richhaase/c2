package tui

import (
	"context"
	"errors"
	"testing"

	syncsvc "github.com/richhaase/c2/internal/sync"
)

func TestSyncCmdReturnsCompletionMessage(t *testing.T) {
	service := &fakeSyncService{result: syncsvc.Result{NewWorkouts: 4, TotalWorkouts: 9}}

	msg := syncCmd(context.Background(), service)()

	got, ok := msg.(syncCompletedMsg)
	if !ok {
		t.Fatalf("syncCmd() message = %T, want syncCompletedMsg", msg)
	}
	if got.Result.NewWorkouts != 4 || got.Result.TotalWorkouts != 9 || got.Err != nil {
		t.Fatalf("syncCompletedMsg = %+v", got)
	}
	if !service.called {
		t.Fatal("sync service was not called")
	}
}

func TestSyncCmdReturnsErrors(t *testing.T) {
	errSync := errors.New("sync failed")

	msg := syncCmd(context.Background(), &fakeSyncService{err: errSync})()

	got, ok := msg.(syncCompletedMsg)
	if !ok {
		t.Fatalf("syncCmd() message = %T, want syncCompletedMsg", msg)
	}
	if !errors.Is(got.Err, errSync) {
		t.Fatalf("Err = %v, want %v", got.Err, errSync)
	}
}

func TestReportCmdReturnsCompletionMessage(t *testing.T) {
	generator := &fakeReportGenerator{}

	msg := reportCmd(context.Background(), generator, "/tmp/report.html")()

	got, ok := msg.(reportCompletedMsg)
	if !ok {
		t.Fatalf("reportCmd() message = %T, want reportCompletedMsg", msg)
	}
	if got.Path != "/tmp/report.html" || got.Err != nil {
		t.Fatalf("reportCompletedMsg = %+v", got)
	}
	if generator.path != "/tmp/report.html" {
		t.Fatalf("report path = %q", generator.path)
	}
}

func TestExportCmdReturnsCompletionMessage(t *testing.T) {
	exporter := &fakeExporter{}

	msg := exportCmd(context.Background(), exporter, "csv", "/tmp/workouts.csv")()

	got, ok := msg.(exportCompletedMsg)
	if !ok {
		t.Fatalf("exportCmd() message = %T, want exportCompletedMsg", msg)
	}
	if got.Format != "csv" || got.Path != "/tmp/workouts.csv" || got.Err != nil {
		t.Fatalf("exportCompletedMsg = %+v", got)
	}
	if exporter.format != "csv" || exporter.path != "/tmp/workouts.csv" {
		t.Fatalf("export call = format %q path %q", exporter.format, exporter.path)
	}
}

type fakeSyncService struct {
	result syncsvc.Result
	err    error
	called bool
}

func (f *fakeSyncService) Sync(context.Context) (syncsvc.Result, error) {
	f.called = true
	return f.result, f.err
}

type fakeReportGenerator struct {
	path string
	err  error
}

func (f *fakeReportGenerator) GenerateReport(_ context.Context, path string) error {
	f.path = path
	return f.err
}

type fakeExporter struct {
	format string
	path   string
	err    error
}

func (f *fakeExporter) Export(_ context.Context, format string, path string) error {
	f.format = format
	f.path = path
	return f.err
}
