package stats

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/models"
)

func makeWorkout(id int64, date string, distance int) models.Workout {
	return models.Workout{
		ID:            id,
		UserID:        1,
		Date:          date,
		Distance:      distance,
		Type:          "rower",
		Time:          int(math.Round(float64(distance) * 3.5)),
		TimeFormatted: "0:00.0",
	}
}

func makeGoalConfig(t *testing.T) config.Config {
	t.Helper()
	cfg, err := config.Default()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Goal.StartDate = "2026-01-01"
	cfg.Goal.EndDate = "2026-12-31"
	cfg.Goal.TargetMeters = 1_000_000
	return cfg
}

func localDate(y int, m time.Month, d, hour int) time.Time {
	return time.Date(y, m, d, hour, 0, 0, 0, time.Local)
}

func goalProgressAt(t *testing.T, workouts []models.Workout, cfg config.Config, now time.Time) GoalProgress {
	t.Helper()
	goal, err := ComputeGoalProgress(workouts, cfg, now)
	if err != nil {
		t.Fatalf("ComputeGoalProgress: %v", err)
	}
	return goal
}

func TestMondayOfMondayReturnsSameDate(t *testing.T) {
	d := localDate(2026, time.March, 2, 0)
	m := MondayOf(d)
	if m.Weekday() != time.Monday {
		t.Errorf("weekday = %v, want Monday", m.Weekday())
	}
	if m.Day() != 2 {
		t.Errorf("day = %d, want 2", m.Day())
	}
}

func TestMondayOfWednesdayReturnsPreviousMonday(t *testing.T) {
	d := localDate(2026, time.March, 4, 0)
	m := MondayOf(d)
	if m.Weekday() != time.Monday {
		t.Errorf("weekday = %v, want Monday", m.Weekday())
	}
	if m.Day() != 2 {
		t.Errorf("day = %d, want 2", m.Day())
	}
}

func TestMondayOfSundayReturnsPreviousMonday(t *testing.T) {
	d := localDate(2026, time.March, 8, 0)
	m := MondayOf(d)
	if m.Weekday() != time.Monday {
		t.Errorf("weekday = %v, want Monday", m.Weekday())
	}
	if m.Day() != 2 {
		t.Errorf("day = %d, want 2", m.Day())
	}
}

func TestMondayOfHandlesMonthBoundary(t *testing.T) {
	d := localDate(2026, time.March, 1, 0)
	m := MondayOf(d)
	if m.Weekday() != time.Monday {
		t.Errorf("weekday = %v, want Monday", m.Weekday())
	}
	if m.After(d) {
		t.Errorf("monday %v is after %v", m, d)
	}
}

func TestWorkoutsInRangeFiltersWithinDateRange(t *testing.T) {
	workouts := []models.Workout{
		makeWorkout(1, "2026-01-15 10:00:00", 5000),
		makeWorkout(2, "2026-02-15 10:00:00", 5000),
		makeWorkout(3, "2026-03-15 10:00:00", 5000),
	}
	from := localDate(2026, time.February, 1, 0)
	to := localDate(2026, time.March, 1, 0)
	result := WorkoutsInRange(workouts, from, to)
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
	if result[0].ID != 2 {
		t.Errorf("id = %d, want 2", result[0].ID)
	}
}

func TestWorkoutsInRangeReturnsEmptyForNoMatches(t *testing.T) {
	workouts := []models.Workout{makeWorkout(1, "2026-06-15 10:00:00", 5000)}
	from := localDate(2026, time.January, 1, 0)
	to := localDate(2026, time.February, 1, 0)
	if got := WorkoutsInRange(workouts, from, to); len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestBuildWeekSummariesBucketsWorkoutsIntoCorrectWeeks(t *testing.T) {
	now := localDate(2026, time.March, 7, 0)
	workouts := []models.Workout{
		makeWorkout(1, "2026-03-02 10:00:00", 5000),
		makeWorkout(2, "2026-02-23 10:00:00", 6000),
	}
	summaries := BuildWeekSummaries(workouts, now, 2)
	if len(summaries) != 2 {
		t.Fatalf("len = %d, want 2", len(summaries))
	}
	if summaries[0].Meters != 6000 {
		t.Errorf("summaries[0].Meters = %d, want 6000", summaries[0].Meters)
	}
	if summaries[1].Meters != 5000 {
		t.Errorf("summaries[1].Meters = %d, want 5000", summaries[1].Meters)
	}
}

func TestBuildWeekSummariesCountsSessionsAsUniqueDays(t *testing.T) {
	now := localDate(2026, time.March, 7, 0)
	workouts := []models.Workout{
		makeWorkout(1, "2026-03-02 09:00:00", 1000),
		makeWorkout(2, "2026-03-02 10:00:00", 2000),
		makeWorkout(3, "2026-03-04 10:00:00", 3000),
	}
	summaries := BuildWeekSummaries(workouts, now, 1)
	if summaries[0].Meters != 6000 {
		t.Errorf("meters = %d, want 6000", summaries[0].Meters)
	}
	if summaries[0].Sessions != 2 {
		t.Errorf("sessions = %d, want 2", summaries[0].Sessions)
	}
}

func TestBuildWeekSummariesReturnsEmptySummariesForNoWorkouts(t *testing.T) {
	now := localDate(2026, time.March, 7, 0)
	summaries := BuildWeekSummaries(nil, now, 4)
	if len(summaries) != 4 {
		t.Fatalf("len = %d, want 4", len(summaries))
	}
	for i, s := range summaries {
		if s.Meters != 0 {
			t.Errorf("summaries[%d].Meters = %d, want 0", i, s.Meters)
		}
	}
}

func TestComputeGoalProgressMidSeason(t *testing.T) {
	cfg := makeGoalConfig(t)
	workouts := []models.Workout{
		makeWorkout(1, "2026-03-01 10:00:00", 100_000),
		makeWorkout(2, "2026-02-01 10:00:00", 100_000),
	}
	goal := goalProgressAt(t, workouts, cfg, localDate(2026, time.March, 7, 0))
	if goal.TotalMeters != 200_000 {
		t.Errorf("TotalMeters = %d, want 200000", goal.TotalMeters)
	}
	if goal.Target != 1_000_000 {
		t.Errorf("Target = %d, want 1000000", goal.Target)
	}
	if math.Abs(goal.Progress-0.2) > 0.005 {
		t.Errorf("Progress = %v, want ~0.2", goal.Progress)
	}
	if goal.RemainingMeters != 800_000 {
		t.Errorf("RemainingMeters = %d, want 800000", goal.RemainingMeters)
	}
	if goal.WeeksElapsed <= 0 {
		t.Errorf("WeeksElapsed = %d, want > 0", goal.WeeksElapsed)
	}
}

func TestGoalPaceUsesRemainingCalendarTime(t *testing.T) {
	cfg := makeGoalConfig(t)
	cfg.Location = time.UTC
	workouts := []models.Workout{
		makeWorkout(1, "2026-08-01 10:00:00", 562375),
		makeWorkout(2, "2026-08-31 10:00:00", 22750),
		makeWorkout(3, "2026-09-07 10:00:00", 22750),
		makeWorkout(4, "2026-09-14 10:00:00", 22750),
		makeWorkout(5, "2026-09-21 10:00:00", 22750),
	}
	now := time.Date(2026, time.September, 28, 0, 0, 0, 0, time.UTC)
	goal := goalProgressAt(t, workouts, cfg, now)
	if goal.RequiredPace != 25541 || goal.RemainingWeeks != 14 {
		t.Fatalf("required pace = %d, remaining week buckets = %d", goal.RequiredPace, goal.RemainingWeeks)
	}
	if goal.OnPace {
		t.Fatal("annual average must not mark an insufficient recent pace on track")
	}
	end := time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	projection := ProjectGoal(goal, start, end, now)
	if projection.RemainingWeeks != 13.6 || projection.ProjectedTotalMeters != 962125 {
		t.Fatalf("projection = %+v", projection)
	}
	goal.CurrentAvgPace = goal.RequiredPace
	if got := ProjectGoal(goal, start, end, now); got.ShortfallMeters != 0 {
		t.Fatalf("required pace does not reach the target: %+v", got)
	}
}

func TestGoalPaceBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name       string
		at         string
		meters     int
		weeks      int
		required   int
		onPace     bool
		projection float64
	}{
		{"before start", "2026-12-01 00:00:00", 0, 1, 21000, false, 0.3},
		{"first day", "2026-12-30 00:00:00", 0, 1, 21000, false, 0.3},
		{"partial last day", "2026-12-31 12:00:00", 1000, 1, 70000, false, 0.1},
		{"expired", "2027-01-01 00:00:00", 1000, 0, 0, false, 0},
		{"achieved", "2026-12-31 12:00:00", 6000, 1, 0, true, 0.1},
		{"achieved and ended", "2027-01-01 00:00:00", 6000, 0, 0, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := makeGoalConfig(t)
			cfg.Location = time.UTC
			cfg.Goal.StartDate = "2026-12-30"
			cfg.Goal.EndDate = "2026-12-31"
			cfg.Goal.TargetMeters = 6000
			now, err := time.Parse("2006-01-02 15:04:05", tc.at)
			if err != nil {
				t.Fatal(err)
			}
			goal := goalProgressAt(t, []models.Workout{makeWorkout(1, "2026-12-30 10:00:00", tc.meters)}, cfg, now)
			if goal.RequiredPace != tc.required || goal.RemainingWeeks != tc.weeks || goal.OnPace != tc.onPace {
				t.Fatalf("goal = %+v; want required=%d weeks=%d onPace=%t", goal, tc.required, tc.weeks, tc.onPace)
			}
			start := time.Date(2026, time.December, 30, 0, 0, 0, 0, time.UTC)
			end := time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)
			projection := ProjectGoal(goal, start, end, now)
			if projection.RemainingWeeks != tc.projection || (projection.ShortfallMeters == 0) != goal.OnPace {
				t.Fatalf("projection and pace disagree: %+v, %+v", projection, goal)
			}
		})
	}
}

func TestGoalOnPaceRecognizesAccumulatedProgress(t *testing.T) {
	cfg := makeGoalConfig(t)
	cfg.Location = time.UTC
	workouts := []models.Workout{
		makeWorkout(1, "2026-08-01 10:00:00", 940000),
		makeWorkout(2, "2026-08-31 10:00:00", 5000),
		makeWorkout(3, "2026-09-07 10:00:00", 5000),
		makeWorkout(4, "2026-09-14 10:00:00", 5000),
		makeWorkout(5, "2026-09-21 10:00:00", 5000),
	}
	goal := goalProgressAt(t, workouts, cfg, time.Date(2026, time.September, 28, 0, 0, 0, 0, time.UTC))
	if !goal.OnPace || goal.RequiredPace != 2948 {
		t.Fatalf("banked progress should permit a lower recent pace: %+v", goal)
	}
}

func TestComputeGoalProgressClampsRemainingMetersWhenGoalExceeded(t *testing.T) {
	cfg := makeGoalConfig(t)
	cfg.Goal.TargetMeters = 100_000
	workouts := []models.Workout{makeWorkout(1, "2026-03-01 10:00:00", 150_000)}
	goal := goalProgressAt(t, workouts, cfg, localDate(2026, time.March, 7, 0))
	if goal.RemainingMeters != 0 {
		t.Errorf("RemainingMeters = %d, want 0", goal.RemainingMeters)
	}
	if goal.Progress <= 1 {
		t.Errorf("Progress = %v, want > 1", goal.Progress)
	}
}

func TestComputeGoalProgressExcludesWorkoutsOutsideGoalDateRange(t *testing.T) {
	cfg := makeGoalConfig(t)
	workouts := []models.Workout{
		makeWorkout(1, "2025-12-01 10:00:00", 50_000),
		makeWorkout(2, "2026-03-01 10:00:00", 100_000),
		makeWorkout(3, "2027-02-01 10:00:00", 50_000),
	}
	goal := goalProgressAt(t, workouts, cfg, localDate(2026, time.March, 7, 0))
	if goal.TotalMeters != 100_000 {
		t.Errorf("TotalMeters = %d, want 100000", goal.TotalMeters)
	}
}

func TestComputeGoalProgressIncludesEntireEndDate(t *testing.T) {
	cfg := makeGoalConfig(t)
	cfg.Goal.StartDate = "2026-12-31"
	cfg.Goal.EndDate = "2026-12-31"
	workouts := []models.Workout{
		makeWorkout(1, "2026-12-31 23:59:59", 10_000),
		makeWorkout(2, "2027-01-01 00:00:00", 20_000),
	}
	goal := goalProgressAt(t, workouts, cfg, localDate(2026, time.December, 31, 12))
	if goal.TotalMeters != 10_000 {
		t.Fatalf("TotalMeters = %d, want 10000", goal.TotalMeters)
	}
	if goal.TotalWeeks != 1 {
		t.Fatalf("TotalWeeks = %d, want 1", goal.TotalWeeks)
	}
}

func TestGoalProgressCalendarMathAcrossMidnightDST(t *testing.T) {
	loc, err := time.LoadLocation("America/Santiago")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ from, to, at string }{
		{"2026-09-01", "2026-09-06", "2026-09-06 23:45:00"},
		{"2026-09-06", "2026-10-06", "2026-09-14 12:00:00"},
		{"2026-09-06", "2026-09-06", "2026-09-07 00:00:00"},
	} {
		t.Run(tc.from+"/"+tc.to, func(t *testing.T) {
			cfg := makeGoalConfig(t)
			cfg.Goal.StartDate, cfg.Goal.EndDate = tc.from, tc.to
			workouts := []models.Workout{
				makeWorkout(1, "2026-08-31 23:30:00", 90000),
				makeWorkout(2, "2026-09-05 23:30:00", 1000),
				makeWorkout(3, "2026-09-06 23:30:00", 5000),
				makeWorkout(4, "2026-09-07 01:30:00", 7000),
			}
			now, err := time.ParseInLocation("2006-01-02 15:04:05", tc.at, loc)
			if err != nil {
				t.Fatal(err)
			}
			calendarNow, err := time.Parse("2006-01-02 15:04:05", tc.at)
			if err != nil {
				t.Fatal(err)
			}
			cfg.Location = time.UTC
			want := goalProgressAt(t, workouts, cfg, calendarNow)
			cfg.Location = loc
			got := goalProgressAt(t, workouts, cfg, now.UTC())
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("DST changed goal totals or pace: %+v; want %+v", got, want)
			}
		})
	}
}

func TestComputeGoalProgressCompletesPartialFinalWeek(t *testing.T) {
	tests := []struct {
		name  string
		start string
		end   string
		now   time.Time
		weeks int
	}{
		{
			name:  "one day",
			start: "2026-12-31",
			end:   "2026-12-31",
			now:   localDate(2027, time.January, 1, 0),
			weeks: 1,
		},
		{
			name:  "calendar year",
			start: "2026-01-01",
			end:   "2026-12-31",
			now:   localDate(2027, time.January, 1, 0),
			weeks: 53,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := makeGoalConfig(t)
			cfg.Goal.StartDate = test.start
			cfg.Goal.EndDate = test.end
			goal := goalProgressAt(t, nil, cfg, test.now)
			if goal.TotalWeeks != test.weeks || goal.WeeksElapsed != test.weeks {
				t.Fatalf("weeks = %d / %d, want %d / %d",
					goal.WeeksElapsed, goal.TotalWeeks, test.weeks, test.weeks)
			}
		})
	}
}

func TestComputeGoalProgressRejectsInvalidGoalBounds(t *testing.T) {
	cfg := makeGoalConfig(t)
	cfg.Goal.TargetMeters = 0
	if _, err := ComputeGoalProgress(nil, cfg, projectNow); err == nil || !strings.Contains(err.Error(), "positive") {
		t.Fatalf("target error = %v", err)
	}

	cfg = makeGoalConfig(t)
	cfg.Goal.StartDate = "2026-12-31"
	cfg.Goal.EndDate = "2026-01-01"
	if _, err := ComputeGoalProgress(nil, cfg, projectNow); err == nil || !strings.Contains(err.Error(), "before") {
		t.Fatalf("date error = %v", err)
	}
}

func TestGoalCalendarMathIgnoresDSTHourChanges(t *testing.T) {
	location, err := time.LoadLocation("America/Denver")
	if err != nil {
		t.Fatal(err)
	}
	oldLocal := time.Local
	time.Local = location
	t.Cleanup(func() {
		time.Local = oldLocal
	})

	cfg := makeGoalConfig(t)
	cfg.Goal.StartDate = "2026-03-02"
	cfg.Goal.EndDate = "2026-03-29"
	now := time.Date(2026, time.March, 9, 0, 0, 0, 0, location)
	goal := goalProgressAt(t, nil, cfg, now)
	if goal.WeeksElapsed != 1 {
		t.Fatalf("WeeksElapsed = %d, want 1", goal.WeeksElapsed)
	}

	end := time.Date(2026, time.March, 16, 0, 0, 0, 0, location)
	projection := ProjectGoal(goalFixture(), now, end, now)
	if projection.RemainingWeeks != 1 {
		t.Fatalf("RemainingWeeks = %v, want 1", projection.RemainingWeeks)
	}
}

func TestComputeGoalProgressBeforeStartDate(t *testing.T) {
	cfg := makeGoalConfig(t)
	cfg.Goal.StartDate = "2026-06-01"
	goal := goalProgressAt(t, nil, cfg, localDate(2026, time.March, 7, 0))
	if goal.WeeksElapsed != 0 {
		t.Errorf("WeeksElapsed = %d, want 0", goal.WeeksElapsed)
	}
	if goal.CurrentAvgPace != 0 {
		t.Errorf("CurrentAvgPace = %d, want 0", goal.CurrentAvgPace)
	}
}

func TestComputeGoalProgressCurrentAvgPaceUsesRecentFourWeekWindow(t *testing.T) {
	cfg := makeGoalConfig(t)
	workouts := []models.Workout{
		makeWorkout(1, "2026-01-05 10:00:00", 5_000),
		makeWorkout(2, "2026-01-12 10:00:00", 8_000),
		makeWorkout(3, "2026-01-19 10:00:00", 11_000),
		makeWorkout(4, "2026-03-09 10:00:00", 20_000),
		makeWorkout(5, "2026-03-16 10:00:00", 20_000),
		makeWorkout(6, "2026-03-23 10:00:00", 20_000),
		makeWorkout(7, "2026-03-30 10:00:00", 20_000),
	}
	goal := goalProgressAt(t, workouts, cfg, localDate(2026, time.April, 6, 18))
	if goal.CurrentAvgPace != 20_000 {
		t.Errorf("CurrentAvgPace = %d, want 20000", goal.CurrentAvgPace)
	}
}

func TestComputeGoalProgressCurrentAvgPaceIgnoresWorkoutsBeforeWindow(t *testing.T) {
	cfg := makeGoalConfig(t)
	workouts := []models.Workout{
		makeWorkout(1, "2026-01-05 10:00:00", 100_000),
		makeWorkout(2, "2026-03-23 10:00:00", 15_000),
		makeWorkout(3, "2026-03-30 10:00:00", 15_000),
	}
	goal := goalProgressAt(t, workouts, cfg, localDate(2026, time.April, 6, 18))
	if goal.CurrentAvgPace != 7_500 {
		t.Errorf("CurrentAvgPace = %d, want 7500", goal.CurrentAvgPace)
	}
}

func TestComputeGoalProgressCurrentAvgPaceExcludesCurrentWeek(t *testing.T) {
	cfg := makeGoalConfig(t)
	workouts := []models.Workout{
		makeWorkout(1, "2026-04-06 10:00:00", 20_000),
		makeWorkout(2, "2026-04-13 06:55:00", 5_500),
	}
	goal := goalProgressAt(t, workouts, cfg, localDate(2026, time.April, 13, 18))
	if goal.CurrentAvgPace != 5_000 {
		t.Errorf("CurrentAvgPace = %d, want 5000", goal.CurrentAvgPace)
	}
}

func TestSessionCountCountsUniqueCalendarDays(t *testing.T) {
	workouts := []models.Workout{
		makeWorkout(1, "2026-03-07 09:21:00", 1000),
		makeWorkout(2, "2026-03-07 09:45:00", 2500),
		makeWorkout(3, "2026-03-07 09:53:00", 1000),
		makeWorkout(4, "2026-03-05 14:00:00", 5000),
	}
	if got := SessionCount(workouts); got != 2 {
		t.Errorf("SessionCount = %d, want 2", got)
	}
}

func TestSessionCountReturnsZeroForEmptyList(t *testing.T) {
	if got := SessionCount(nil); got != 0 {
		t.Errorf("SessionCount = %d, want 0", got)
	}
}

func goalFixture() GoalProgress {
	return GoalProgress{
		Target:          1_000_000,
		TotalMeters:     900_000,
		Progress:        0.9,
		WeeksElapsed:    26,
		TotalWeeks:      52,
		RemainingMeters: 100_000,
		RemainingWeeks:  26,
		RequiredPace:    3846,
		CurrentAvgPace:  20_000,
		OnPace:          true,
	}
}

var projectNow = time.Date(2026, time.July, 6, 12, 0, 0, 0, time.UTC)

func daysAfterNow(n float64) time.Time {
	return projectNow.Add(time.Duration(n * 24 * float64(time.Hour)))
}

func TestProjectGoalProjectsOverRemainingCalendarDays(t *testing.T) {
	active := ProjectGoal(goalFixture(), daysAfterNow(-26*7), daysAfterNow(26*7), projectNow)
	if want := 900_000 + 26*20_000; active.ProjectedTotalMeters != want {
		t.Errorf("ProjectedTotalMeters = %d, want %d", active.ProjectedTotalMeters, want)
	}
	if active.ShortfallMeters != 0 {
		t.Errorf("ShortfallMeters = %d, want 0", active.ShortfallMeters)
	}
}

func TestProjectGoalAddsNothingAfterGoalWindowEnds(t *testing.T) {
	goal := goalFixture()
	goal.WeeksElapsed = 60
	goal.RemainingWeeks = 1
	expired := ProjectGoal(goal, daysAfterNow(-26*7), daysAfterNow(-2), projectNow)
	if expired.ProjectedTotalMeters != 900_000 {
		t.Errorf("ProjectedTotalMeters = %d, want 900000", expired.ProjectedTotalMeters)
	}
	if expired.ProjectedPct != 90 {
		t.Errorf("ProjectedPct = %v, want 90", expired.ProjectedPct)
	}
	if expired.ShortfallMeters != 100_000 {
		t.Errorf("ShortfallMeters = %d, want 100000", expired.ShortfallMeters)
	}
}

func TestProjectGoalHandlesFractionalFinalWeeks(t *testing.T) {
	halfWeek := ProjectGoal(goalFixture(), daysAfterNow(-26*7), daysAfterNow(3.5), projectNow)
	if want := 900_000 + 10_000; halfWeek.ProjectedTotalMeters != want {
		t.Errorf("ProjectedTotalMeters = %d, want %d", halfWeek.ProjectedTotalMeters, want)
	}
}

func TestWeekSummaryDataOfEmitsNullsForMissingAverages(t *testing.T) {
	data := WeekSummaryDataOf(WeekSummary{
		WeekStart: localDate(2026, time.March, 2, 0),
		Meters:    6000,
		Sessions:  2,
	})
	out, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"week_start":"2026-03-02","meters":6000,"sessions":2,` +
		`"avg_pace_500m_seconds":null,"avg_pace_500m":null,"avg_spm":null,"avg_hr":null}`
	if string(out) != want {
		t.Errorf("json = %s, want %s", out, want)
	}
}

func TestWeekSummaryDataOfRoundsAverages(t *testing.T) {
	spm := 24
	hr := 141
	workout := makeWorkout(1, "2026-03-02 10:00:00", 6000)
	workout.Time = 15000
	workout.StrokeRate = &spm
	workout.HeartRate = &models.HeartRate{Average: &hr}
	summaries := BuildWeekSummaries([]models.Workout{workout}, localDate(2026, time.March, 7, 0), 1)
	data := WeekSummaryDataOf(summaries[0])
	if data.AvgPace500mSeconds == nil || *data.AvgPace500mSeconds != 125 {
		t.Errorf("AvgPace500mSeconds = %v, want 125", data.AvgPace500mSeconds)
	}
	if data.AvgPace500m == nil || *data.AvgPace500m != "2:05.0" {
		t.Errorf("AvgPace500m = %v, want 2:05.0", data.AvgPace500m)
	}
	if data.AvgSPM == nil || *data.AvgSPM != 24 {
		t.Errorf("AvgSPM = %v, want 24", data.AvgSPM)
	}
	if data.AvgHR == nil || *data.AvgHR != 141 {
		t.Errorf("AvgHR = %v, want 141", data.AvgHR)
	}
}

func TestGoalProgressJSONUsesCamelCaseKeys(t *testing.T) {
	out, err := json.Marshal(goalFixture())
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"target":1000000,"totalMeters":900000,"progress":0.9,"weeksElapsed":26,` +
		`"totalWeeks":52,"remainingMeters":100000,"remainingWeeks":26,"requiredPace":3846,` +
		`"currentAvgPace":20000,"onPace":true}`
	if string(out) != want {
		t.Errorf("json = %s, want %s", out, want)
	}
}

func TestGoalProjectionJSONKeys(t *testing.T) {
	out, err := json.Marshal(ProjectGoal(goalFixture(), daysAfterNow(-26*7), daysAfterNow(26*7), projectNow))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"remaining_weeks":26,"projected_total_meters":1420000,` +
		`"projected_pct":142,"shortfall_meters":0}`
	if string(out) != want {
		t.Errorf("json = %s, want %s", out, want)
	}
}

func TestRecentWeeksSplitsByWeek(t *testing.T) {
	workouts := []models.Workout{
		makeWorkout(1, "2026-03-02 09:00:00", 1000),
		makeWorkout(2, "2026-03-02 10:00:00", 2000),
		makeWorkout(3, "2026-02-24 10:00:00", 4000),
	}
	weeks := RecentWeeks(workouts, localDate(2026, time.March, 7, 0), 2)
	if len(weeks) != 2 {
		t.Fatalf("len = %d, want 2", len(weeks))
	}
	if weeks[0].Meters != 3000 || weeks[0].Sessions != 1 {
		t.Errorf("weeks[0] = %+v, want 3000m/1 session", weeks[0])
	}
	if weeks[1].Meters != 4000 || weeks[1].Sessions != 1 {
		t.Errorf("weeks[1] = %+v, want 4000m/1 session", weeks[1])
	}
	if got := LocalYMD(weeks[1].WeekStart); got != "2026-02-23" {
		t.Errorf("weeks[1].WeekStart = %s, want 2026-02-23", got)
	}
}
