package report

import (
	"strings"
	"testing"
	"time"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/goals"
	"github.com/richhaase/c2/internal/models"
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
