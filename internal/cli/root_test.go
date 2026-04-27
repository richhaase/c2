package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/model"
	syncsvc "github.com/richhaase/c2/internal/sync"
	"github.com/richhaase/c2/internal/tui"
)

func TestRootVersion(t *testing.T) {
	cmd := NewRootCommand("c2 test-version", Dependencies{
		TUIRunner: func(tui.Services) error { return nil },
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--version"})

	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got, want := out.String(), "c2 test-version\n"; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}

func TestExplicitCommandsAreRegistered(t *testing.T) {
	cmd := NewRootCommand("test", Dependencies{
		TUIRunner: func(tui.Services) error { return nil },
	})

	for _, name := range []string{"setup", "sync", "log", "status", "trend", "export", "report"} {
		if found, _, err := cmd.Find([]string{name}); err != nil || found == nil || found.Name() != name {
			t.Fatalf("Find(%q) = command %v, err %v; want registered command", name, found, err)
		}
	}
}

func TestBareRootRunsTUIAction(t *testing.T) {
	calls := 0
	cmd := NewRootCommand("test", Dependencies{
		TUIRunner: func(tui.Services) error {
			calls++
			return nil
		},
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("TUIRunner calls = %d, want 1", calls)
	}
}

func TestBareRootPassesInjectedServicesToTUIRunner(t *testing.T) {
	cfg := config.Default()
	cfg.API.Token = "token"
	cfg.Goal.StartDate = "2026-01-01"
	cfg.Goal.EndDate = "2026-12-31"
	workouts := []model.Workout{{ID: 1, Date: "2026-04-20 07:00:00", Distance: 5000}}
	var got tui.Services
	runSyncVersion := ""

	cmd := NewRootCommand("test-version", Dependencies{
		TUIRunner: func(services tui.Services) error {
			got = services
			return nil
		},
		LoadConfig: func() (config.Config, error) {
			return cfg, nil
		},
		ReadWorkouts: func() ([]model.Workout, error) {
			return workouts, nil
		},
		RunSync: func(_ context.Context, gotCfg config.Config, version string) (syncsvc.Result, error) {
			if gotCfg.API.Token != "token" {
				t.Fatalf("RunSync cfg token = %q", gotCfg.API.Token)
			}
			runSyncVersion = version
			return syncsvc.Result{TotalWorkouts: 7}, nil
		},
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.LoadConfig == nil || got.ReadWorkouts == nil || got.SyncService == nil {
		t.Fatalf("TUI services not wired: %+v", got)
	}
	loadedCfg, err := got.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loadedCfg.API.Token != "token" {
		t.Fatalf("LoadConfig token = %q", loadedCfg.API.Token)
	}
	loadedWorkouts, err := got.ReadWorkouts()
	if err != nil {
		t.Fatalf("ReadWorkouts() error = %v", err)
	}
	if len(loadedWorkouts) != 1 || loadedWorkouts[0].ID != 1 {
		t.Fatalf("ReadWorkouts() = %+v", loadedWorkouts)
	}
	result, err := got.SyncService.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if result.TotalWorkouts != 7 {
		t.Fatalf("Sync total = %d, want 7", result.TotalWorkouts)
	}
	if runSyncVersion != "test-version" {
		t.Fatalf("RunSync version = %q", runSyncVersion)
	}
}

func TestScriptableCommandsRejectUnexpectedPositionalArgs(t *testing.T) {
	errRunE := errors.New("command RunE should not execute")
	for _, command := range []string{"setup", "sync", "log", "status", "trend", "export", "report"} {
		t.Run(command, func(t *testing.T) {
			cmd := NewRootCommand("test", Dependencies{
				TUIRunner: func(tui.Services) error { return nil },
				LoadConfig: func() (config.Config, error) {
					return config.Config{}, errRunE
				},
				ReadWorkouts: func() ([]model.Workout, error) {
					return nil, errRunE
				},
				RunSync: func(context.Context, config.Config, string) (syncsvc.Result, error) {
					return syncsvc.Result{}, errRunE
				},
				Stdin: strings.NewReader(""),
			})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs([]string{command, "unexpected"})

			err := cmd.Execute()
			if err == nil {
				t.Fatal("Execute() error = nil, want error")
			}
			if errors.Is(err, errRunE) {
				t.Fatalf("Execute() error = %v, command RunE executed instead of rejecting positional arg", err)
			}
		})
	}
}
