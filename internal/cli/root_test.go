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
)

func TestRootVersion(t *testing.T) {
	cmd := NewRootCommand("c2 test-version", Dependencies{
		TUIRunner: func() error { return nil },
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
		TUIRunner: func() error { return nil },
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
		TUIRunner: func() error {
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

func TestScriptableCommandsRejectUnexpectedPositionalArgs(t *testing.T) {
	errRunE := errors.New("command RunE should not execute")
	for _, command := range []string{"setup", "sync", "log", "status", "trend", "export", "report"} {
		t.Run(command, func(t *testing.T) {
			cmd := NewRootCommand("test", Dependencies{
				TUIRunner: func() error { return nil },
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
