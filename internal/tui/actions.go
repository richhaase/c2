package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	syncsvc "github.com/richhaase/c2/internal/sync"
)

type SyncService interface {
	Sync(context.Context) (syncsvc.Result, error)
}

type ReportGenerator interface {
	GenerateReport(context.Context, string) error
}

type Exporter interface {
	Export(context.Context, string, string) error
}

type SyncServiceFunc func(context.Context) (syncsvc.Result, error)

func (f SyncServiceFunc) Sync(ctx context.Context) (syncsvc.Result, error) {
	return f(ctx)
}

type ReportGeneratorFunc func(context.Context, string) error

func (f ReportGeneratorFunc) GenerateReport(ctx context.Context, path string) error {
	return f(ctx, path)
}

type ExporterFunc func(context.Context, string, string) error

func (f ExporterFunc) Export(ctx context.Context, format string, path string) error {
	return f(ctx, format, path)
}

type syncCompletedMsg struct {
	Result syncsvc.Result
	Err    error
}

type reportCompletedMsg struct {
	Path string
	Err  error
}

type exportCompletedMsg struct {
	Format string
	Path   string
	Err    error
}

func syncCmd(ctx context.Context, service SyncService) tea.Cmd {
	return func() tea.Msg {
		result, err := service.Sync(ctx)
		return syncCompletedMsg{Result: result, Err: err}
	}
}

func reportCmd(ctx context.Context, generator ReportGenerator, path string) tea.Cmd {
	return func() tea.Msg {
		return reportCompletedMsg{Path: path, Err: generator.GenerateReport(ctx, path)}
	}
}

func exportCmd(ctx context.Context, exporter Exporter, format string, path string) tea.Cmd {
	return func() tea.Msg {
		return exportCompletedMsg{Format: format, Path: path, Err: exporter.Export(ctx, format, path)}
	}
}
