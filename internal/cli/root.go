package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/richhaase/c2/internal/api"
	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/model"
	"github.com/richhaase/c2/internal/storage"
	syncsvc "github.com/richhaase/c2/internal/sync"
	"github.com/richhaase/c2/internal/tui"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	TUIRunner    func(tui.Services) error
	LoadConfig   func() (config.Config, error)
	SaveConfig   func(config.Config) error
	EnsureDirs   func() error
	ReadWorkouts func() ([]model.Workout, error)
	RunSync      func(context.Context, config.Config, string) (syncsvc.Result, error)
	VerifyUser   func(context.Context, config.Config, string) (model.UserProfile, error)
	WriteFile    func(string, []byte, os.FileMode) error
	TempDir      func(string, string) (string, error)
	OpenFile     func(string) error
	Now          func() time.Time
	Stdin        io.Reader
}

func DefaultDependencies() Dependencies {
	return Dependencies{
		TUIRunner:    tui.RunWithServices,
		LoadConfig:   config.Load,
		SaveConfig:   config.Save,
		EnsureDirs:   config.EnsureDirs,
		ReadWorkouts: storage.ReadWorkouts,
		RunSync: func(ctx context.Context, cfg config.Config, version string) (syncsvc.Result, error) {
			client := api.FromConfig(cfg, version)
			service := syncsvc.NewService(cfg, client)
			return service.Sync(ctx)
		},
		VerifyUser: func(ctx context.Context, cfg config.Config, version string) (model.UserProfile, error) {
			return api.FromConfig(cfg, version).GetUser(ctx)
		},
		WriteFile: os.WriteFile,
		TempDir:   os.MkdirTemp,
		OpenFile:  openFile,
		Now:       time.Now,
		Stdin:     os.Stdin,
	}
}

func NewRootCommand(version string, deps Dependencies) *cobra.Command {
	deps = withDefaults(deps)
	cmd := &cobra.Command{
		Use:           "c2",
		Short:         "Concept2 Logbook CLI",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.TUIRunner(newTUIServices(version, deps))
		},
	}

	cmd.AddCommand(newSetupCommand(version, deps))
	cmd.AddCommand(newSyncCommand(version, deps))
	cmd.AddCommand(newLogCommand(deps))
	cmd.AddCommand(newStatusCommand(deps))
	cmd.AddCommand(newTrendCommand(deps))
	cmd.AddCommand(newExportCommand(deps))
	cmd.AddCommand(newReportCommand(deps))
	cmd.SetVersionTemplate("{{.Version}}\n")
	return cmd
}

func newTUIServices(version string, deps Dependencies) tui.Services {
	return tui.Services{
		LoadConfig:   deps.LoadConfig,
		ReadWorkouts: deps.ReadWorkouts,
		SyncService: tui.SyncServiceFunc(func(ctx context.Context) (syncsvc.Result, error) {
			cfg, err := deps.LoadConfig()
			if err != nil {
				return syncsvc.Result{}, err
			}
			return deps.RunSync(ctx, cfg, version)
		}),
		ReportGenerator: tui.NewReportGenerator(deps.ReadWorkouts, deps.LoadConfig, deps.WriteFile, deps.Now),
		Exporter:        tui.NewExporter(deps.ReadWorkouts, deps.WriteFile),
		Now:             deps.Now,
	}
}

func withDefaults(deps Dependencies) Dependencies {
	defaults := DefaultDependencies()
	if deps.TUIRunner == nil {
		deps.TUIRunner = defaults.TUIRunner
	}
	if deps.LoadConfig == nil {
		deps.LoadConfig = defaults.LoadConfig
	}
	if deps.SaveConfig == nil {
		deps.SaveConfig = defaults.SaveConfig
	}
	if deps.EnsureDirs == nil {
		deps.EnsureDirs = defaults.EnsureDirs
	}
	if deps.ReadWorkouts == nil {
		deps.ReadWorkouts = defaults.ReadWorkouts
	}
	if deps.RunSync == nil {
		deps.RunSync = defaults.RunSync
	}
	if deps.VerifyUser == nil {
		deps.VerifyUser = defaults.VerifyUser
	}
	if deps.WriteFile == nil {
		deps.WriteFile = defaults.WriteFile
	}
	if deps.TempDir == nil {
		deps.TempDir = defaults.TempDir
	}
	if deps.OpenFile == nil {
		deps.OpenFile = defaults.OpenFile
	}
	if deps.Now == nil {
		deps.Now = defaults.Now
	}
	if deps.Stdin == nil {
		deps.Stdin = defaults.Stdin
	}
	return deps
}

func positiveIntFlag(value int, name string) error {
	if value < 1 {
		return fmt.Errorf("--%s must be a positive integer", name)
	}
	return nil
}

func openFile(path string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", path)
	case "windows":
		command = exec.Command("cmd", "/c", "start", "", path)
	default:
		command = exec.Command("xdg-open", path)
	}
	return command.Start()
}
