package report

import (
	"bytes"
	"html/template"
	"time"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/display"
	"github.com/richhaase/c2/internal/goals"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/notes"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/stats"
	"github.com/richhaase/c2/internal/storage"
)

type ActivityPeriod struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Weeks    int    `json:"weeks"`
	Timezone string `json:"timezone"`
}

type ActivitySummary struct {
	Meters       int      `json:"meters"`
	Workouts     int      `json:"workouts"`
	TrainingDays int      `json:"training_days"`
	AveragePace  *float64 `json:"average_pace_500m_seconds"`
	AverageHR    *int     `json:"average_hr"`
}

type Freshness struct {
	GeneratedAt   string `json:"generated_at"`
	LastSync      string `json:"last_successful_sync,omitempty"`
	LatestWorkout string `json:"latest_workout,omitempty"`
}

type Overview struct {
	Period         ActivityPeriod          `json:"period"`
	Summary        ActivitySummary         `json:"summary"`
	Freshness      Freshness               `json:"freshness"`
	Goals          []goals.Progress        `json:"goals"`
	Weekly         []stats.WeekSummaryData `json:"weekly"`
	RecentWorkouts []display.WorkoutOutput `json:"recent_workouts"`
	LatestSplits   *Splits                 `json:"latest_splits"`
	Narrative      *Narrative              `json:"narrative"`
	Notes          []notes.Record          `json:"notes"`
	PlanExcerpt    *string                 `json:"plan_excerpt"`
}

func BuildOverview(cfg config.Config, p paths.DataPaths, workouts []models.Workout, now time.Time, weeks int) (Overview, error) {
	items, loc, err := goals.Load(p, cfg)
	if err != nil {
		return Overview{}, err
	}
	now = now.In(loc)
	calendarDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	cutoff := stats.MondayOf(calendarDate).AddDate(0, 0, -(weeks-1)*7)
	end := calendarDate.AddDate(0, 0, 1)
	windowed := stats.WorkoutsInRange(workouts, cutoff, end)
	coaching, err := gatherCoaching(p, now)
	if err != nil {
		return Overview{}, err
	}
	summaries := stats.BuildWeekSummaries(workouts, end.Add(-time.Nanosecond), weeks)
	base := buildReportPayload(workouts, windowed, weeks, stats.GoalProgress{}, stats.GoalProjection{}, summaries, coaching)
	o := Overview{
		Period:    ActivityPeriod{From: cutoff.Format("2006-01-02"), To: now.Format("2006-01-02"), Weeks: weeks, Timezone: loc.String()},
		Summary:   ActivitySummary{Workouts: len(windowed), TrainingDays: stats.SessionCount(windowed), AveragePace: base.Summary.AvgPace500mSeconds, AverageHR: base.Summary.AvgHR},
		Freshness: Freshness{GeneratedAt: now.Format(time.RFC3339)},
		Goals:     goals.EvaluateAll(items, workouts, now), Weekly: base.Weekly, RecentWorkouts: base.RecentWorkouts, LatestSplits: base.LatestSplits, Narrative: base.Narrative, Notes: base.Notes, PlanExcerpt: base.PlanExcerpt,
	}
	for _, w := range windowed {
		o.Summary.Meters += w.Distance
	}
	for _, w := range workouts {
		if w.Date > o.Freshness.LatestWorkout {
			o.Freshness.LatestWorkout = w.Date
		}
	}
	if meta := storage.ReadMeta(p, nil); meta != nil {
		o.Freshness.LastSync = meta.LastSync
	}
	return o, nil
}

func RenderOverview(o Overview) (string, error) {
	t, err := template.New("overview").Funcs(template.FuncMap{
		"value": func(kind string, v float64) string {
			if kind == "pace" {
				return models.FormatSeconds(v) + " /500m"
			}
			return models.ToFixed(v, 0) + " m"
		},
		"pace":   models.FormatSeconds,
		"meters": display.FormatMeters,
		"or": func(value, fallback string) string {
			if value == "" {
				return fallback
			}
			return value
		},
	}).Parse(overviewTemplate)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, o); err != nil {
		return "", err
	}
	return b.String(), nil
}

const overviewTemplate = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>C2 — Personal progress</title>
<style>
:root{color-scheme:dark}*{box-sizing:border-box}body{margin:0;background:#10151b;color:#e9edf2;font:16px/1.55 system-ui,sans-serif}main{max-width:1100px;margin:auto;padding:40px 24px}h1{font-size:36px;letter-spacing:-1px;margin:0}h2{margin-top:40px}h3{margin-top:0}.muted,small{color:#a8b5c4}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(230px,1fr));gap:16px}.card{background:#1b2430;border:1px solid #344354;border-radius:12px;padding:20px}.number{font-size:28px;font-weight:650;color:#89d9c5}.label{color:#b2bfce;font-size:14px}table{width:100%;border-collapse:collapse}td,th{text-align:left;padding:10px;border-bottom:1px solid #344354}th{font-size:13px;color:#b2bfce}.table{overflow-x:auto}details{margin-top:14px}summary{cursor:pointer;color:#89d9c5}.text{white-space:pre-wrap;overflow-wrap:anywhere}footer{margin-top:40px;font-size:13px;color:#a8b5c4}.status{font-size:13px;text-transform:uppercase;color:#89d9c5}p{margin:8px 0}code{overflow-wrap:anywhere}@media(max-width:600px){main{padding:24px 16px}h1{font-size:28px}}@media print{body{background:white;color:black}.card{background:white}small,.muted,.label{color:#444}}
</style></head><body><main>
<div class="label">C2 / PERSONAL TRAINING</div><h1>Your progress</h1>
<p class="muted">Last successful sync: {{or .Freshness.LastSync "unknown"}} · Latest workout: {{or .Freshness.LatestWorkout "none recorded"}}</p>
<h2>Recent activity</h2><p class="muted">{{.Period.From}} through {{.Period.To}} (inclusive) · {{.Period.Timezone}}</p>
<div class="grid"><div class="card"><div class="label">Meters in this period</div><div class="number">{{meters .Summary.Meters}}</div></div>
<div class="card"><div class="label">Workouts / training days</div><div class="number">{{.Summary.Workouts}} / {{.Summary.TrainingDays}}</div><small>Each recorded workout / distinct workout dates</small></div>
<div class="card"><div class="label">Average workout pace</div><div class="number">{{if .Summary.AveragePace}}{{pace .Summary.AveragePace}}{{else}}—{{end}}</div><small>Per 500m; each workout weighted equally</small></div>
<div class="card"><div class="label">Average heart rate</div><div class="number">{{if .Summary.AverageHR}}{{.Summary.AverageHR}}{{else}}—{{end}}</div><small>Workouts with recorded average HR</small></div></div>
<h2>Personal goals</h2><p class="muted">Each goal has its own evidence and time window.</p><div class="grid">
{{range .Goals}}<article class="card"><div class="status">{{if .Achieved}}Achieved {{.AchievedOn}}{{else}}In progress{{end}}</div><h3>{{.Goal.Name}}</h3>
<div class="label">Target</div><div class="number">{{value .Goal.Kind .Goal.Target}}</div>
<p>{{if .Value}}Recorded: {{value .Goal.Kind .Value}}{{else}}No qualifying evidence{{end}}</p>
<small>{{or .Goal.From "Available history"}} → {{or .Goal.To "No deadline"}}<br>{{.Goal.Equipment}} · {{.Goal.Effort}} efforts{{if .Goal.MinDistance}} · at least {{.Goal.MinDistance}} m{{end}}</small>
{{if eq .Goal.Kind "pace"}}<p><small>Whole-workout average; interval records exclude rests.</small></p>{{end}}
{{if .Projection}}<p>Projected volume: {{meters .Projection.ProjectedTotalMeters}} m</p><small>Based on recent completed weeks; an estimate, not a guarantee.</small>{{end}}
<details><summary>Evidence and coverage</summary><small>Goal ID: {{.Goal.ID}}<br>Recorded history: {{or .CoverageFrom "none"}} → {{or .CoverageTo "none"}}<br>{{.QualifyingWorkouts}} qualifying workouts</small>
{{if .Evidence}}<p>Workout {{.Evidence.WorkoutID}} · {{.Evidence.Date}}<br>{{.Evidence.Distance}} m · {{pace .Evidence.Pace}} /500m<br>{{.Evidence.Continuity}} · {{.Evidence.WorkoutType}}</p>{{end}}
{{if .UnknownEfforts}}<p>{{.UnknownEfforts}} workouts excluded because continuity could not be established.</p>{{end}}</details></article>
{{else}}<p>No active goals. Your activity and coaching history remain available below.</p>{{end}}</div>
<h2>Weekly activity</h2><div class="table"><table><thead><tr><th>Week starting</th><th>Meters</th><th>Training days</th><th>Average pace /500m</th><th>SPM</th><th>HR</th></tr></thead><tbody>
{{range .Weekly}}<tr><td>{{.WeekStart}}</td><td>{{meters .Meters}}</td><td>{{.Sessions}}</td><td>{{if .AvgPace500m}}{{.AvgPace500m}}{{else}}—{{end}}</td><td>{{if .AvgSPM}}{{.AvgSPM}}{{else}}—{{end}}</td><td>{{if .AvgHR}}{{.AvgHR}}{{else}}—{{end}}</td></tr>{{end}}</tbody></table></div>
<h2>Recent workouts</h2><p class="muted">Latest recorded workouts across all history, independent of the activity window.</p><div class="table"><table><thead><tr><th>Date / ID</th><th>Distance</th><th>Pace /500m</th><th>SPM</th><th>HR</th></tr></thead><tbody>
{{range .RecentWorkouts}}<tr><td>{{.Date}}<br><small>{{.ID}}{{if .Interval}} · interval{{end}}</small></td><td>{{meters .Distance}} m</td><td>{{if .Pace500m}}{{.Pace500m}}{{else}}—{{end}}</td><td>{{if .StrokeRate}}{{.StrokeRate}}{{else}}—{{end}}</td><td>{{if .HRAvg}}{{.HRAvg}}{{else}}—{{end}}</td></tr>{{end}}</tbody></table></div>
{{if .LatestSplits}}<details><summary>Latest workout splits · {{.LatestSplits.WorkoutID}}</summary><div class="table"><table><thead><tr><th>Split</th><th>Distance</th><th>Pace /500m</th><th>SPM</th><th>HR</th></tr></thead><tbody>
{{range .LatestSplits.Splits}}<tr><td>{{.Index}}</td><td>{{if .Distance}}{{.Distance}} m{{else}}—{{end}}</td><td>{{if .Pace500m}}{{.Pace500m}}{{else}}—{{end}}</td><td>{{if .StrokeRate}}{{.StrokeRate}}{{else}}—{{end}}</td><td>{{if .HRAvg}}{{.HRAvg}}{{else}}—{{end}}</td></tr>{{end}}</tbody></table></div></details>{{end}}
{{if .Narrative}}<h2>Coach's report · {{.Narrative.Date}}</h2><div class="card text">{{.Narrative.Text}}</div>{{end}}
{{if .Notes}}<h2>Recent coaching notes</h2>{{range .Notes}}<article class="card"><small>{{.Date}} · {{.Type}} · {{.Author}}{{if .WorkoutID}} · workout {{.WorkoutID}}{{end}}</small><div class="text">{{.Body}}</div></article>{{end}}{{end}}
{{if .PlanExcerpt}}<h2>Training plan</h2><div class="card text">{{.PlanExcerpt}}</div>{{end}}
<footer>Generated {{.Freshness.GeneratedAt}} · C2 · Data from Concept2 Logbook<br>Generation time does not indicate when the data was last synchronized.</footer>
</main></body></html>`
