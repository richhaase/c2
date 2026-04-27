package tui

import (
	"context"
	"fmt"
	"sort"
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
	actionsTab
)

var allTabs = []tab{dashboardTab, workoutsTab, trendsTab, actionsTab}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

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
	workoutCursor  int
	width          int
	height         int
	status         string
	lastReportPath string
	lastExportPath string
	now            time.Time
	busy           bool
	spinnerFrame   int
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
		m.workouts = sortWorkoutsNewestFirst(workouts)
		if len(m.workouts) == 0 && m.status == "" {
			m.status = "No workouts found. Run `c2 sync` first."
		}
	}
	return m
}

func sortWorkoutsNewestFirst(workouts []model.Workout) []model.Workout {
	out := append([]model.Workout(nil), workouts...)
	sort.Slice(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	return out
}

func (m Model) Init() tea.Cmd { return tickSpinner() }

type spinnerTickMsg struct{}

func tickSpinner() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case spinnerTickMsg:
		if m.busy {
			m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerFrames)
		}
		return m, tickSpinner()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.cancelInFlightAction()
			return m, tea.Quit
		case "right", "tab", "l":
			m.activeTab = m.nextTab()
		case "left", "shift+tab", "h":
			m.activeTab = m.previousTab()
		case "down", "j":
			if m.activeTab == workoutsTab && len(m.workouts) > 0 && m.workoutCursor < len(m.workouts)-1 {
				m.workoutCursor++
			}
		case "up", "k":
			if m.activeTab == workoutsTab && m.workoutCursor > 0 {
				m.workoutCursor--
			}
		case "g", "home":
			if m.activeTab == workoutsTab {
				m.workoutCursor = 0
			}
		case "G", "end":
			if m.activeTab == workoutsTab && len(m.workouts) > 0 {
				m.workoutCursor = len(m.workouts) - 1
			}
		case "s":
			return m.startSync()
		case "r":
			return m.startReport()
		case "e":
			return m.startExport()
		}
	case syncCompletedMsg:
		m.clearBusy()
		if msg.Err != nil {
			m.status = fmt.Sprintf("Sync failed: %v", msg.Err)
			return m, nil
		}
		if workouts, err := m.services.ReadWorkouts(); err == nil {
			m.workouts = sortWorkoutsNewestFirst(workouts)
			if m.workoutCursor >= len(m.workouts) {
				m.workoutCursor = 0
			}
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

func (m Model) startSync() (tea.Model, tea.Cmd) {
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
}

func (m Model) startReport() (tea.Model, tea.Cmd) {
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
}

func (m Model) startExport() (tea.Model, tea.Cmd) {
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
