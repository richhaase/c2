package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/richhaase/c2/internal/display"
	"github.com/richhaase/c2/internal/model"
	"github.com/richhaase/c2/internal/stats"
)

func render(m Model) string {
	parts := []string{
		renderTabs(m.activeTab),
		renderBody(m),
		renderStatus(m),
		renderHelp(),
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n")) + "\n"
}

func renderTabs(active tab) string {
	labels := make([]string, 0, len(allTabs))
	for _, t := range allTabs {
		label := tabLabel(t)
		if t == active {
			labels = append(labels, activeTabStyle.Render(label))
			continue
		}
		labels = append(labels, tabStyle.Render(label))
	}
	return strings.Join(labels, "  ")
}

func renderBody(m Model) string {
	switch m.activeTab {
	case workoutsTab:
		return renderWorkouts(m)
	case trendsTab:
		return renderTrends(m)
	case detailTab:
		return renderDetail(m)
	case actionsTab:
		return renderActions(m)
	default:
		return renderDashboard(m)
	}
}

func renderDashboard(m Model) string {
	total := totalMeters(m.workouts)
	return fmt.Sprintf("%s\n\nTotal: %sm\nSessions: %d\nGoal: %sm",
		titleStyle.Render("Dashboard"),
		display.FormatMeters(total),
		stats.SessionCount(m.workouts),
		display.FormatMeters(m.cfg.Goal.TargetMeters),
	)
}

func renderWorkouts(m Model) string {
	lines := []string{titleStyle.Render("Workouts")}
	if len(m.workouts) == 0 {
		return strings.Join(append(lines, "No workouts found."), "\n")
	}

	workouts := append([]model.Workout(nil), m.workouts...)
	sort.Slice(workouts, func(i, j int) bool { return workouts[i].Date > workouts[j].Date })
	limit := min(len(workouts), 8)
	for _, workout := range workouts[:limit] {
		lines = append(lines, display.FormatWorkoutLine(workout, m.cfg.Display.DateFormat))
	}
	return strings.Join(lines, "\n")
}

func renderTrends(m Model) string {
	lines := []string{titleStyle.Render("Trends")}
	summaries := stats.BuildWeekSummaries(m.workouts, 6, m.now)
	if len(summaries) == 0 {
		return strings.Join(append(lines, "No weekly data yet."), "\n")
	}
	for _, summary := range summaries {
		lines = append(lines, fmt.Sprintf("%s  %8sm  %d sessions",
			display.FormatDate(summary.WeekStart, m.cfg.Display.DateFormat),
			display.FormatMeters(summary.Meters),
			summary.Sessions,
		))
	}
	return strings.Join(lines, "\n")
}

func renderDetail(m Model) string {
	lines := []string{titleStyle.Render("Detail")}
	if len(m.workouts) == 0 {
		return strings.Join(append(lines, "No workout selected."), "\n")
	}
	workout := latestWorkout(m.workouts)
	lines = append(lines,
		fmt.Sprintf("Latest: %s", workout.Date),
		fmt.Sprintf("Distance: %sm", display.FormatMeters(workout.Distance)),
		fmt.Sprintf("Time: %s", workout.TimeFormatted),
		fmt.Sprintf("Pace: %s/500m", model.Pace500m(workout)),
	)
	if workout.Comments != "" {
		lines = append(lines, fmt.Sprintf("Notes: %s", workout.Comments))
	}
	return strings.Join(lines, "\n")
}

func renderActions(m Model) string {
	lines := []string{
		titleStyle.Render("Actions"),
		"s  Sync from Concept2",
		"r  Generate HTML report",
		"e  Export workouts as CSV",
	}
	if m.lastReportPath != "" {
		lines = append(lines, fmt.Sprintf("Last report: %s", m.lastReportPath))
	}
	if m.lastExportPath != "" {
		lines = append(lines, fmt.Sprintf("Last export: %s", m.lastExportPath))
	}
	return strings.Join(lines, "\n")
}

func renderStatus(m Model) string {
	if m.status == "" {
		return statusStyle.Render("Ready")
	}
	return statusStyle.Render(m.status)
}

func renderHelp() string {
	return helpStyle.Render("left/right switch tabs  s sync  r report  e export  q quit")
}

func tabLabel(t tab) string {
	switch t {
	case workoutsTab:
		return "Workouts"
	case trendsTab:
		return "Trends"
	case detailTab:
		return "Detail"
	case actionsTab:
		return "Actions"
	default:
		return "Dashboard"
	}
}

func totalMeters(workouts []model.Workout) int {
	total := 0
	for _, workout := range workouts {
		total += workout.Distance
	}
	return total
}

func latestWorkout(workouts []model.Workout) model.Workout {
	latest := workouts[0]
	for _, workout := range workouts[1:] {
		if workout.Date > latest.Date {
			latest = workout
		}
	}
	return latest
}

func defaultReportPath(now time.Time) string {
	return fmt.Sprintf("c2-report-%s.html", now.Format("20060102-150405"))
}

func defaultExportPath(now time.Time, format string) string {
	return fmt.Sprintf("c2-workouts-%s.%s", now.Format("20060102-150405"), format)
}
