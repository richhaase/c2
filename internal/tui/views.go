package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/display"
	"github.com/richhaase/c2/internal/model"
	"github.com/richhaase/c2/internal/stats"
)

const (
	defaultViewWidth  = 100
	defaultViewHeight = 32
	minPanelWidth     = 60
	panelBorderWidth  = 2
	panelChromeWidth  = 6
)

func panelOf(totalWidth int, body string) string {
	if totalWidth < 10 {
		totalWidth = 10
	}
	return panelStyle.Width(totalWidth - panelBorderWidth).Render(body)
}

func innerWidth(totalWidth int) int {
	w := totalWidth - panelChromeWidth
	if w < 4 {
		w = 4
	}
	return w
}

func render(m Model) string {
	width, height := viewSize(m)
	body := renderBody(m, width, height)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		renderTabs(m.activeTab),
		"",
		body,
		renderFooter(m, width),
	)
}

func viewSize(m Model) (int, int) {
	w, h := m.width, m.height
	if w < minPanelWidth {
		w = defaultViewWidth
	}
	if h < 16 {
		h = defaultViewHeight
	}
	return w, h
}

func renderTabs(active tab) string {
	parts := make([]string, 0, len(allTabs))
	for _, t := range allTabs {
		label := tabLabel(t)
		if t == active {
			parts = append(parts, activeTabStyle.Render(label))
			continue
		}
		parts = append(parts, tabStyle.Render(label))
	}
	return strings.Join(parts, " ")
}

func tabLabel(t tab) string {
	switch t {
	case workoutsTab:
		return "Workouts"
	case trendsTab:
		return "Trends"
	case actionsTab:
		return "Actions"
	default:
		return "Dashboard"
	}
}

func renderBody(m Model, width, height int) string {
	switch m.activeTab {
	case workoutsTab:
		return renderWorkouts(m, width, height)
	case trendsTab:
		return renderTrends(m, width)
	case actionsTab:
		return renderActions(m, width)
	default:
		return renderDashboard(m, width)
	}
}

func renderDashboard(m Model, width int) string {
	progress := stats.ComputeGoalProgress(m.workouts, m.cfg.Goal, m.now)
	lifetime := totalMeters(m.workouts)
	season := seasonWorkouts(m.workouts, m.cfg.Goal)

	row1Cards := []card{
		{"Lifetime Meters", display.FormatMeters(lifetime), "m"},
		{"Season Meters", display.FormatMeters(totalMeters(season)), "m"},
		{"Sessions", fmt.Sprintf("%d", stats.SessionCount(season)), ""},
	}
	row2Cards := []card{
		{"Avg Pace", averagePace(season), "/500m"},
		{"Avg Heart Rate", averageHeartRate(season), "bpm"},
		{"Weekly Avg", weeklyAvg(progress), weeklyUnit(progress)},
	}

	header := dashboardHeader(m, progress)
	row1 := renderCardRow(row1Cards, width)
	row2 := renderCardRow(row2Cards, width)
	goal := renderGoalProgress(progress, width)

	return lipgloss.JoinVertical(lipgloss.Left, header, "", row1, row2, "", goal)
}

func dashboardHeader(m Model, p stats.GoalProgress) string {
	if p.Target == 0 || m.cfg.Goal.StartDate == "" {
		return mutedStyle.Render("All-time stats. Run `c2 goal set` to track a season.")
	}
	start, errS := config.ParseGoalDate(m.cfg.Goal.StartDate)
	end, errE := config.ParseGoalDate(m.cfg.Goal.EndDate)
	if errS != nil || errE != nil {
		return mutedStyle.Render("All-time stats.")
	}
	title := emphaticStyle.Render(fmt.Sprintf("%d Season", start.Year()))
	rangeStr := fmt.Sprintf("%s — %s · week %d of %d",
		start.Format("Jan 2"), end.Format("Jan 2"), p.WeeksElapsed, p.TotalWeeks)
	scope := mutedStyle.Render("Season-scoped except where labeled Lifetime")
	return lipgloss.JoinVertical(lipgloss.Left,
		title+mutedStyle.Render(" · "+rangeStr),
		scope,
	)
}

func seasonWorkouts(workouts []model.Workout, goal config.GoalConfig) []model.Workout {
	start, errS := config.ParseGoalDate(goal.StartDate)
	end, errE := config.ParseGoalDate(goal.EndDate)
	if errS != nil || errE != nil {
		return workouts
	}
	return stats.WorkoutsInRange(workouts, start, end.AddDate(0, 0, 1))
}

type card struct {
	label string
	value string
	unit  string
}

func renderCardRow(cards []card, totalWidth int) string {
	if len(cards) == 0 {
		return ""
	}
	gap := 1
	cardTotal := (totalWidth - gap*(len(cards)-1)) / len(cards)
	if cardTotal < 14 {
		cardTotal = 14
	}

	parts := make([]string, 0, 2*len(cards)-1)
	for i, c := range cards {
		if i > 0 {
			parts = append(parts, " ")
		}
		parts = append(parts, renderStatCard(c, cardTotal))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func renderStatCard(c card, totalWidth int) string {
	label := statLabelStyle.Render(strings.ToUpper(c.label))
	valueText := emphaticStyle.Render(c.value)
	if c.unit != "" {
		valueText += " " + mutedStyle.Render(c.unit)
	}
	body := lipgloss.JoinVertical(lipgloss.Left, label, valueText)
	return panelOf(totalWidth, body)
}

func renderGoalProgress(p stats.GoalProgress, width int) string {
	contentW := innerWidth(width)

	if p.Target == 0 {
		body := lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("Goal Progress"),
			"",
			mutedStyle.Render("No goal configured. Run `c2 goal set` to define one."),
		)
		return panelOf(width, body)
	}

	pctText := display.FormatPercent(p.Progress)
	leftRaw := fmt.Sprintf("%sm · %s", display.FormatMeters(p.TotalMeters), pctText)
	leftStyled := goodStyle.Render(leftRaw)
	if !p.OnPace {
		leftStyled = badStyle.Render(leftRaw)
	}
	rightRaw := fmt.Sprintf("%sm", display.FormatMeters(p.Target))
	rightStyled := mutedStyle.Render(rightRaw)
	header := padBetween(leftStyled, lipgloss.Width(leftRaw), rightStyled, lipgloss.Width(rightRaw), contentW)

	weekFrac := 0.0
	if p.TotalWeeks > 0 {
		weekFrac = float64(p.WeeksElapsed) / float64(p.TotalWeeks)
	}
	bar := renderBar(contentW, p.Progress, weekFrac, !p.OnPace)

	gap := math.Abs(p.Progress-weekFrac) * 100
	var status string
	switch {
	case p.OnPace && p.Progress >= weekFrac:
		status = goodStyle.Render(fmt.Sprintf("On pace · %.1f%% ahead", gap))
	case p.OnPace:
		status = goodStyle.Render("On pace")
	default:
		status = badStyle.Render(fmt.Sprintf("Behind pace · %.1f%% behind", gap))
	}
	subtitle := fmt.Sprintf("%s · target %sm/wk · week %d of %d",
		status,
		display.FormatMeters(p.RequiredPace),
		p.WeeksElapsed,
		p.TotalWeeks,
	)

	body := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Goal Progress"),
		"",
		header,
		bar,
		"",
		subtitle,
	)
	return panelOf(width, body)
}

func renderWorkouts(m Model, width, height int) string {
	if len(m.workouts) == 0 {
		body := lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("Workouts"),
			"",
			mutedStyle.Render("No workouts found. Press `s` to sync."),
		)
		return panelOf(width, body)
	}

	listTotal := width / 2
	if listTotal < 50 {
		listTotal = 50
	}
	if listTotal > width-30 {
		listTotal = width - 30
	}
	gap := 1
	detailTotal := width - listTotal - gap

	rowsAvailable := height - 12
	if rowsAvailable < 6 {
		rowsAvailable = 6
	}

	listBody := renderWorkoutList(m, innerWidth(listTotal), rowsAvailable)
	detailBody := renderWorkoutDetail(m.workouts[m.workoutCursor], m.cfg.Display.DateFormat)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		panelOf(listTotal, listBody),
		" ",
		panelOf(detailTotal, detailBody),
	)
}

func renderWorkoutList(m Model, contentW, maxRows int) string {
	header := titleStyle.Render(fmt.Sprintf("Workouts (%d)", len(m.workouts)))

	offset := 0
	if m.workoutCursor >= maxRows {
		offset = m.workoutCursor - maxRows + 1
	}
	end := offset + maxRows
	if end > len(m.workouts) {
		end = len(m.workouts)
	}

	lines := []string{header, ""}
	for i := offset; i < end; i++ {
		w := m.workouts[i]
		parsed, _ := model.ParsedDate(w)
		marker := "  "
		if i == m.workoutCursor {
			marker = "▸ "
		}
		raw := fmt.Sprintf("%s%s  %8sm  %7s  %7s",
			marker,
			display.FormatDate(parsed, m.cfg.Display.DateFormat),
			display.FormatMeters(w.Distance),
			w.TimeFormatted,
			model.Pace500m(w),
		)
		raw = clipRunes(raw, contentW)
		raw = padRight(raw, contentW)
		if i == m.workoutCursor {
			raw = selectedRowStyle.Render(raw)
		}
		lines = append(lines, raw)
	}
	if end < len(m.workouts) {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("  … %d more", len(m.workouts)-end)))
	}
	return strings.Join(lines, "\n")
}

func renderWorkoutDetail(w model.Workout, dateFormat string) string {
	parsed, _ := model.ParsedDate(w)
	lines := []string{
		titleStyle.Render("Detail"),
		"",
		detailRow("Date", display.FormatDate(parsed, dateFormat)),
		detailRow("Distance", display.FormatMeters(w.Distance)+" m"),
		detailRow("Time", w.TimeFormatted),
		detailRow("Pace", model.Pace500m(w)+" /500m"),
	}
	if w.StrokeRate != nil && *w.StrokeRate > 0 {
		lines = append(lines, detailRow("Stroke Rate", fmt.Sprintf("%d spm", *w.StrokeRate)))
	}
	if w.HeartRate != nil && w.HeartRate.Average != nil && *w.HeartRate.Average > 0 {
		lines = append(lines, detailRow("Heart Rate", fmt.Sprintf("%d bpm", *w.HeartRate.Average)))
	}
	if w.DragFactor != nil && *w.DragFactor > 0 {
		lines = append(lines, detailRow("Drag Factor", fmt.Sprintf("%d", *w.DragFactor)))
	}
	if w.CaloriesTotal != nil && *w.CaloriesTotal > 0 {
		lines = append(lines, detailRow("Calories", fmt.Sprintf("%d", *w.CaloriesTotal)))
	}
	if model.IsIntervalWorkout(w) {
		tag := strings.TrimSpace(strings.Trim(display.FormatIntervalTag(w), "[]"))
		lines = append(lines, detailRow("Type", accentStyle.Render(tag)))
	}
	if w.Comments != "" {
		lines = append(lines, "", mutedStyle.Render("Notes"), w.Comments)
	}
	return strings.Join(lines, "\n")
}

func detailRow(label, value string) string {
	return fmt.Sprintf("%s  %s", mutedStyle.Render(fmt.Sprintf("%-12s", label)), value)
}

func renderTrends(m Model, width int) string {
	contentW := innerWidth(width)
	weeks := stats.BuildWeekSummaries(m.workouts, 12, m.now)
	if len(weeks) == 0 {
		body := lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("Weekly Volume"),
			"",
			mutedStyle.Render("No weekly data yet."),
		)
		return panelOf(width, body)
	}

	progress := stats.ComputeGoalProgress(m.workouts, m.cfg.Goal, m.now)
	target := progress.RequiredPace
	if target <= 0 {
		target = trendDefaultTarget(weeks)
	}

	labelW := 8
	metaW := 22
	barW := contentW - labelW - metaW - 3
	if barW < 12 {
		barW = 12
	}

	maxScale := int(math.Max(float64(target)*1.25, 1))
	for _, w := range weeks {
		if w.Meters > maxScale {
			maxScale = int(float64(w.Meters) * 1.05)
		}
	}
	targetMarker := float64(target) / float64(maxScale)

	rows := make([]string, 0, len(weeks))
	for _, w := range weeks {
		filledFrac := float64(w.Meters) / float64(maxScale)
		bar := renderBar(barW, filledFrac, targetMarker, w.Meters < target)
		label := mutedStyle.Render(padRight(w.WeekStart.Format("Jan 02"), labelW))
		meta := mutedStyle.Render(padLeft(fmt.Sprintf("%sm · %d sess", display.FormatMeters(w.Meters), w.Sessions), metaW))
		rows = append(rows, fmt.Sprintf("%s %s  %s", label, bar, meta))
	}

	headerLeft := titleStyle.Render("Weekly Volume")
	headerRight := mutedStyle.Render(fmt.Sprintf("Target: %sm/wk", display.FormatMeters(target)))
	header := padBetween(headerLeft, lipgloss.Width("Weekly Volume"), headerRight, lipgloss.Width(fmt.Sprintf("Target: %sm/wk", display.FormatMeters(target))), contentW)

	body := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		strings.Join(rows, "\n"),
	)
	return panelOf(width, body)
}

func trendDefaultTarget(weeks []stats.WeekSummary) int {
	maxMeters := 0
	for _, w := range weeks {
		if w.Meters > maxMeters {
			maxMeters = w.Meters
		}
	}
	if maxMeters == 0 {
		return 20000
	}
	return maxMeters
}

func renderActions(m Model, width int) string {
	rows := []string{
		titleStyle.Render("Actions"),
		"",
		actionRow("s", "Sync", "Pull new workouts from log.concept2.com"),
		actionRow("r", "Report", "Generate an HTML progress report in the current directory"),
		actionRow("e", "Export", "Export all workouts to CSV in the current directory"),
	}
	if m.lastReportPath != "" {
		rows = append(rows, "", mutedStyle.Render("Last report:")+" "+m.lastReportPath)
	}
	if m.lastExportPath != "" {
		rows = append(rows, mutedStyle.Render("Last export:")+" "+m.lastExportPath)
	}
	body := strings.Join(rows, "\n")
	return panelOf(width, body)
}

func actionRow(key, name, desc string) string {
	return fmt.Sprintf("%s  %s  %s",
		accentStyle.Render(fmt.Sprintf(" %s ", key)),
		emphaticStyle.Render(padRight(name, 8)),
		mutedStyle.Render(desc),
	)
}

func renderFooter(m Model, width int) string {
	left := renderStatus(m)
	right := helpForTab(m.activeTab)
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	if leftW+rightW+1 > width {
		return left + "\n" + right
	}
	return padBetween(left, leftW, right, rightW, width)
}

func renderStatus(m Model) string {
	prefix := ""
	if m.busy {
		prefix = accentStyle.Render(spinnerFrames[m.spinnerFrame]) + " "
	}
	text := m.status
	if text == "" {
		text = "Ready"
	}
	return prefix + statusStyle.Render(text)
}

func helpForTab(t tab) string {
	common := "←/→ tabs · s sync · r report · e export · q quit"
	switch t {
	case workoutsTab:
		return helpStyle.Render("↑/↓ select · " + common)
	default:
		return helpStyle.Render(common)
	}
}

func renderBar(width int, filled, marker float64, behind bool) string {
	if width < 4 {
		width = 4
	}
	filled = clamp01(filled)
	marker = clamp01(marker)

	filledCells := int(math.Round(filled * float64(width)))
	if filledCells > width {
		filledCells = width
	}
	markerCell := int(math.Round(marker * float64(width)))
	if markerCell >= width {
		markerCell = width - 1
	}
	if markerCell < 0 {
		markerCell = 0
	}

	fill := goodStyle
	if behind {
		fill = badStyle
	}

	var b strings.Builder
	for i := 0; i < width; i++ {
		switch {
		case i == markerCell:
			b.WriteString(accentStyle.Render("│"))
		case i < filledCells:
			b.WriteString(fill.Render("█"))
		default:
			b.WriteString(mutedStyle.Render("░"))
		}
	}
	return b.String()
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func padBetween(left string, leftW int, right string, rightW int, total int) string {
	gap := total - leftW - rightW
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func padLeft(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", width-w) + s
}

func clipRunes(s string, width int) string {
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	return string(r[:width])
}

func totalMeters(workouts []model.Workout) int {
	total := 0
	for _, w := range workouts {
		total += w.Distance
	}
	return total
}

func averagePace(workouts []model.Workout) string {
	totalSec := 0.0
	count := 0
	for _, w := range workouts {
		s := model.Pace500mSeconds(w)
		if s > 0 {
			totalSec += s
			count++
		}
	}
	if count == 0 {
		return "—"
	}
	return model.FormatSeconds(totalSec / float64(count))
}

func averageHeartRate(workouts []model.Workout) string {
	total := 0
	count := 0
	for _, w := range workouts {
		if w.HeartRate != nil && w.HeartRate.Average != nil && *w.HeartRate.Average > 0 {
			total += *w.HeartRate.Average
			count++
		}
	}
	if count == 0 {
		return "—"
	}
	return fmt.Sprintf("%d", total/count)
}

func weeklyAvg(p stats.GoalProgress) string {
	if p.Target == 0 || p.WeeksElapsed == 0 {
		return "—"
	}
	return display.FormatMeters(p.CurrentAvgPace)
}

func weeklyUnit(p stats.GoalProgress) string {
	if p.Target == 0 {
		return ""
	}
	return "m/wk"
}

func defaultReportPath(now time.Time) string {
	return fmt.Sprintf("c2-report-%s.html", now.Format("20060102-150405"))
}

func defaultExportPath(now time.Time, format string) string {
	return fmt.Sprintf("c2-workouts-%s.%s", now.Format("20060102-150405"), format)
}
