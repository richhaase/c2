package report

import (
	"bytes"
	"fmt"
	"html/template"
	"math"
	"sort"
	"time"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/display"
	"github.com/richhaase/c2/internal/model"
	"github.com/richhaase/c2/internal/stats"
)

func HTML(workouts []model.Workout, goalCfg config.GoalConfig, weeks int, now time.Time) (string, error) {
	if weeks < 1 {
		return "", fmt.Errorf("weeks must be a positive integer")
	}
	if now.IsZero() {
		now = time.Now()
	}

	goal := stats.ComputeGoalProgress(workouts, goalCfg, now)
	summaries := stats.BuildWeekSummaries(workouts, weeks, now)
	data := reportData{
		GeneratedYear: now.Year(),
		GeneratedDate: fullDate(now),
		Goal:          newGoalView(goal),
		Sessions:      stats.SessionCount(workouts),
		AvgPace:       fmtPace(avgPaceForWorkouts(workouts)),
		AvgHR:         avgHRForWorkouts(workouts),
		Weekly:        newWeeklyViews(summaries, goal.RequiredPace),
		Trends:        newTrendViews(summaries),
		Recent:        newRecentViews(workouts, 10),
		Projection:    newProjectionView(goal, workouts),
	}

	var buf bytes.Buffer
	if err := reportTemplate.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type reportData struct {
	GeneratedYear int
	GeneratedDate string
	Goal          goalView
	Sessions      int
	AvgPace       string
	AvgHR         string
	Weekly        weeklyView
	Trends        trendView
	Recent        []recentWorkoutView
	Projection    projectionView
}

type goalView struct {
	Target             string
	TotalMeters        string
	ProgressPct        string
	OnPacePct          string
	PaceClass          string
	DiffClass          string
	DiffLabel          string
	CurrentWeeklyAvg   string
	RequiredWeeklyPace string
	WeeksElapsed       int
	TotalWeeks         int
	QuarterMarks       []string
}

type weeklyView struct {
	RequiredPace string
	TargetPct    string
	Rows         []weeklyRowView
}

type weeklyRowView struct {
	Week       string
	Meters     string
	Sessions   int
	BarClass   string
	BarPct     string
	CurrentTag string
}

type trendView struct {
	Rows []trendRowView
	Note string
}

type trendRowView struct {
	Week     string
	Volume   string
	AvgPace  string
	AvgSPM   string
	AvgHR    string
	BestVol  bool
	BestPace bool
}

type recentWorkoutView struct {
	Date           string
	Distance       string
	Pace           string
	SPM            string
	HR             string
	Comments       string
	Annotation     string
	Muted          bool
	HardAnnotation bool
	HardPace       bool
	HardHR         bool
}

type projectionView struct {
	CurrentClass       string
	ProjectedMeters    string
	ProjectedPct       string
	ShortfallLabel     string
	CurrentWeeklyAvg   string
	RemainingWeeks     int
	TotalMeters        string
	Target             string
	RequiredWeeklyPace string
	RemainingMeters    string
	SessionsPerWeek    string
	AvgSessionDistance string
	IncreaseLabel      string
}

func newGoalView(goal stats.GoalProgress) goalView {
	progressPct := goal.Progress * 100
	onPacePct := 0.0
	if goal.TotalWeeks > 0 {
		onPacePct = (float64(goal.WeeksElapsed) / float64(goal.TotalWeeks)) * 100
	}
	diff := progressPct - onPacePct
	diffClass := "green"
	diffLabel := fmt.Sprintf("%.1f%% ahead of pace", diff)
	if diff < 0 {
		diffClass = "red"
		diffLabel = fmt.Sprintf("%.1f%% behind pace", math.Abs(diff))
	}

	paceClass := "red"
	if goal.OnPace {
		paceClass = "green"
	}

	q := goal.Target / 4
	return goalView{
		Target:             display.FormatMeters(goal.Target),
		TotalMeters:        display.FormatMeters(goal.TotalMeters),
		ProgressPct:        percentNumber(progressPct),
		OnPacePct:          percentNumber(onPacePct),
		PaceClass:          paceClass,
		DiffClass:          diffClass,
		DiffLabel:          diffLabel,
		CurrentWeeklyAvg:   display.FormatMeters(goal.CurrentAvgPace),
		RequiredWeeklyPace: display.FormatMeters(goal.RequiredPace),
		WeeksElapsed:       goal.WeeksElapsed,
		TotalWeeks:         goal.TotalWeeks,
		QuarterMarks: []string{
			fmtShortNum(0),
			fmtShortNum(q),
			fmtShortNum(q * 2),
			fmtShortNum(q * 3),
			fmtShortNum(goal.Target),
		},
	}
}

func newWeeklyViews(summaries []stats.WeekSummary, requiredPace int) weeklyView {
	maxMeters := float64(requiredPace) * 1.25
	for _, summary := range summaries {
		if float64(summary.Meters) > maxMeters {
			maxMeters = float64(summary.Meters)
		}
	}
	if maxMeters <= 0 {
		maxMeters = 1
	}

	rows := make([]weeklyRowView, 0, len(summaries))
	lastIdx := len(summaries) - 1
	for i, summary := range summaries {
		barClass := "behind"
		if summary.Meters >= requiredPace {
			barClass = "on-pace"
		}
		currentTag := ""
		if i == lastIdx {
			currentTag = "(now)"
		}
		rows = append(rows, weeklyRowView{
			Week:       shortDate(summary.WeekStart),
			Meters:     display.FormatMeters(summary.Meters),
			Sessions:   summary.Sessions,
			BarClass:   barClass,
			BarPct:     percentNumber((float64(summary.Meters) / maxMeters) * 100),
			CurrentTag: currentTag,
		})
	}

	return weeklyView{
		RequiredPace: display.FormatMeters(requiredPace),
		TargetPct:    percentNumber((float64(requiredPace) / maxMeters) * 100),
		Rows:         rows,
	}
}

func newTrendViews(summaries []stats.WeekSummary) trendView {
	bestVolume := 0
	bestPace := math.Inf(1)
	for _, summary := range summaries {
		if summary.Meters > bestVolume {
			bestVolume = summary.Meters
		}
		if summary.PaceCount > 0 {
			avg := summary.PaceSum / float64(summary.PaceCount)
			if avg < bestPace {
				bestPace = avg
			}
		}
	}

	rows := make([]trendRowView, 0, len(summaries))
	for _, summary := range summaries {
		avgPace := 0.0
		if summary.PaceCount > 0 {
			avgPace = summary.PaceSum / float64(summary.PaceCount)
		}

		rows = append(rows, trendRowView{
			Week:     shortDate(summary.WeekStart),
			Volume:   display.FormatMeters(summary.Meters),
			AvgPace:  fmtPace(avgPace),
			AvgSPM:   avgString(summary.SPMSum, summary.SPMCount, 1),
			AvgHR:    avgString(summary.HRSum, summary.HRCount, 0),
			BestVol:  summary.Meters == bestVolume && summary.Meters > 0,
			BestPace: avgPace == bestPace && avgPace > 0,
		})
	}

	return trendView{
		Rows: rows,
		Note: paceTrendNote(summaries),
	}
}

func newRecentViews(workouts []model.Workout, count int) []recentWorkoutView {
	sorted := append([]model.Workout(nil), workouts...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Date > sorted[j].Date
	})
	if len(sorted) > count {
		sorted = sorted[:count]
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Date < sorted[j].Date
	})

	dayCounts := make(map[string]int)
	for _, workout := range sorted {
		dayCounts[model.CalendarDay(workout)]++
	}

	rows := make([]recentWorkoutView, 0, len(sorted))
	for _, workout := range sorted {
		parsed, err := model.ParsedDate(workout)
		date := model.CalendarDay(workout)
		if err == nil {
			date = shortDate(parsed)
		}

		annotation := ""
		muted := false
		hardAnnotation := false
		hardPace := false
		hardHR := false
		if dayCounts[model.CalendarDay(workout)] > 1 {
			isShort := workout.Distance <= 1500
			isHard := model.Pace500mSeconds(workout) > 0 && model.Pace500mSeconds(workout) < 160
			if isShort && !isHard {
				if sameDayIndex(sorted, workout) == sameDayLastIndex(sorted, workout) {
					annotation = "cooldown"
				} else {
					annotation = "warmup"
				}
				muted = true
			} else if isHard {
				annotation = "hard"
				hardAnnotation = true
				hardPace = true
				if workout.HeartRate != nil && workout.HeartRate.Average != nil && *workout.HeartRate.Average >= 135 {
					hardHR = true
				}
			}
		}

		rows = append(rows, recentWorkoutView{
			Date:           date,
			Distance:       display.FormatMeters(workout.Distance),
			Pace:           model.Pace500m(workout),
			SPM:            intPtrString(workout.StrokeRate),
			HR:             heartRateString(workout),
			Comments:       workout.Comments,
			Annotation:     annotation,
			Muted:          muted,
			HardAnnotation: hardAnnotation,
			HardPace:       hardPace,
			HardHR:         hardHR,
		})
	}
	return rows
}

func sameDayIndex(workouts []model.Workout, workout model.Workout) int {
	day := model.CalendarDay(workout)
	idx := 0
	for _, candidate := range workouts {
		if model.CalendarDay(candidate) != day {
			continue
		}
		if candidate.ID == workout.ID && candidate.Date == workout.Date {
			return idx
		}
		idx++
	}
	return -1
}

func sameDayLastIndex(workouts []model.Workout, workout model.Workout) int {
	day := model.CalendarDay(workout)
	last := -1
	for _, candidate := range workouts {
		if model.CalendarDay(candidate) == day {
			last++
		}
	}
	return last
}

func newProjectionView(goal stats.GoalProgress, workouts []model.Workout) projectionView {
	projected := goal.CurrentAvgPace*goal.RemainingWeeks + goal.TotalMeters
	projectedPct := 0.0
	if goal.Target > 0 {
		projectedPct = (float64(projected) / float64(goal.Target)) * 100
	}
	shortfall := goal.Target - projected
	shortfallLabel := "On track to exceed goal"
	if shortfall > 0 {
		shortfallLabel = fmt.Sprintf("%sm short of goal", display.FormatMeters(shortfall))
	}
	currentClass := "red"
	if projected >= goal.Target {
		currentClass = "green"
	}

	avgSessionDistance := 5000
	if len(workouts) > 0 {
		total := 0
		for _, workout := range workouts {
			total += workout.Distance
		}
		avgSessionDistance = int(math.Round(float64(total) / float64(len(workouts))))
	}
	sessionsPerWeek := "-"
	if avgSessionDistance > 0 {
		sessionsPerWeek = fmt.Sprintf("%.1f", float64(goal.RequiredPace)/float64(avgSessionDistance))
	}

	increaseLabel := "Pace is sufficient"
	if goal.CurrentAvgPace > 0 && goal.RequiredPace > goal.CurrentAvgPace {
		increase := ((float64(goal.RequiredPace) - float64(goal.CurrentAvgPace)) / float64(goal.CurrentAvgPace)) * 100
		increaseLabel = fmt.Sprintf("+%.0f%% increase needed", increase)
	}

	return projectionView{
		CurrentClass:       currentClass,
		ProjectedMeters:    display.FormatMeters(roundToNearestThousand(projected)),
		ProjectedPct:       percentNumber(projectedPct),
		ShortfallLabel:     shortfallLabel,
		CurrentWeeklyAvg:   display.FormatMeters(goal.CurrentAvgPace),
		RemainingWeeks:     goal.RemainingWeeks,
		TotalMeters:        display.FormatMeters(goal.TotalMeters),
		Target:             display.FormatMeters(goal.Target),
		RequiredWeeklyPace: display.FormatMeters(goal.RequiredPace),
		RemainingMeters:    display.FormatMeters(goal.RemainingMeters),
		SessionsPerWeek:    sessionsPerWeek,
		AvgSessionDistance: display.FormatMeters(avgSessionDistance),
		IncreaseLabel:      increaseLabel,
	}
}

func avgPaceForWorkouts(workouts []model.Workout) float64 {
	sum := 0.0
	count := 0
	for _, workout := range workouts {
		pace := model.Pace500mSeconds(workout)
		if pace > 0 {
			sum += pace
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func avgHRForWorkouts(workouts []model.Workout) string {
	sum := 0
	count := 0
	for _, workout := range workouts {
		if workout.HeartRate != nil && workout.HeartRate.Average != nil && *workout.HeartRate.Average > 0 {
			sum += *workout.HeartRate.Average
			count++
		}
	}
	if count == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", int(math.Round(float64(sum)/float64(count))))
}

func fmtPace(secs float64) string {
	if secs == 0 {
		return "-"
	}
	mins := int(secs / 60)
	rem := secs - float64(mins*60)
	return fmt.Sprintf("%d:%04.1f", mins, rem)
}

func avgString(sum int, count int, decimals int) string {
	if count == 0 {
		return "-"
	}
	avg := float64(sum) / float64(count)
	if decimals == 0 {
		return fmt.Sprintf("%.0f", avg)
	}
	return fmt.Sprintf("%.*f", decimals, avg)
}

func paceTrendNote(summaries []stats.WeekSummary) string {
	var first *stats.WeekSummary
	var last *stats.WeekSummary
	for i := range summaries {
		if summaries[i].PaceCount > 0 {
			if first == nil {
				first = &summaries[i]
			}
			last = &summaries[i]
		}
	}
	if first == nil || last == nil || first == last {
		return ""
	}

	firstPace := first.PaceSum / float64(first.PaceCount)
	lastPace := last.PaceSum / float64(last.PaceCount)
	direction := "slower"
	if lastPace < firstPace {
		direction = "faster"
	}
	return fmt.Sprintf("Pace trending %s: %s to %s over %d weeks", direction, fmtPace(firstPace), fmtPace(lastPace), len(summaries))
}

func intPtrString(value *int) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *value)
}

func heartRateString(workout model.Workout) string {
	if workout.HeartRate == nil || workout.HeartRate.Average == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *workout.HeartRate.Average)
}

func shortDate(day time.Time) string {
	return day.Format("Jan 2")
}

func fullDate(day time.Time) string {
	return day.Format("Jan 2, 2006")
}

func percentNumber(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0.0"
	}
	if v < 0 {
		return "0.0"
	}
	return fmt.Sprintf("%.1f", v)
}

func fmtShortNum(n int) string {
	if n == 0 {
		return "0"
	}
	if n >= 1_000_000 && n%1_000_000 == 0 {
		return fmt.Sprintf("%dM", n/1_000_000)
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%dK", int(math.Round(float64(n)/1000)))
	}
	return fmt.Sprintf("%d", n)
}

func roundToNearestThousand(n int) int {
	return int(math.Round(float64(n)/1000) * 1000)
}

var reportTemplate = template.Must(template.New("report").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Rowing Progress - {{.GeneratedYear}}</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { background: #0d1117; color: #c9d1d9; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; line-height: 1.6; padding: 24px; max-width: 960px; margin: 0 auto; }
  h1 { color: #f0f6fc; font-size: 28px; font-weight: 700; }
  h2 { color: #f0f6fc; font-size: 20px; font-weight: 600; margin-bottom: 16px; }
  h3 { font-size: 14px; font-weight: 600; margin-bottom: 8px; }
  header { margin-bottom: 32px; }
  table { width: 100%; border-collapse: collapse; font-size: 13px; }
  th { text-align: left; color: #8b949e; font-weight: 600; padding: 8px 10px; border-bottom: 1px solid #30363d; font-size: 12px; text-transform: uppercase; }
  td { padding: 8px 10px; border-bottom: 1px solid #21262d; font-variant-numeric: tabular-nums; vertical-align: top; }
  th.r, td.r { text-align: right; }
  .subtitle, .date, .muted { color: #8b949e; }
  .green { color: #3fb950; }
  .red { color: #f85149; }
  .blue { color: #58a6ff; }
  .stats-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin-bottom: 32px; }
  .stat-card, .section { background: #161b22; border: 1px solid #30363d; border-radius: 8px; }
  .stat-card { padding: 16px; }
  .stat-card .label { color: #8b949e; font-size: 12px; text-transform: uppercase; margin-bottom: 4px; }
  .stat-card .value { color: #f0f6fc; font-size: 24px; font-weight: 700; }
  .stat-card .unit { color: #8b949e; font-size: 13px; font-weight: 400; }
  .section { padding: 20px; margin-bottom: 24px; }
  .progress-container { position: relative; background: #21262d; border-radius: 6px; height: 32px; margin: 16px 0 8px; overflow: visible; }
  .progress-fill { height: 100%; border-radius: 6px; background: #f85149; min-width: 2px; }
  .progress-marker { position: absolute; top: -6px; height: 44px; width: 2px; background: #3fb950; }
  .progress-marker-label { position: absolute; top: -22px; transform: translateX(-50%); font-size: 11px; color: #3fb950; white-space: nowrap; font-weight: 600; }
  .progress-label-row { display: flex; justify-content: space-between; font-size: 12px; color: #8b949e; margin-top: 4px; }
  .week-row { display: flex; align-items: center; margin-bottom: 8px; font-size: 13px; }
  .week-label { width: 70px; flex-shrink: 0; color: #8b949e; font-size: 12px; text-align: right; padding-right: 10px; }
  .week-bar-container { flex: 1; position: relative; height: 24px; background: #21262d; border-radius: 4px; overflow: visible; }
  .week-bar { height: 100%; border-radius: 4px; min-width: 2px; }
  .week-bar.on-pace { background: #238636; }
  .week-bar.behind { background: #8b2a2d; }
  .week-target-line { position: absolute; top: -2px; height: 28px; width: 0; border-left: 2px dashed #58a6ff; opacity: 0.7; }
  .week-meta { width: 140px; flex-shrink: 0; text-align: right; font-size: 12px; color: #8b949e; padding-left: 8px; }
  .projection-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
  .projection-card { background: #21262d; border-radius: 6px; padding: 16px; }
  .big-num { font-size: 28px; font-weight: 700; margin-bottom: 4px; }
  .detail { font-size: 12px; color: #8b949e; line-height: 1.8; }
  @media (max-width: 640px) { .stats-grid { grid-template-columns: repeat(2, 1fr); } .projection-grid { grid-template-columns: 1fr; } body { padding: 16px; } }
</style>
</head>
<body>
<header>
  <h1>Rowing Progress</h1>
  <div class="subtitle">{{.GeneratedYear}} Season - {{.Goal.Target}}m Goal</div>
  <div class="date">{{.GeneratedDate}}</div>
</header>

<div class="stats-grid">
  <div class="stat-card"><div class="label">Total Meters</div><div class="value">{{.Goal.TotalMeters}} <span class="unit">m</span></div></div>
  <div class="stat-card"><div class="label">Sessions</div><div class="value">{{.Sessions}}</div></div>
  <div class="stat-card"><div class="label">Avg Pace</div><div class="value">{{.AvgPace}} <span class="unit">/500m</span></div></div>
  <div class="stat-card"><div class="label">Avg Heart Rate</div><div class="value">{{.AvgHR}} <span class="unit">bpm</span></div></div>
  <div class="stat-card"><div class="label">Current Weekly Avg</div><div class="value {{.Goal.PaceClass}}">{{.Goal.CurrentWeeklyAvg}} <span class="unit">m/wk</span></div></div>
  <div class="stat-card"><div class="label">Required Weekly Pace</div><div class="value blue">{{.Goal.RequiredWeeklyPace}} <span class="unit">m/wk</span></div></div>
</div>

<div class="section">
  <h2>Goal Progress</h2>
  <div style="display:flex; justify-content:space-between; font-size:13px; margin-bottom:4px;">
    <span class="{{.Goal.DiffClass}}" style="font-weight:600;">{{.Goal.TotalMeters}}m - {{.Goal.ProgressPct}}%</span>
    <span class="muted">{{.Goal.Target}}m</span>
  </div>
  <div class="progress-container">
    <div class="progress-fill" style="width: {{.Goal.ProgressPct}}%;"></div>
    <div class="progress-marker" style="left: {{.Goal.OnPacePct}}%;"><div class="progress-marker-label">On Pace ({{.Goal.OnPacePct}}%)</div></div>
  </div>
  <div class="progress-label-row">{{range .Goal.QuarterMarks}}<span>{{.}}</span>{{end}}</div>
  <div style="margin-top: 12px; font-size: 13px;">
    <span class="{{.Goal.DiffClass}}">&#9632;</span> Actual &nbsp;&nbsp;
    <span class="green">|</span> On-pace target (week {{.Goal.WeeksElapsed}} of {{.Goal.TotalWeeks}})
    - <span class="{{.Goal.DiffClass}}" style="font-weight:600;">{{.Goal.DiffLabel}}</span>
  </div>
</div>

<div class="section">
  <h2>Weekly Volume</h2>
  <div style="font-size:11px; color:#8b949e; margin-bottom:12px; text-align:right;">Target: {{.Weekly.RequiredPace}} m/wk</div>
  {{range .Weekly.Rows}}
  <div class="week-row">
    <div class="week-label">{{.Week}}</div>
    <div class="week-bar-container">
      <div class="week-bar {{.BarClass}}" style="width: {{.BarPct}}%;"></div>
      <div class="week-target-line" style="left: {{$.Weekly.TargetPct}}%;"></div>
    </div>
    <div class="week-meta">{{.Meters}} m &middot; {{.Sessions}} sess <span class="blue">{{.CurrentTag}}</span></div>
  </div>
  {{end}}
</div>

<div class="section">
  <h2>Weekly Trends</h2>
  <table>
    <thead><tr><th>Week</th><th class="r">Volume</th><th class="r">Avg Pace /500m</th><th class="r">Avg SPM</th><th class="r">Avg HR</th></tr></thead>
    <tbody>
    {{range .Trends.Rows}}
      <tr>
        <td>{{.Week}}</td>
        <td class="r {{if .BestVol}}green{{end}}">{{.Volume}}m</td>
        <td class="r {{if .BestPace}}green{{end}}">{{.AvgPace}}</td>
        <td class="r">{{.AvgSPM}}</td>
        <td class="r">{{.AvgHR}}</td>
      </tr>
    {{end}}
    </tbody>
  </table>
  {{if .Trends.Note}}<div class="muted" style="margin-top:12px; font-size:12px;">{{.Trends.Note}}</div>{{end}}
</div>

<div class="section">
  <h2>Recent Workouts</h2>
  <table>
    <thead><tr><th>Date</th><th class="r">Distance</th><th class="r">Pace /500m</th><th class="r">SPM</th><th class="r">HR</th><th>Comments</th></tr></thead>
    <tbody>
    {{range .Recent}}
      <tr{{if .Muted}} style="color:#8b949e;"{{end}}><td>{{.Date}}{{if .Annotation}} <span style="font-size:10px;{{if .HardAnnotation}} color:#3fb950;{{end}}">({{.Annotation}})</span>{{end}}</td><td class="r">{{.Distance}}m</td><td class="r"{{if .HardPace}} style="color:#3fb950;"{{end}}>{{.Pace}}</td><td class="r">{{.SPM}}</td><td class="r"{{if .HardHR}} style="color:#f85149;"{{end}}>{{.HR}}</td><td>{{.Comments}}</td></tr>
    {{end}}
    </tbody>
  </table>
</div>

<div class="section">
  <h2>Year-End Projection</h2>
  <div class="projection-grid">
    <div class="projection-card">
      <h3 class="{{.Projection.CurrentClass}}">At Current Pace</h3>
      <div class="big-num {{.Projection.CurrentClass}}">~{{.Projection.ProjectedMeters}}m</div>
      <div class="detail">{{.Projection.CurrentWeeklyAvg}} m/wk &times; {{.Projection.RemainingWeeks}} remaining + {{.Projection.TotalMeters}}<br>{{.Projection.ShortfallLabel}}<br><span class="{{.Projection.CurrentClass}}">{{.Projection.ProjectedPct}}% of target</span></div>
    </div>
    <div class="projection-card">
      <h3 class="green">To Hit {{.Projection.Target}}m</h3>
      <div class="big-num green">{{.Projection.RequiredWeeklyPace}} <span style="font-size:16px; font-weight:400;">m/wk</span></div>
      <div class="detail">{{.Projection.RemainingMeters}}m remaining over {{.Projection.RemainingWeeks}} weeks<br>~{{.Projection.SessionsPerWeek}} sessions of {{.Projection.AvgSessionDistance}}m per week<br><span class="green">{{.Projection.IncreaseLabel}}</span></div>
    </div>
  </div>
</div>

<div style="text-align: center; color: #484f58; font-size: 12px; margin-top: 32px; padding-bottom: 16px;">
  Generated by c2 &middot; Data from Concept2 Logbook &middot; {{.GeneratedDate}}
</div>
</body>
</html>`))
