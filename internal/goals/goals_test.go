package goals

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/stats"
)

func testGoal(kind string, target float64) Goal {
	return Goal{ID: kind, Name: kind, Kind: kind, Target: target, Equipment: "rower", Effort: "workout"}
}

func TestIndependentAchievementsAndEvidence(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	workouts := []models.Workout{
		{ID: 1, Date: "2026-09-01 09:00:00", Distance: 5000, Time: 15000, Type: "rower", WorkoutType: "FixedDistanceSplits"},
		{ID: 2, Date: "2026-09-02 09:00:00", Distance: 15000, Time: 10000, Type: "bike", WorkoutType: "FixedDistanceSplits"},
		{ID: 3, Date: "2026-09-03 09:00:00", Distance: 12000, Time: 20000, Type: "rower", WorkoutType: "FixedDistanceInterval"},
		{ID: 4, Date: "2026-09-04 09:00:00", Distance: 11000, Time: 20000, Type: "rower"},
	}
	pace := testGoal("pace", 150)
	pace.Effort = "continuous"
	p := Evaluate(pace, workouts, now)
	if !p.Achieved || p.Evidence.WorkoutID != 1 || *p.Value != 150 || p.Projection != nil || p.UnknownEfforts != 1 {
		t.Fatalf("pace: %+v", p)
	}
	distance := testGoal("distance", 10000)
	distance.Effort = "continuous"
	d := Evaluate(distance, workouts, now)
	if d.Achieved || *d.Value != 5000 {
		t.Fatalf("distance: %+v", d)
	}
	pace.MinDistance = 10000
	p = Evaluate(pace, workouts, now)
	if p.Achieved || p.Value != nil {
		t.Fatalf("a shorter pace must not satisfy minimum distance: %+v", p)
	}
	workouts = append(workouts, models.Workout{ID: 5, Date: "2026-09-05", Distance: 10500, Time: 40000, Type: "rower", WorkoutType: "FixedDistanceSplits"})
	d = Evaluate(distance, workouts, now)
	if !d.Achieved || d.Evidence.WorkoutID != 5 {
		t.Fatalf("longer continuous distance qualifies: %+v", d)
	}
}

func TestOptionalDatesAndNoEvidence(t *testing.T) {
	w := []models.Workout{{ID: 1, Date: "2026-01-01 23:59:59", Distance: 1000, Type: "rower"}, {ID: 2, Date: "2026-02-01", Distance: 2000, Type: "rower"}, {ID: 3, Date: "2027-01-01", Distance: 9000, Type: "rower"}}
	now := time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		from, to string
		want     float64
	}{{"", "", 3000}, {"2026-02-01", "", 2000}, {"", "2026-01-01", 1000}, {"2026-01-01", "2026-01-01", 1000}} {
		g := testGoal("volume", 5000)
		g.From, g.To = tc.from, tc.to
		p := Evaluate(g, w, now)
		if *p.Value != tc.want || p.Achieved {
			t.Fatalf("%+v: %+v", tc, p)
		}
		if tc.to == "" && (p.Volume != nil || p.Projection != nil) {
			t.Fatal("undated projection")
		}
	}
	p := Evaluate(testGoal("pace", 150), w, now)
	if p.Value != nil || p.Remaining != nil {
		t.Fatal("missing time cannot establish pace")
	}
}

func TestLegacyMigrationAndPortableCalendar(t *testing.T) {
	loc, err := time.LoadLocation("America/Denver")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{Goal: config.GoalConfig{TargetMeters: 1000000, StartDate: "2026-01-01", EndDate: "2026-12-31"}, Location: loc}
	workouts := []models.Workout{{ID: 1, UserID: 7, Date: "2026-03-08 01:30:00", Type: "rower", Distance: 5000}, {ID: 2, UserID: 7, Date: "2026-03-09 23:30:00", Type: "bike", Distance: 7000}}
	c, err := NewCollection(cfg, "America/Denver", workouts)
	if err != nil {
		t.Fatal(err)
	}
	p := paths.For(t.TempDir())
	if err := Write(p, c); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, loc)
	legacy, err := stats.ComputeGoalProgress(workouts, cfg, now)
	if err != nil {
		t.Fatal(err)
	}
	loaded, zone, err := Load(p, config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	got := Evaluate(loaded[0], workouts, now.UTC().In(zone))
	if got.Volume == nil || *got.Volume != legacy || got.Goal.Equipment != "all" {
		t.Fatalf("legacy totals changed: %+v vs %+v", got.Volume, legacy)
	}
	c.Goals = []Goal{}
	if err := Write(p, c); err != nil {
		t.Fatal(err)
	}
	loaded, _, err = Load(p, cfg)
	if err != nil || len(loaded) != 0 {
		t.Fatal("empty collection must not resurrect legacy goal")
	}
	if err := os.WriteFile(File(p), []byte(`{"version":1,"timezone":"UTC"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(p); err == nil {
		t.Fatal("missing array accepted")
	}
}

func TestAccountChecks(t *testing.T) {
	for _, tc := range []struct {
		c   *Collection
		w   []models.Workout
		id  int64
		bad bool
	}{
		{&Collection{AccountID: 1}, nil, 2, true}, {nil, []models.Workout{{ID: 3, UserID: 1}}, 2, true}, {nil, []models.Workout{{ID: 3}}, 1, true}, {nil, nil, 0, true}, {&Collection{AccountID: 1}, []models.Workout{{UserID: 1}}, 1, false},
	} {
		if err := CheckAccount(tc.c, tc.w, tc.id); (err != nil) != tc.bad {
			t.Fatalf("check %+v: %v", tc, err)
		}
	}
}

func TestCalendarDatesIndependentOfHostTimezone(t *testing.T) {
	prior := time.Local
	t.Cleanup(func() { time.Local = prior })
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	workouts := []models.Workout{{ID: 1, Date: "2026-09-06 12:00:00", Distance: 5000, Time: 15000, Type: "rower"}}
	for _, kind := range []string{"volume", "distance", "pace"} {
		g := testGoal(kind, 5000)
		if kind == "pace" {
			g.Target = 150
		}
		g.From, g.To = "2026-09-06", "2026-09-06"
		time.Local = time.UTC
		want := Evaluate(g, workouts, now)
		if !want.Achieved || want.QualifyingWorkouts != 1 {
			t.Fatalf("missing baseline achievement: %+v", want)
		}
		for _, zone := range []string{"UTC", "America/Santiago", "Pacific/Apia"} {
			t.Run(kind+"/"+zone, func(t *testing.T) {
				loc, err := time.LoadLocation(zone)
				if err != nil {
					t.Fatal(err)
				}
				time.Local = loc
				if err := Validate(g); err != nil {
					t.Errorf("valid calendar bounds rejected: %v", err)
				}
				if got := Evaluate(g, workouts, now); !reflect.DeepEqual(got, want) {
					t.Errorf("host timezone changed progress: %+v; want %+v", got, want)
				}
				skippedOnHost := g
				skippedOnHost.From, skippedOnHost.To = "2011-12-30", "2011-12-30"
				if err := Validate(skippedOnHost); err != nil {
					t.Errorf("calendar date skipped only on host rejected: %v", err)
				}
			})
		}
	}
}
