package report

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/richhaase/c2/internal/analysis"
	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/display"
	"github.com/richhaase/c2/internal/documents"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/notes"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/stats"
)

const (
	recentNoteDays       = 14
	maxRecentNotes       = 20
	planExcerptMaxChars  = 1500
	reportRecentWorkouts = 10
)

var htmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func esc(s string) string {
	return htmlEscaper.Replace(s)
}

func roundHalfUp(v float64) float64 {
	return math.Floor(v + 0.5)
}

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

func avgPaceForWorkouts(workouts []models.Workout) float64 {
	sum := 0.0
	count := 0
	for _, w := range workouts {
		if p := models.Pace500mSeconds(w); p > 0 {
			sum += p
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func avgHRForWorkouts(workouts []models.Workout) int {
	sum := 0
	count := 0
	for _, w := range workouts {
		if w.HeartRate != nil && w.HeartRate.Average != nil && *w.HeartRate.Average > 0 {
			sum += *w.HeartRate.Average
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return int(roundHalfUp(float64(sum) / float64(count)))
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
    <div class="value blue">` + display.FormatMeters(goal.RequiredPace) + ` <span class="unit">m/wk</span></div>
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
	onPacePct := display.ToFixed(float64(goal.WeeksElapsed)/float64(goal.TotalWeeks)*100, 1)
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
    <span class="green">|</span> On-pace target (week ` + strconv.Itoa(goal.WeeksElapsed) + ` of ` + strconv.Itoa(goal.TotalWeeks) + `)
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

func sortedByDateDesc(workouts []models.Workout) []models.Workout {
	sorted := make([]models.Workout, len(workouts))
	copy(sorted, workouts)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[j].Date < sorted[i].Date
	})
	return sorted
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
        ` + display.FormatMeters(goal.RemainingMeters) + `m remaining over ` + strconv.Itoa(goal.RemainingWeeks) + ` weeks<br>
        ~` + sessionsPerWeek + ` sessions of ` + display.FormatMeters(avgSessionDist) + `m per week<br>
        <span class="green" style="font-weight:600;">` + increaseLabel + `</span>
      </div>
    </div>
  </div>
</div>`
}

type Narrative struct {
	Date string `json:"date"`
	Text string `json:"text"`
}

type coachingContent struct {
	narrative   *Narrative
	notes       []notes.Record
	planExcerpt *string
}

func utf16Len(s string) int {
	return len(utf16.Encode([]rune(s)))
}

func utf16Slice(s string, n int) string {
	units := utf16.Encode([]rune(s))
	if n >= len(units) {
		return s
	}
	return string(utf16.Decode(units[:n]))
}

func splitPlanSections(plan string) []string {
	var out []string
	start := 0
	for i := 0; i+3 < len(plan); i++ {
		if plan[i] == '\n' && plan[i+1] == '#' && plan[i+2] == '#' && plan[i+3] == ' ' {
			out = append(out, plan[start:i])
			start = i + 1
		}
	}
	return append(out, plan[start:])
}

var mdHeadingLine = regexp.MustCompile(`^#{1,4}\s`)

func planSectionIsSubstantive(section string) bool {
	for _, line := range strings.Split(section, "\n") {
		t := strings.TrimSpace(line)
		if t != "" && !mdHeadingLine.MatchString(t) && t != "---" {
			return true
		}
	}
	return false
}

func gatherCoaching(p paths.DataPaths, now time.Time) (coachingContent, error) {
	var out coachingContent

	sinceKey := stats.LocalYMD(now.AddDate(0, 0, -recentNoteDays))
	allNotes, err := notes.ReadAll(p)
	if err != nil {
		return out, err
	}
	recent := notes.Apply(allNotes, notes.Filter{Since: sinceKey})
	if len(recent) > maxRecentNotes {
		recent = recent[len(recent)-maxRecentNotes:]
	}
	out.notes = recent

	dates, err := documents.ListNarratives(p)
	if err != nil {
		return out, err
	}
	if len(dates) > 0 {
		latest := dates[len(dates)-1]
		text, ok, err := documents.Read(p.NarrativeFile(latest))
		if err != nil {
			return out, err
		}
		if ok && strings.TrimSpace(text) != "" {
			out.narrative = &Narrative{Date: latest, Text: text}
		}
	}

	plan, ok, err := documents.Read(p.Plan)
	if err != nil {
		return out, err
	}
	if ok && strings.TrimSpace(plan) != "" {
		sections := splitPlanSections(plan)
		end := 1
		excerpt := strings.TrimSpace(sections[0])
		for !planSectionIsSubstantive(excerpt) && end < len(sections) {
			excerpt = excerpt + "\n\n" + strings.TrimSpace(sections[end])
			end++
		}
		if utf16Len(excerpt) > planExcerptMaxChars {
			excerpt = utf16Slice(excerpt, planExcerptMaxChars) + "…"
		} else if len(sections) > end {
			excerpt = excerpt + "\n\n_(full plan: `c2 plan show`)_"
		}
		out.planExcerpt = &excerpt
	}

	return out, nil
}

var (
	mdHeading = regexp.MustCompile(`^(#{1,4})\s+(.*)$`)
	mdItem    = regexp.MustCompile(`^[-*]\s+(.*)$`)
)

func mdLite(text string) string {
	var blocks []string
	var paragraph []string
	var list []string

	flushParagraph := func() {
		if len(paragraph) > 0 {
			blocks = append(blocks, "<p>"+strings.Join(paragraph, " ")+"</p>")
			paragraph = nil
		}
	}
	flushList := func() {
		if len(list) > 0 {
			var items strings.Builder
			for _, i := range list {
				items.WriteString("<li>" + i + "</li>")
			}
			blocks = append(blocks, "<ul>"+items.String()+"</ul>")
			list = nil
		}
	}

	for _, rawLine := range strings.Split(text, "\n") {
		line := esc(strings.TrimSpace(rawLine))
		if line == "" {
			flushParagraph()
			flushList()
			continue
		}
		if m := mdHeading.FindStringSubmatch(line); m != nil {
			flushParagraph()
			flushList()
			level := "h4"
			if len(m[1]) <= 2 {
				level = "h3"
			}
			blocks = append(blocks, "<"+level+">"+m[2]+"</"+level+">")
			continue
		}
		if m := mdItem.FindStringSubmatch(line); m != nil {
			flushParagraph()
			list = append(list, m[1])
			continue
		}
		flushList()
		paragraph = append(paragraph, line)
	}
	flushParagraph()
	flushList()
	return strings.Join(blocks, "\n")
}

func buildNarrativeSection(narrative Narrative) string {
	return `<div class="section">
  <h2>Coach's Report &mdash; ` + esc(narrative.Date) + `</h2>
  <div class="prose">
` + mdLite(narrative.Text) + `
  </div>
</div>`
}

func buildNotesSection(records []notes.Record) string {
	rows := make([]string, 0, len(records))
	for _, n := range records {
		workout := ""
		if n.WorkoutID != nil {
			workout = " &middot; workout " + strconv.FormatInt(*n.WorkoutID, 10)
		}
		rows = append(rows, `  <div class="note-row">
    <div class="note-meta">`+esc(n.Date[:10])+` &middot; `+esc(n.Type)+` (`+esc(n.Author)+`)`+workout+`</div>
    <div class="note-body">`+esc(n.Body)+`</div>
  </div>`)
	}
	return `<div class="section">
  <h2>Recent Notes</h2>
` + strings.Join(rows, "\n") + `
</div>`
}

func buildPlanSection(excerpt string) string {
	return `<div class="section">
  <h2>Training Plan</h2>
  <div class="prose">
` + mdLite(excerpt) + `
  </div>
</div>`
}

const reportStyleBlock = `</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    background: #0d1117;
    color: #c9d1d9;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    line-height: 1.6;
    padding: 24px;
    max-width: 960px;
    margin: 0 auto;
  }
  h1 { color: #f0f6fc; font-size: 28px; font-weight: 700; }
  h2 { color: #f0f6fc; font-size: 20px; font-weight: 600; margin-bottom: 16px; }
  .subtitle { color: #8b949e; font-size: 15px; margin-top: 4px; }
  .date { color: #8b949e; font-size: 13px; margin-top: 2px; }
  .muted { color: #8b949e; }
  .green { color: #3fb950; }
  .red { color: #f85149; }
  .blue { color: #58a6ff; }

  header { margin-bottom: 32px; }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px;
    margin-bottom: 32px;
  }
  .stat-card {
    background: #161b22;
    border: 1px solid #30363d;
    border-radius: 8px;
    padding: 16px;
  }
  .stat-card .label {
    color: #8b949e;
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 4px;
  }
  .stat-card .value {
    color: #f0f6fc;
    font-size: 24px;
    font-weight: 700;
  }
  .stat-card .unit {
    color: #8b949e;
    font-size: 13px;
    font-weight: 400;
  }

  .section {
    background: #161b22;
    border: 1px solid #30363d;
    border-radius: 8px;
    padding: 20px;
    margin-bottom: 24px;
  }

  .prose { font-size: 14px; }
  .prose p { margin-bottom: 10px; }
  .prose h3 { color: #f0f6fc; font-size: 15px; font-weight: 600; margin: 14px 0 6px; }
  .prose h4 { color: #c9d1d9; font-size: 13px; font-weight: 600; margin: 12px 0 4px; }
  .prose ul { margin: 0 0 10px 20px; }
  .prose li { margin-bottom: 4px; }

  .note-row { padding: 10px 0; border-bottom: 1px solid #21262d; }
  .note-row:last-child { border-bottom: none; }
  .note-meta { color: #8b949e; font-size: 11px; text-transform: uppercase; letter-spacing: 0.3px; margin-bottom: 3px; }
  .note-body { font-size: 13px; }

  .progress-container {
    position: relative;
    background: #21262d;
    border-radius: 6px;
    height: 32px;
    margin: 16px 0 8px;
    overflow: visible;
  }
  .progress-fill {
    height: 100%;
    border-radius: 6px;
    background: #f85149;
    position: relative;
    z-index: 1;
    min-width: 2px;
  }
  .progress-marker {
    position: absolute;
    top: -6px;
    height: 44px;
    width: 2px;
    background: #3fb950;
    z-index: 2;
  }
  .progress-marker-label {
    position: absolute;
    top: -22px;
    transform: translateX(-50%);
    font-size: 11px;
    color: #3fb950;
    white-space: nowrap;
    font-weight: 600;
  }
  .progress-label-row {
    display: flex;
    justify-content: space-between;
    font-size: 12px;
    color: #8b949e;
    margin-top: 4px;
  }

  .week-row {
    display: flex;
    align-items: center;
    margin-bottom: 8px;
    font-size: 13px;
  }
  .week-label {
    width: 70px;
    flex-shrink: 0;
    color: #8b949e;
    font-size: 12px;
    text-align: right;
    padding-right: 10px;
  }
  .week-bar-container {
    flex: 1;
    position: relative;
    height: 24px;
    background: #21262d;
    border-radius: 4px;
    overflow: visible;
  }
  .week-bar {
    height: 100%;
    border-radius: 4px;
    min-width: 2px;
  }
  .week-bar.on-pace { background: #238636; }
  .week-bar.behind { background: #8b2a2d; }
  .week-meta {
    width: 140px;
    flex-shrink: 0;
    text-align: right;
    font-size: 12px;
    color: #8b949e;
    padding-left: 8px;
  }
  .week-meta .meters { color: #c9d1d9; font-weight: 500; }
  .week-target-line {
    position: absolute;
    top: -2px;
    height: 28px;
    width: 0;
    border-left: 2px dashed #58a6ff;
    z-index: 2;
    opacity: 0.7;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
  }
  thead th {
    text-align: left;
    color: #8b949e;
    font-weight: 600;
    padding: 8px 10px;
    border-bottom: 1px solid #30363d;
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }
  th.r, td.r { text-align: right; }
  tbody td {
    padding: 8px 10px;
    border-bottom: 1px solid #21262d;
    font-variant-numeric: tabular-nums;
  }
  tbody tr:last-child td { border-bottom: none; }
  tbody tr:hover { background: #1c2128; }

  .projection-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }
  .projection-card {
    background: #21262d;
    border-radius: 6px;
    padding: 16px;
  }
  .projection-card h3 {
    font-size: 14px;
    font-weight: 600;
    margin-bottom: 8px;
  }
  .projection-card .big-num {
    font-size: 28px;
    font-weight: 700;
    margin-bottom: 4px;
  }
  .projection-card .detail {
    font-size: 12px;
    color: #8b949e;
    line-height: 1.8;
  }

  .target-legend {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    color: #8b949e;
    margin-bottom: 12px;
    justify-content: flex-end;
    padding-right: 140px;
  }
  .target-legend-line {
    width: 16px;
    border-top: 2px dashed #58a6ff;
  }

  @media (max-width: 640px) {
    .stats-grid { grid-template-columns: repeat(2, 1fr); }
    .projection-grid { grid-template-columns: 1fr; }
    body { padding: 16px; }
  }
</style>
</head>
<body>

<header>
  <h1>Rowing Progress</h1>
  <div class="subtitle">`

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

type Period struct {
	Weeks int     `json:"weeks"`
	To    *string `json:"to"`
}

type Summary struct {
	TotalMeters        int      `json:"total_meters"`
	Sessions           int      `json:"sessions"`
	AvgPace500mSeconds *float64 `json:"avg_pace_500m_seconds"`
	AvgHR              *int     `json:"avg_hr"`
}

type Splits struct {
	WorkoutID  int64               `json:"workout_id"`
	Date       string              `json:"date"`
	SplitShape analysis.Shape      `json:"split_shape"`
	Splits     []analysis.SplitRow `json:"splits"`
}

type Payload struct {
	Period         Period                  `json:"period"`
	Summary        Summary                 `json:"summary"`
	Goal           stats.GoalProgress      `json:"goal"`
	Projection     stats.GoalProjection    `json:"projection"`
	Weekly         []stats.WeekSummaryData `json:"weekly"`
	RecentWorkouts []display.WorkoutOutput `json:"recent_workouts"`
	LatestSplits   *Splits                 `json:"latest_splits"`
	Narrative      *Narrative              `json:"narrative"`
	Notes          []notes.Record          `json:"notes"`
	PlanExcerpt    *string                 `json:"plan_excerpt"`
}

type Result struct {
	HTML    string
	Payload Payload
}

func Build(cfg config.Config, p paths.DataPaths, workouts []models.Workout, now time.Time, weeks int) (Result, error) {
	goal, err := stats.ComputeGoalProgress(workouts, cfg, now)
	if err != nil {
		return Result{}, err
	}
	end, err := config.ParseGoalDate(cfg.Goal.EndDate)
	if err != nil {
		return Result{}, err
	}
	projection := stats.ProjectGoal(goal, end.AddDate(0, 0, 1), now)
	summaries := stats.BuildWeekSummaries(workouts, now, weeks)
	cutoff := stats.MondayOf(now).AddDate(0, 0, -(weeks-1)*7)
	windowed := stats.WorkoutsInRange(workouts, cutoff, now)
	coaching, err := gatherCoaching(p, now)
	if err != nil {
		return Result{}, err
	}
	return Result{
		HTML:    buildHTML(goal, projection, summaries, workouts, windowed, reportRecentWorkouts, coaching, now),
		Payload: buildReportPayload(workouts, windowed, weeks, goal, projection, summaries, coaching),
	}, nil
}

func buildReportPayload(
	workouts []models.Workout,
	windowed []models.Workout,
	weeks int,
	goal stats.GoalProgress,
	projection stats.GoalProjection,
	summaries []stats.WeekSummary,
	coaching coachingContent,
) Payload {
	sorted := sortedByDateDesc(workouts)

	var period Period
	period.Weeks = weeks
	var splitsSource *models.Workout
	if len(sorted) > 0 {
		day := models.CalendarDay(sorted[0])
		period.To = &day

		candidates := make([]models.Workout, 0, len(sorted))
		for _, w := range sorted {
			if models.CalendarDay(w) != day {
				continue
			}
			if w.Workout == nil || len(w.Workout.Splits) == 0 {
				continue
			}
			candidates = append(candidates, w)
		}
		sort.SliceStable(candidates, func(i, j int) bool {
			return candidates[j].Distance < candidates[i].Distance
		})
		if len(candidates) > 0 {
			splitsSource = &candidates[0]
		}
	}

	var latestSplits *Splits
	if splitsSource != nil {
		rows := analysis.SplitTable(*splitsSource)
		if len(rows) > 0 {
			latestSplits = &Splits{
				WorkoutID:  splitsSource.ID,
				Date:       splitsSource.Date,
				SplitShape: analysis.SplitShape(rows),
				Splits:     rows,
			}
		}
	}

	summary := Summary{
		TotalMeters: goal.TotalMeters,
		Sessions:    stats.SessionCount(windowed),
	}
	if pace := math.Round(avgPaceForWorkouts(windowed)*10) / 10; pace != 0 {
		summary.AvgPace500mSeconds = &pace
	}
	if hr := avgHRForWorkouts(windowed); hr != 0 {
		summary.AvgHR = &hr
	}

	weekly := make([]stats.WeekSummaryData, 0, len(summaries))
	for _, ws := range summaries {
		weekly = append(weekly, stats.WeekSummaryDataOf(ws))
	}

	limit := reportRecentWorkouts
	if limit > len(sorted) {
		limit = len(sorted)
	}
	recent := make([]display.WorkoutOutput, 0, limit)
	for _, w := range sorted[:limit] {
		recent = append(recent, display.WorkoutOutputOf(w))
	}

	records := coaching.notes
	if records == nil {
		records = []notes.Record{}
	}

	return Payload{
		Period:         period,
		Summary:        summary,
		Goal:           goal,
		Projection:     projection,
		Weekly:         weekly,
		RecentWorkouts: recent,
		LatestSplits:   latestSplits,
		Narrative:      coaching.narrative,
		Notes:          records,
		PlanExcerpt:    coaching.planExcerpt,
	}
}
