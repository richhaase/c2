package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/model"
	syncsvc "github.com/richhaase/c2/internal/sync"
)

func TestInitialModelLoadsDashboard(t *testing.T) {
	m := NewModel(Services{
		LoadConfig: func() (config.Config, error) {
			cfg := config.Default()
			cfg.API.Token = "token"
			cfg.Goal.StartDate = "2026-01-01"
			cfg.Goal.EndDate = "2026-12-31"
			return cfg, nil
		},
		ReadWorkouts: func() ([]model.Workout, error) {
			return []model.Workout{
				{ID: 1, Date: "2026-04-20 07:00:00", Distance: 5000, TimeFormatted: "20:00.0", Time: 12000, Type: "rower"},
				{ID: 2, Date: "2026-04-21 07:00:00", Distance: 7345, TimeFormatted: "30:00.0", Time: 18000, Type: "rower"},
			}, nil
		},
		Now: func() time.Time { return time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC) },
	})

	if m.activeTab != dashboardTab {
		t.Fatalf("activeTab = %v, want dashboardTab", m.activeTab)
	}
	view := m.View()
	for _, want := range []string{"Dashboard", "12,345m", "2 workouts"} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() missing %q:\n%s", want, view)
		}
	}
}

func TestTabNavigation(t *testing.T) {
	m := NewModel(Services{})

	updated, _ := m.Update(key("right"))
	m = updated.(Model)
	if m.activeTab != workoutsTab {
		t.Fatalf("activeTab after right = %v, want workoutsTab", m.activeTab)
	}

	updated, _ = m.Update(key("left"))
	m = updated.(Model)
	if m.activeTab != dashboardTab {
		t.Fatalf("activeTab after left = %v, want dashboardTab", m.activeTab)
	}

	updated, _ = m.Update(key("left"))
	m = updated.(Model)
	if m.activeTab != actionsTab {
		t.Fatalf("activeTab after wrapping left = %v, want actionsTab", m.activeTab)
	}
}

func TestSyncCompletedMessageUpdatesStatus(t *testing.T) {
	m := NewModel(Services{})

	updated, _ := m.Update(syncCompletedMsg{
		Result: syncsvc.Result{
			FetchedWorkouts: 3,
			NewWorkouts:     2,
			StrokeCount:     1,
			TotalWorkouts:   10,
		},
	})
	m = updated.(Model)

	if m.status != "Sync complete: 2 new workouts, 10 total" {
		t.Fatalf("status = %q", m.status)
	}
}

func TestReportCompletedMessageShowsPath(t *testing.T) {
	m := NewModel(Services{})

	updated, _ := m.Update(reportCompletedMsg{Path: "/tmp/c2-report.html"})
	m = updated.(Model)

	if m.status != "Report written to /tmp/c2-report.html" {
		t.Fatalf("status = %q", m.status)
	}
	if !strings.Contains(m.View(), "/tmp/c2-report.html") {
		t.Fatalf("View() missing report path:\n%s", m.View())
	}
}

func TestExportCompletedMessageShowsPath(t *testing.T) {
	m := NewModel(Services{})

	updated, _ := m.Update(exportCompletedMsg{Format: "csv", Path: "/tmp/c2-workouts.csv"})
	m = updated.(Model)

	if m.status != "Export csv written to /tmp/c2-workouts.csv" {
		t.Fatalf("status = %q", m.status)
	}
	if !strings.Contains(m.View(), "/tmp/c2-workouts.csv") {
		t.Fatalf("View() missing export path:\n%s", m.View())
	}
}

func TestActionKeysWhileBusyDoNotLaunchAnotherCommand(t *testing.T) {
	m := NewModel(Services{
		SyncService:     &blockingSyncService{},
		ReportGenerator: &fakeReportGenerator{},
		Exporter:        &fakeExporter{},
	})

	updated, cmd := m.Update(key("s"))
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("first sync key returned nil command")
	}
	if !m.busy {
		t.Fatal("busy = false after starting sync")
	}

	for _, actionKey := range []string{"s", "r", "e"} {
		updated, cmd = m.Update(key(actionKey))
		m = updated.(Model)
		if cmd != nil {
			t.Fatalf("action key %q returned command while busy", actionKey)
		}
	}

	updated, _ = m.Update(syncCompletedMsg{Result: syncsvc.Result{TotalWorkouts: 1}})
	m = updated.(Model)
	if m.busy {
		t.Fatal("busy = true after sync completed")
	}
}

func TestQuitCancelsInFlightActionContext(t *testing.T) {
	service := &observingSyncService{}
	m := NewModel(Services{SyncService: service})

	updated, cmd := m.Update(key("s"))
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("sync key returned nil command")
	}

	updated, _ = m.Update(key("q"))
	m = updated.(Model)

	msg := cmd()
	if _, ok := msg.(syncCompletedMsg); !ok {
		t.Fatalf("sync command message = %T, want syncCompletedMsg", msg)
	}
	if !service.sawCanceled {
		t.Fatal("sync service did not observe canceled context")
	}
}

func key(s string) tea.KeyMsg {
	switch s {
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

type blockingSyncService struct{}

func (blockingSyncService) Sync(context.Context) (syncsvc.Result, error) {
	return syncsvc.Result{}, nil
}

type observingSyncService struct {
	sawCanceled bool
}

func (s *observingSyncService) Sync(ctx context.Context) (syncsvc.Result, error) {
	if ctx.Err() != nil {
		s.sawCanceled = true
	}
	return syncsvc.Result{}, ctx.Err()
}
