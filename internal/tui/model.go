package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/model"
)

type tab int

const (
	dashboardTab tab = iota
	workoutsTab
	trendsTab
	detailTab
	actionsTab
)

var allTabs = []tab{dashboardTab, workoutsTab, trendsTab, detailTab, actionsTab}

type Services struct {
	LoadConfig      func() (config.Config, error)
	ReadWorkouts    func() ([]model.Workout, error)
	SyncService     SyncService
	ReportGenerator ReportGenerator
	Exporter        Exporter
	Now             func() time.Time
}

type Model struct {
	services       Services
	cfg            config.Config
	workouts       []model.Workout
	activeTab      tab
	status         string
	lastReportPath string
	lastExportPath string
	now            time.Time
	busy           bool
	cancelAction   context.CancelFunc
}

func NewModel(services Services) Model {
	services = withServiceDefaults(services)
	m := Model{
		services:  services,
		cfg:       config.Default(),
		activeTab: dashboardTab,
		now:       services.Now(),
	}

	cfg, err := services.LoadConfig()
	if err != nil {
		m.status = fmt.Sprintf("Config unavailable: %v", err)
	} else {
		m.cfg = cfg
		if cfg.API.Token == "" {
			m.status = "No API token configured. Run `c2 setup` first."
		}
	}

	workouts, err := services.ReadWorkouts()
	if err != nil {
		m.status = fmt.Sprintf("Workouts unavailable: %v", err)
	} else {
		m.workouts = workouts
		if len(workouts) == 0 && m.status == "" {
			m.status = "No workouts found. Run `c2 sync` first."
		}
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.cancelInFlightAction()
			return m, tea.Quit
		case "right", "tab", "l":
			m.activeTab = m.nextTab()
		case "left", "shift+tab", "h":
			m.activeTab = m.previousTab()
		case "s":
			if m.busy {
				return m, nil
			}
			if m.services.SyncService == nil {
				m.status = "Sync is unavailable."
				return m, nil
			}
			ctx, cancel := context.WithCancel(context.Background())
			m.busy = true
			m.cancelAction = cancel
			m.status = "Syncing..."
			return m, syncCmd(ctx, m.services.SyncService)
		case "r":
			if m.busy {
				return m, nil
			}
			if m.services.ReportGenerator == nil {
				m.status = "Report generation is unavailable."
				return m, nil
			}
			ctx, cancel := context.WithCancel(context.Background())
			m.busy = true
			m.cancelAction = cancel
			path := defaultReportPath(m.services.Now())
			m.status = "Generating report..."
			return m, reportCmd(ctx, m.services.ReportGenerator, path)
		case "e":
			if m.busy {
				return m, nil
			}
			if m.services.Exporter == nil {
				m.status = "Export is unavailable."
				return m, nil
			}
			ctx, cancel := context.WithCancel(context.Background())
			m.busy = true
			m.cancelAction = cancel
			path := defaultExportPath(m.services.Now(), "csv")
			m.status = "Exporting csv..."
			return m, exportCmd(ctx, m.services.Exporter, "csv", path)
		}
	case syncCompletedMsg:
		m.clearBusy()
		if msg.Err != nil {
			m.status = fmt.Sprintf("Sync failed: %v", msg.Err)
			return m, nil
		}
		if workouts, err := m.services.ReadWorkouts(); err == nil {
			m.workouts = workouts
		}
		m.status = fmt.Sprintf("Sync complete: %d new workouts, %d total", msg.Result.NewWorkouts, msg.Result.TotalWorkouts)
	case reportCompletedMsg:
		m.clearBusy()
		if msg.Err != nil {
			m.status = fmt.Sprintf("Report failed: %v", msg.Err)
			return m, nil
		}
		m.lastReportPath = msg.Path
		m.status = fmt.Sprintf("Report written to %s", msg.Path)
	case exportCompletedMsg:
		m.clearBusy()
		if msg.Err != nil {
			m.status = fmt.Sprintf("Export failed: %v", msg.Err)
			return m, nil
		}
		m.lastExportPath = msg.Path
		m.status = fmt.Sprintf("Export %s written to %s", msg.Format, msg.Path)
	}
	return m, nil
}

func (m *Model) cancelInFlightAction() {
	if m.cancelAction != nil {
		m.cancelAction()
	}
	m.clearBusy()
}

func (m *Model) clearBusy() {
	m.busy = false
	m.cancelAction = nil
}

func (m Model) View() string {
	return render(m)
}

func (m Model) nextTab() tab {
	for i, candidate := range allTabs {
		if candidate == m.activeTab {
			return allTabs[(i+1)%len(allTabs)]
		}
	}
	return dashboardTab
}

func (m Model) previousTab() tab {
	for i, candidate := range allTabs {
		if candidate == m.activeTab {
			return allTabs[(i+len(allTabs)-1)%len(allTabs)]
		}
	}
	return dashboardTab
}

func withServiceDefaults(services Services) Services {
	if services.LoadConfig == nil {
		services.LoadConfig = func() (config.Config, error) { return config.Default(), nil }
	}
	if services.ReadWorkouts == nil {
		services.ReadWorkouts = func() ([]model.Workout, error) { return []model.Workout{}, nil }
	}
	if services.Now == nil {
		services.Now = time.Now
	}
	return services
}
