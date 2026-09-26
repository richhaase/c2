package report

import (
	"strings"
	"testing"
	"time"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/goals"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/paths"
)

func TestOverviewSharesEvidenceAndLabelsActualPeriods(t *testing.T) {
	cfg, p, workouts, now := reportFixture(t)
	c, err := goals.NewCollection(cfg, "America/Denver", workouts)
	if err != nil {
		t.Fatal(err)
	}
	c.Goals = append(c.Goals, goals.Goal{ID: "pace", Name: "Pace <script>alert(1)</script>", Kind: "pace", Target: 300, Equipment: "all", Effort: "workout"})
	if err := goals.Write(p, c); err != nil {
		t.Fatal(err)
	}
	workouts = append(workouts, models.Workout{ID: 4, Date: "2026-01-01", Distance: 50000, Time: 100000})
	o, err := BuildOverview(cfg, p, workouts, now, 2)
	if err != nil {
		t.Fatal(err)
	}
	if o.Summary.Meters != 8000 || o.Summary.Workouts != 3 || o.Summary.TrainingDays != 2 {
		t.Fatalf("mixed periods: %+v", o.Summary)
	}
	if *o.Goals[0].Value != 58000 || !o.Goals[1].Achieved {
		t.Fatalf("goals: %+v", o.Goals)
	}
	if o.Period.To == models.CalendarDay(workouts[2]) || o.Freshness.LatestWorkout != workouts[2].Date {
		t.Fatalf("period/freshness: %+v", o)
	}
	html, err := RenderOverview(o)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"8,000", "3 / 2", "&lt;script&gt;", "Hold &lt;145 bpm.", "Whole-workout average", "No deadline", o.Period.From, o.Period.To} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q", want)
		}
	}
	if strings.Contains(html, "<script>") {
		t.Fatal("unsafe coaching or goal HTML")
	}
	o2, err := BuildOverview(cfg, p, workouts, now.In(time.FixedZone("Elsewhere", 9*3600)), 2)
	if err != nil {
		t.Fatal(err)
	}
	if o.Period != o2.Period || o.Summary.Meters != o2.Summary.Meters {
		t.Fatal("host timezone changed report")
	}
}

func TestOverviewWithoutGoalsOrWorkouts(t *testing.T) {
	_, p, _, now := reportFixture(t)
	c := goals.Collection{Version: 1, Timezone: "UTC", Goals: []goals.Goal{}}
	if err := goals.Write(p, c); err != nil {
		t.Fatal(err)
	}
	o, err := BuildOverview(config.Config{}, p, nil, now, 4)
	if err != nil {
		t.Fatal(err)
	}
	html, err := RenderOverview(o)
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Goals) != 0 || o.Summary.Workouts != 0 || !strings.Contains(html, "No active goals") {
		t.Fatal("goal-free report not useful")
	}
}

func TestLegacyReportVolumeAndProjectionIncludeDSTDeadline(t *testing.T) {
	loc, err := time.LoadLocation("America/Santiago")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{Location: loc, Goal: config.GoalConfig{TargetMeters: 50000, StartDate: "2026-08-01", EndDate: "2026-09-06"}}
	workouts := []models.Workout{
		{ID: 1, Date: "2026-08-24 12:00:00", Distance: 28000, Time: 84000},
		{ID: 2, Date: "2026-09-06 23:30:00", Distance: 5000, Time: 15000},
	}
	now := time.Date(2026, 9, 6, 23, 45, 0, 0, loc)
	r, err := Build(cfg, paths.For(t.TempDir()), workouts, now, 4)
	if err != nil {
		t.Fatal(err)
	}
	if r.Payload.Goal.TotalMeters != 33000 || r.Payload.Projection.ProjectedTotalMeters != 33010 {
		t.Fatalf("legacy report truncated deadline: goal=%+v projection=%+v", r.Payload.Goal, r.Payload.Projection)
	}
}

func TestOverviewIncludesWholeCalendarDaysAcrossMidnightDST(t *testing.T) {
	loc, err := time.LoadLocation("America/Santiago")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, today, from string
		weeks             int
	}{
		{"before skipped midnight", "2026-09-05", "2026-08-31", 1},
		{"day with skipped midnight", "2026-09-06", "2026-08-31", 1},
		{"next week", "2026-09-07", "2026-09-07", 1},
		{"two weeks", "2026-09-07", "2026-08-31", 2},
		{"repeated midnight", "2026-04-04", "2026-03-30", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := paths.For(t.TempDir())
			if err := goals.Write(p, goals.Collection{Version: 1, Timezone: loc.String(), Goals: []goals.Goal{}}); err != nil {
				t.Fatal(err)
			}
			now, err := time.ParseInLocation("2006-01-02 15:04:05", tc.today+" 23:45:00", loc)
			if err != nil {
				t.Fatal(err)
			}
			first, err := time.Parse("2006-01-02", tc.from)
			if err != nil {
				t.Fatal(err)
			}
			last, err := time.Parse("2006-01-02", tc.today)
			if err != nil {
				t.Fatal(err)
			}
			workouts := []models.Workout{
				{ID: 1, Date: first.AddDate(0, 0, -1).Format("2006-01-02") + " 23:59:59", Distance: 90000, Time: 270000},
				{ID: 2, Date: tc.from + " 00:00:00", Distance: 1000, Time: 3000},
				{ID: 3, Date: tc.today + " 01:30:00", Distance: 2000, Time: 6000},
				{ID: 4, Date: tc.today + " 23:30:00", Distance: 5000, Time: 15000},
				{ID: 5, Date: last.AddDate(0, 0, 1).Format("2006-01-02") + " 01:00:00", Distance: 90000, Time: 270000},
			}
			o, err := BuildOverview(config.Config{}, p, workouts, now.UTC(), tc.weeks)
			if err != nil {
				t.Fatal(err)
			}
			days := 2
			if tc.from == tc.today {
				days = 1
			}
			if o.Period.From != tc.from || o.Period.To != tc.today || o.Period.Timezone != loc.String() {
				t.Fatalf("wrong period: %+v", o.Period)
			}
			if o.Summary.Meters != 8000 || o.Summary.Workouts != 3 || o.Summary.TrainingDays != days {
				t.Fatalf("incomplete calendar window: %+v", o.Summary)
			}
			meters, sessions := 0, 0
			for _, week := range o.Weekly {
				meters += week.Meters
				sessions += week.Sessions
			}
			if len(o.Weekly) != tc.weeks || o.Weekly[0].WeekStart != tc.from || meters != 8000 || sessions != days {
				t.Fatalf("weekly activity disagrees with period: %+v", o.Weekly)
			}
			html, err := RenderOverview(o)
			if err != nil || !strings.Contains(html, tc.from+" through "+tc.today+" (inclusive)") || !strings.Contains(html, "8,000") {
				t.Fatalf("report does not show the inclusive calendar totals: %v", err)
			}
		})
	}
}
