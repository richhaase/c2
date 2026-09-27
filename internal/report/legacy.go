package report

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/richhaase/c2/internal/display"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/stats"
)

func formatNumber(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func reportShortDate(d time.Time) string {
	return d.Format("Jan 2")
}

func reportFullDate(d time.Time) string {
	return d.Format("Jan 2, 2006")
}

func fmtPace(secs float64) string {
	if secs == 0 {
		return "-"
	}
	return models.FormatSeconds(secs)
}

func buildStatsCards(goal stats.GoalProgress, sessions int, avgPace float64, avgHR int) string {
	paceClass := "red"
	if goal.OnPace {
		paceClass = "green"
	}
	hr := "-"
	if avgHR > 0 {
		hr = strconv.Itoa(avgHR)
	}
	requiredPace := display.FormatMeters(goal.RequiredPace) + ` <span class="unit">m/wk</span>`
	if goal.RemainingWeeks == 0 {
		requiredPace = "Goal window ended"
	}
	return `<div class="stats-grid">
  <div class="stat-card">
    <div class="label">Total Meters</div>
    <div class="value">` + display.FormatMeters(goal.TotalMeters) + ` <span class="unit">m</span></div>
  </div>
  <div class="stat-card">
    <div class="label">Sessions</div>
    <div class="value">` + strconv.Itoa(sessions) + `</div>
  </div>
  <div class="stat-card">
    <div class="label">Avg Pace</div>
    <div class="value">` + fmtPace(avgPace) + ` <span class="unit">/500m</span></div>
  </div>
  <div class="stat-card">
    <div class="label">Avg Heart Rate</div>
    <div class="value">` + hr + ` <span class="unit">bpm</span></div>
  </div>
  <div class="stat-card">
    <div class="label">Current Weekly Avg</div>
    <div class="value ` + paceClass + `">` + display.FormatMeters(goal.CurrentAvgPace) + ` <span class="unit">m/wk</span></div>
  </div>
  <div class="stat-card">
    <div class="label">Required Weekly Pace</div>
    <div class="value blue">` + requiredPace + `</div>
  </div>
</div>`
}

func fmtShortNum(n float64) string {
	if n == 0 {
		return "0"
	}
	if n >= 1_000_000 && math.Mod(n, 1_000_000) == 0 {
		return formatNumber(n/1_000_000) + "M"
	}
	if n >= 1_000_000 {
		return display.ToFixed(n/1_000_000, 1) + "M"
	}
	if n >= 1000 {
		return formatNumber(roundHalfUp(n/1000)) + "K"
	}
	return formatNumber(n)
}

func buildGoalProgress(goal stats.GoalProgress) string {
	pct := display.ToFixed(goal.Progress*100, 1)
	onPacePct := display.ToFixed(goal.ElapsedFraction*100, 1)
	onPaceVal, _ := strconv.ParseFloat(onPacePct, 64)
	diff := display.ToFixed(goal.Progress*100-onPaceVal, 1)
	diffVal, _ := strconv.ParseFloat(diff, 64)

	diffLabel := diff + "% ahead of pace"
	diffClass := "green"
	if diffVal < 0 {
		diffLabel = display.ToFixed(math.Abs(diffVal), 1) + "% behind pace"
		diffClass = "red"
	}

	q := float64(goal.Target) / 4

	return `<div class="section">
  <h2>Goal Progress</h2>
  <div style="display:flex; justify-content:space-between; font-size:13px; margin-bottom:4px;">
    <span class="` + diffClass + `" style="font-weight:600;">` + display.FormatMeters(goal.TotalMeters) + `m &mdash; ` + pct + `%</span>
    <span class="muted">` + display.FormatMeters(goal.Target) + `m</span>
  </div>
  <div class="progress-container">
    <div class="progress-fill" style="width: ` + pct + `%;"></div>
    <div class="progress-marker" style="left: ` + onPacePct + `%;">
      <div class="progress-marker-label">On Pace (` + onPacePct + `%)</div>
    </div>
  </div>
  <div class="progress-label-row">
    <span>` + fmtShortNum(0) + `</span>
    <span>` + fmtShortNum(q) + `</span>
    <span>` + fmtShortNum(q*2) + `</span>
    <span>` + fmtShortNum(q*3) + `</span>
    <span>` + fmtShortNum(float64(goal.Target)) + `</span>
  </div>
  <div style="margin-top: 12px; font-size: 13px;">
    <span class="` + diffClass + `">&#9632;</span> Actual &nbsp;&nbsp;
    <span class="green">|</span> Target by today (` + onPacePct + `% of goal window elapsed)
    &mdash; <span class="` + diffClass + `" style="font-weight:600;">` + diffLabel + `</span>
  </div>
</div>`
}

func buildWeeklyVolume(summaries []stats.WeekSummary, requiredPace int) string {
	maxM := float64(requiredPace) * 1.25
	for _, w := range summaries {
		if float64(w.Meters) > maxM {
			maxM = float64(w.Meters)
		}
	}
	scale := maxM
	if maxM <= 0 {
		scale = 1
	}
	targetPct := display.ToFixed(float64(requiredPace)/scale*100, 1)
	lastIdx := len(summaries) - 1

	rows := make([]string, 0, len(summaries))
	for i, ws := range summaries {
		pct := display.ToFixed(float64(ws.Meters)/scale*100, 1)
		barClass := "behind"
		if ws.Meters >= requiredPace {
			barClass = "on-pace"
		}
		labelStyle := ""
		nowTag := ""
		if i == lastIdx {
			labelStyle = ` style="color:#c9d1d9; font-weight:600;"`
			nowTag = ` <span style="color:#58a6ff; font-size:10px;">(now)</span>`
		}
		rows = append(rows, `  <div class="week-row">
    <div class="week-label"`+labelStyle+`>`+reportShortDate(ws.WeekStart)+`</div>
    <div class="week-bar-container">
      <div class="week-bar `+barClass+`" style="width: `+pct+`%;"></div>
      <div class="week-target-line" style="left: `+targetPct+`%;"></div>
    </div>
    <div class="week-meta"><span class="meters">`+display.FormatMeters(ws.Meters)+`</span> m &middot; `+strconv.Itoa(ws.Sessions)+` sess`+nowTag+`</div>
  </div>`)
	}

	return `<div class="section">
  <h2>Weekly Volume</h2>
  <div class="target-legend">
    <span class="target-legend-line"></span>
    <span>Target: ` + display.FormatMeters(requiredPace) + ` m/wk</span>
  </div>

` + strings.Join(rows, "\n\n") + `
</div>`
}

func buildWeeklyTrends(summaries []stats.WeekSummary) string {
	bestVolume := 0
	bestPace := math.Inf(1)
	for _, ws := range summaries {
		if ws.Meters > bestVolume {
			bestVolume = ws.Meters
		}
		if ws.PaceCount > 0 {
			avg := ws.PaceSum / float64(ws.PaceCount)
			if avg < bestPace {
				bestPace = avg
			}
		}
	}

	rows := make([]string, 0, len(summaries))
	for _, ws := range summaries {
		avgPace := 0.0
		if ws.PaceCount > 0 {
			avgPace = ws.PaceSum / float64(ws.PaceCount)
		}
		avgSPM := "-"
		if ws.SPMCount > 0 {
			avgSPM = display.ToFixed(float64(ws.SPMSum)/float64(ws.SPMCount), 1)
		}
		avgHR := "-"
		if ws.HRCount > 0 {
			avgHR = strconv.Itoa(int(roundHalfUp(float64(ws.HRSum) / float64(ws.HRCount))))
		}

		volStyle := ""
		if ws.Meters == bestVolume && ws.Meters > 0 {
			volStyle = ` style="color:#3fb950;"`
		}
		paceStyle := ""
		if avgPace == bestPace && avgPace > 0 {
			paceStyle = ` style="color:#3fb950;"`
		}
		paceCell := "-"
		if avgPace > 0 {
			paceCell = fmtPace(avgPace)
		}

		rows = append(rows, `      <tr>
        <td>`+reportShortDate(ws.WeekStart)+`</td>
        <td class="r"`+volStyle+`>`+display.FormatMeters(ws.Meters)+`m</td>
        <td class="r"`+paceStyle+`>`+paceCell+`</td>
        <td class="r">`+esc(avgSPM)+`</td>
        <td class="r">`+esc(avgHR)+`</td>
      </tr>`)
	}

	firstIdx, lastIdx := -1, -1
	for i, w := range summaries {
		if w.PaceCount > 0 {
			if firstIdx < 0 {
				firstIdx = i
			}
			lastIdx = i
		}
	}

	trendNote := ""
	if firstIdx >= 0 && lastIdx >= 0 && firstIdx != lastIdx {
		first := summaries[firstIdx]
		last := summaries[lastIdx]
		fp := first.PaceSum / float64(first.PaceCount)
		lp := last.PaceSum / float64(last.PaceCount)
		diff := math.Abs(fp - lp)
		direction := "slower"
		change := "decline"
		colorClass := "red"
		if lp < fp {
			direction = "faster"
			change = "improvement"
			colorClass = "green"
		}
		trendNote = "\n" + `  <div style="margin-top:12px; font-size:12px; color:#8b949e;">
    Pace trending ` + direction + `: <span class="` + colorClass + `">` + fmtPace(fp) + ` &rarr; ` + fmtPace(lp) + `</span> &mdash; ` + formatNumber(roundHalfUp(diff)) + ` seconds ` + change + ` over ` + strconv.Itoa(len(summaries)) + ` weeks
  </div>`
	}

	return `<div class="section">
  <h2>Weekly Trends</h2>
  <table>
    <thead>
      <tr>
        <th>Week</th>
        <th class="r">Volume</th>
        <th class="r">Avg Pace /500m</th>
        <th class="r">Avg SPM</th>
        <th class="r">Avg HR</th>
      </tr>
    </thead>
    <tbody>
` + strings.Join(rows, "\n") + `
    </tbody>
  </table>` + trendNote + `
</div>`
}

func buildRecentWorkouts(workouts []models.Workout, count int) string {
	sorted := sortedByDateDesc(workouts)
	n := count
	if n > len(sorted) {
		n = len(sorted)
	}
	recent := make([]models.Workout, n)
	for i := 0; i < n; i++ {
		recent[i] = sorted[n-1-i]
	}

	dayCounts := make(map[string]int, n)
	for _, w := range recent {
		dayCounts[models.CalendarDay(w)]++
	}
	dayIndex := make([]int, n)
	seen := make(map[string]int, n)
	for i, w := range recent {
		day := models.CalendarDay(w)
		dayIndex[i] = seen[day]
		seen[day]++
	}

	rows := make([]string, 0, n)
	for i, w := range recent {
		day := models.CalendarDay(w)
		d := models.ParseLocal(w.Date)
		dateLabel := reportShortDate(d)
		pace := models.Pace500m(w)
		paceS := models.Pace500mSeconds(w)
		spm := "-"
		if w.StrokeRate != nil {
			spm = strconv.Itoa(*w.StrokeRate)
		}
		hr := "-"
		var hrValue *int
		if w.HeartRate != nil && w.HeartRate.Average != nil {
			hrValue = w.HeartRate.Average
			hr = strconv.Itoa(*hrValue)
		}

		if dayCounts[day] <= 1 {
			rows = append(rows, `      <tr>
        <td>`+esc(dateLabel)+`</td>
        <td class="r">`+display.FormatMeters(w.Distance)+`m</td>
        <td class="r">`+esc(pace)+`</td>
        <td class="r">`+spm+`</td>
        <td class="r">`+hr+`</td>
      </tr>`)
			continue
		}

		isShort := w.Distance <= 1500
		isHard := paceS > 0 && paceS < 160
		annotation := ""
		rowStyle := ""
		paceStyle := ""
		hrStyle := ""

		switch {
		case isShort && !isHard:
			if dayIndex[i] == dayCounts[day]-1 && dayIndex[i] != 0 {
				annotation = "cooldown"
			} else {
				annotation = "warmup"
			}
			rowStyle = ` style="color:#8b949e;"`
		case isHard:
			annotation = "hard"
			paceStyle = ` style="color:#3fb950;"`
			if hrValue != nil && *hrValue >= 135 {
				hrStyle = ` style="color:#f85149;"`
			}
		}

		dateCell := esc(dateLabel)
		if annotation != "" {
			hardColor := ""
			if isHard {
				hardColor = " color:#3fb950;"
			}
			dateCell = esc(dateLabel) + ` <span style="font-size:10px;` + hardColor + `">(` + annotation + `)</span>`
		}

		rows = append(rows, `      <tr`+rowStyle+`>
        <td>`+dateCell+`</td>
        <td class="r">`+display.FormatMeters(w.Distance)+`m</td>
        <td class="r"`+paceStyle+`>`+esc(pace)+`</td>
        <td class="r">`+spm+`</td>
        <td class="r"`+hrStyle+`>`+hr+`</td>
      </tr>`)
	}

	return `<div class="section">
  <h2>Recent Workouts</h2>
  <table>
    <thead>
      <tr>
        <th>Date</th>
        <th class="r">Distance</th>
        <th class="r">Pace /500m</th>
        <th class="r">SPM</th>
        <th class="r">HR</th>
      </tr>
    </thead>
    <tbody>
` + strings.Join(rows, "\n") + `
    </tbody>
  </table>
</div>`
}

func buildProjection(goal stats.GoalProgress, projection stats.GoalProjection, workouts []models.Workout) string {
	if goal.RemainingWeeks == 0 {
		outcome := "Goal achieved"
		if goal.RemainingMeters > 0 {
			outcome = display.FormatMeters(goal.RemainingMeters) + "m short of goal"
		}
		return `<div class="section"><h2>Goal window ended</h2><p>` + outcome + `</p></div>`
	}
	avgSessionDist := 5000
	if len(workouts) > 0 {
		sum := 0
		for _, w := range workouts {
			sum += w.Distance
		}
		avgSessionDist = int(roundHalfUp(float64(sum) / float64(len(workouts))))
	}
	sessionsPerWeek := "-"
	if avgSessionDist > 0 {
		sessionsPerWeek = display.ToFixed(float64(goal.RequiredPace)/float64(avgSessionDist), 1)
	}
	increaseNeeded := "-"
	if goal.CurrentAvgPace > 0 {
		increaseNeeded = display.ToFixed(float64(goal.RequiredPace-goal.CurrentAvgPace)/float64(goal.CurrentAvgPace)*100, 0)
	}
	increaseVal, increaseErr := strconv.ParseFloat(increaseNeeded, 64)
	increaseLabel := "Pace is sufficient"
	if increaseErr == nil && increaseVal > 0 {
		increaseLabel = "+" + increaseNeeded + "% increase needed"
	}

	currentClass := "red"
	if projection.ShortfallMeters == 0 {
		currentClass = "green"
	}
	shortfallLine := "On track to exceed goal"
	if projection.ShortfallMeters > 0 {
		shortfallLine = display.FormatMeters(projection.ShortfallMeters) + "m short of goal"
	}

	return `<div class="section">
  <h2>Year-End Projection</h2>
  <div class="projection-grid">
    <div class="projection-card">
      <h3 class="` + currentClass + `">At Current Pace</h3>
      <div class="big-num ` + currentClass + `">~` + display.FormatMeters(int(roundHalfUp(float64(projection.ProjectedTotalMeters)/1000))*1000) + `m</div>
      <div class="detail">
        ` + display.FormatMeters(goal.CurrentAvgPace) + ` m/wk &times; ` + formatNumber(projection.RemainingWeeks) + ` weeks remaining + ` + display.FormatMeters(goal.TotalMeters) + `<br>
        ` + shortfallLine + `<br>
        <span class="` + currentClass + `" style="font-weight:600;">` + formatNumber(projection.ProjectedPct) + `% of target</span>
      </div>
    </div>
    <div class="projection-card">
      <h3 class="green">To Hit ` + display.FormatMeters(goal.Target) + `m</h3>
      <div class="big-num green">` + display.FormatMeters(goal.RequiredPace) + ` <span style="font-size:16px; font-weight:400;">m/wk</span></div>
      <div class="detail">
        ` + display.FormatMeters(goal.RemainingMeters) + `m remaining over ` + formatNumber(projection.RemainingWeeks) + ` weeks<br>
        ~` + sessionsPerWeek + ` sessions of ` + display.FormatMeters(avgSessionDist) + `m per week<br>
        <span class="green" style="font-weight:600;">` + increaseLabel + `</span>
      </div>
    </div>
  </div>
</div>`
}

func buildHTML(
	goal stats.GoalProgress,
	projection stats.GoalProjection,
	summaries []stats.WeekSummary,
	allWorkouts []models.Workout,
	windowedWorkouts []models.Workout,
	recentCount int,
	coaching coachingContent,
	today time.Time,
) string {
	sessions := stats.SessionCount(windowedWorkouts)
	avgPace := avgPaceForWorkouts(windowedWorkouts)
	avgHR := avgHRForWorkouts(windowedWorkouts)
	year := strconv.Itoa(today.Year())

	narrativeSection := ""
	if coaching.narrative != nil {
		narrativeSection = buildNarrativeSection(*coaching.narrative)
	}
	notesSection := ""
	if len(coaching.notes) > 0 {
		notesSection = buildNotesSection(coaching.notes)
	}
	planSection := ""
	if coaching.planExcerpt != nil {
		planSection = buildPlanSection(*coaching.planExcerpt)
	}

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Rowing Progress — `)
	b.WriteString(year)
	b.WriteString(reportStyleBlock)
	b.WriteString(year)
	b.WriteString(` Season &mdash; `)
	b.WriteString(display.FormatMeters(goal.Target))
	b.WriteString(`m Goal</div>
  <div class="date">`)
	b.WriteString(reportFullDate(today))
	b.WriteString(`</div>
</header>

`)
	b.WriteString(buildStatsCards(goal, sessions, avgPace, avgHR))
	b.WriteString("\n\n")
	b.WriteString(buildGoalProgress(goal))
	b.WriteString("\n\n")
	b.WriteString(narrativeSection)
	b.WriteString("\n\n")
	b.WriteString(buildWeeklyVolume(summaries, goal.RequiredPace))
	b.WriteString("\n\n")
	b.WriteString(buildWeeklyTrends(summaries))
	b.WriteString("\n\n")
	b.WriteString(buildRecentWorkouts(allWorkouts, recentCount))
	b.WriteString("\n\n")
	b.WriteString(notesSection)
	b.WriteString("\n\n")
	b.WriteString(buildProjection(goal, projection, allWorkouts))
	b.WriteString("\n\n")
	b.WriteString(planSection)
	b.WriteString(`

<div style="text-align: center; color: #484f58; font-size: 12px; margin-top: 32px; padding-bottom: 16px;">
  Generated by c2 &middot; Data from Concept2 Logbook &middot; `)
	b.WriteString(reportFullDate(today))
	b.WriteString(`
</div>

</body>
</html>`)
	return b.String()
}
