package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorEmphatic  = lipgloss.Color("#f0f6fc")
	colorMuted     = lipgloss.Color("#8b949e")
	colorAccent    = lipgloss.Color("#58a6ff")
	colorGood      = lipgloss.Color("#3fb950")
	colorBad       = lipgloss.Color("#f85149")
	colorBorder    = lipgloss.Color("#30363d")
	colorTabActive = lipgloss.Color("#1f6feb")
	colorRowSelect = lipgloss.Color("#1f2937")
)

var (
	tabStyle       = lipgloss.NewStyle().Foreground(colorMuted).Padding(0, 2)
	activeTabStyle = lipgloss.NewStyle().Bold(true).Foreground(colorEmphatic).Background(colorTabActive).Padding(0, 2)

	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(colorEmphatic)
	statusStyle   = lipgloss.NewStyle().Foreground(colorAccent)
	helpStyle     = lipgloss.NewStyle().Foreground(colorMuted)
	mutedStyle    = lipgloss.NewStyle().Foreground(colorMuted)
	emphaticStyle = lipgloss.NewStyle().Bold(true).Foreground(colorEmphatic)
	goodStyle     = lipgloss.NewStyle().Bold(true).Foreground(colorGood)
	badStyle      = lipgloss.NewStyle().Bold(true).Foreground(colorBad)
	accentStyle   = lipgloss.NewStyle().Foreground(colorAccent)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2)

	statLabelStyle   = lipgloss.NewStyle().Foreground(colorMuted).Bold(true)
	selectedRowStyle = lipgloss.NewStyle().Background(colorRowSelect).Foreground(colorEmphatic)
)
