package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richhaase/c2/internal/api"
	"github.com/richhaase/c2/internal/config"
	exporter "github.com/richhaase/c2/internal/export"
	"github.com/richhaase/c2/internal/model"
	"github.com/richhaase/c2/internal/report"
	"github.com/richhaase/c2/internal/storage"
	syncsvc "github.com/richhaase/c2/internal/sync"
)

func Run() error {
	services, err := DefaultServices("")
	if err != nil {
		return err
	}
	return RunWithServices(services)
}

func RunWithServices(services Services) error {
	program := tea.NewProgram(NewModel(services), tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func DefaultServices(version string) (Services, error) {
	cfg, err := config.Load()
	if err != nil {
		return Services{}, err
	}
	return Services{
		LoadConfig:      config.Load,
		ReadWorkouts:    storage.ReadWorkouts,
		SyncService:     syncsvc.NewService(cfg, api.FromConfig(cfg, version)),
		ReportGenerator: NewReportGenerator(storage.ReadWorkouts, config.Load, os.WriteFile, time.Now),
		Exporter:        NewExporter(storage.ReadWorkouts, os.WriteFile),
		Now:             time.Now,
	}, nil
}

func NewReportGenerator(
	readWorkouts func() ([]model.Workout, error),
	loadConfig func() (config.Config, error),
	writeFile func(string, []byte, os.FileMode) error,
	now func() time.Time,
) ReportGenerator {
	return fileReportGenerator{
		readWorkouts: readWorkouts,
		loadConfig:   loadConfig,
		writeFile:    writeFile,
		now:          now,
	}
}

func NewExporter(
	readWorkouts func() ([]model.Workout, error),
	writeFile func(string, []byte, os.FileMode) error,
) Exporter {
	return fileExporter{
		readWorkouts: readWorkouts,
		writeFile:    writeFile,
	}
}

type fileReportGenerator struct {
	readWorkouts func() ([]model.Workout, error)
	loadConfig   func() (config.Config, error)
	writeFile    func(string, []byte, os.FileMode) error
	now          func() time.Time
}

func (g fileReportGenerator) GenerateReport(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	workouts, err := g.readWorkouts()
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(workouts) == 0 {
		return fmt.Errorf("no workouts found. Run `c2 sync` first")
	}
	cfg, err := g.loadConfig()
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	html, err := report.HTML(workouts, cfg.Goal, 12, g.now())
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return g.writeFile(absPath, []byte(html), 0o644)
}

type fileExporter struct {
	readWorkouts func() ([]model.Workout, error)
	writeFile    func(string, []byte, os.FileMode) error
}

func (e fileExporter) Export(ctx context.Context, format string, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	workouts, err := e.readWorkouts()
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(workouts) == 0 {
		return fmt.Errorf("no workouts found. Run `c2 sync` first")
	}
	workouts = append([]model.Workout(nil), workouts...)
	sort.Slice(workouts, func(i, j int) bool { return workouts[i].Date < workouts[j].Date })

	var (
		output    string
		exportErr error
	)
	switch format {
	case "csv":
		output, exportErr = exporter.CSV(workouts)
	case "json":
		output, exportErr = exporter.JSON(workouts)
	case "jsonl":
		output, exportErr = exporter.JSONL(workouts)
	default:
		return fmt.Errorf("unsupported format %q: must be csv, json, or jsonl", format)
	}
	if exportErr != nil {
		return exportErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return e.writeFile(absPath, []byte(output), 0o644)
}
