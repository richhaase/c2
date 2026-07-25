package stats

import (
	"fmt"
	"math"
	"time"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/models"
)

type WeekSummary struct {
	WeekStart time.Time
	Meters    int
	Sessions  int
	PaceSum   float64
	PaceCount int
	SPMSum    int
	SPMCount  int
	HRSum     int
	HRCount   int
}

type GoalProgress struct {
	Target          int     `json:"target"`
	TotalMeters     int     `json:"totalMeters"`
	Progress        float64 `json:"progress"`
	WeeksElapsed    int     `json:"weeksElapsed"`
	TotalWeeks      int     `json:"totalWeeks"`
	RemainingMeters int     `json:"remainingMeters"`
	RemainingWeeks  int     `json:"remainingWeeks"`
	RequiredPace    int     `json:"requiredPace"`
	CurrentAvgPace  int     `json:"currentAvgPace"`
	OnPace          bool    `json:"onPace"`
}

type RecentWeek struct {
	WeekStart time.Time
	Meters    int
	Sessions  int
}

type WeekSummaryData struct {
	WeekStart          string   `json:"week_start"`
	Meters             int      `json:"meters"`
	Sessions           int      `json:"sessions"`
	AvgPace500mSeconds *float64 `json:"avg_pace_500m_seconds"`
	AvgPace500m        *string  `json:"avg_pace_500m"`
	AvgSPM             *float64 `json:"avg_spm"`
	AvgHR              *int     `json:"avg_hr"`
}

type GoalProjection struct {
	RemainingWeeks       float64 `json:"remaining_weeks"`
	ProjectedTotalMeters int     `json:"projected_total_meters"`
	ProjectedPct         float64 `json:"projected_pct"`
	ShortfallMeters      int     `json:"shortfall_meters"`
}

const (
	recentPaceWeeks = 4
	daysPerWeek     = 7
	hoursPerWeek    = 24 * daysPerWeek
	secondsPerDay   = 24 * 60 * 60
)

func SessionCount(workouts []models.Workout) int {
	days := make(map[string]struct{}, len(workouts))
	for _, w := range workouts {
		days[models.CalendarDay(w)] = struct{}{}
	}
	return len(days)
}

func MondayOf(t time.Time) time.Time {
	y, m, d := t.In(time.Local).Date()
	midnight := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	offset := (int(midnight.Weekday()) + 6) % daysPerWeek
	return midnight.AddDate(0, 0, -offset)
}

func WorkoutsInRange(workouts []models.Workout, from, to time.Time) []models.Workout {
	out := make([]models.Workout, 0, len(workouts))
	for _, w := range workouts {
		t := models.ParsedDate(w)
		if !t.Before(from) && t.Before(to) {
			out = append(out, w)
		}
	}
	return out
}

func BuildWeekSummaries(workouts []models.Workout, now time.Time, weeks int) []WeekSummary {
	thisMonday := MondayOf(now)
	cutoff := thisMonday.AddDate(0, 0, -(weeks-1)*daysPerWeek)

	summaries := make([]WeekSummary, 0, weeks)
	for i := 0; i < weeks; i++ {
		summaries = append(summaries, WeekSummary{
			WeekStart: thisMonday.AddDate(0, 0, -(weeks-1-i)*daysPerWeek),
		})
	}

	daysByWeek := make(map[int]map[string]struct{})

	for _, w := range workouts {
		t := models.ParsedDate(w)
		if t.Before(cutoff) || t.After(now) {
			continue
		}

		idx := floorDiv(dayNumber(MondayOf(t))-dayNumber(cutoff), daysPerWeek)
		if idx < 0 || idx >= weeks {
			continue
		}

		ws := &summaries[idx]
		ws.Meters += w.Distance

		days, ok := daysByWeek[idx]
		if !ok {
			days = make(map[string]struct{})
			daysByWeek[idx] = days
		}
		days[models.CalendarDay(w)] = struct{}{}

		if pace := models.Pace500mSeconds(w); pace > 0 {
			ws.PaceSum += pace
			ws.PaceCount++
		}
		if w.StrokeRate != nil && *w.StrokeRate > 0 {
			ws.SPMSum += *w.StrokeRate
			ws.SPMCount++
		}
		if w.HeartRate != nil && w.HeartRate.Average != nil && *w.HeartRate.Average > 0 {
			ws.HRSum += *w.HeartRate.Average
			ws.HRCount++
		}
	}

	for idx, days := range daysByWeek {
		summaries[idx].Sessions = len(days)
	}

	return summaries
}

func RecentWeeks(workouts []models.Workout, now time.Time, count int) []RecentWeek {
	out := make([]RecentWeek, 0, count)
	for i := 0; i < count; i++ {
		weekStart := MondayOf(now).AddDate(0, 0, -i*daysPerWeek)
		weekEnd := weekStart.AddDate(0, 0, daysPerWeek)
		weekWorkouts := WorkoutsInRange(workouts, weekStart, weekEnd)
		meters := 0
		for _, w := range weekWorkouts {
			meters += w.Distance
		}
		out = append(out, RecentWeek{
			WeekStart: weekStart,
			Meters:    meters,
			Sessions:  SessionCount(weekWorkouts),
		})
	}
	return out
}

func LocalYMD(t time.Time) string {
	y, m, d := t.In(time.Local).Date()
	return fmt.Sprintf("%04d-%02d-%02d", y, int(m), d)
}

func WeekSummaryDataOf(ws WeekSummary) WeekSummaryData {
	out := WeekSummaryData{
		WeekStart: LocalYMD(ws.WeekStart),
		Meters:    ws.Meters,
		Sessions:  ws.Sessions,
	}
	if ws.PaceCount > 0 {
		avgPace := ws.PaceSum / float64(ws.PaceCount)
		seconds := math.Round(avgPace*10) / 10
		formatted := models.FormatSeconds(avgPace)
		out.AvgPace500mSeconds = &seconds
		out.AvgPace500m = &formatted
	}
	if ws.SPMCount > 0 {
		spm := math.Round(float64(ws.SPMSum)/float64(ws.SPMCount)*10) / 10
		out.AvgSPM = &spm
	}
	if ws.HRCount > 0 {
		hr := int(math.Round(float64(ws.HRSum) / float64(ws.HRCount)))
		out.AvgHR = &hr
	}
	return out
}

func ProjectGoal(goal GoalProgress, end, now time.Time) GoalProjection {
	weeksLeft := calendarDaysBetween(now, end) / daysPerWeek
	return projectGoal(goal, weeksLeft)
}

func ProjectGoalByElapsedTime(goal GoalProgress, end, now time.Time) GoalProjection {
	weeksLeft := end.Sub(now).Hours() / hoursPerWeek
	return projectGoal(goal, weeksLeft)
}

func projectGoal(goal GoalProgress, weeksLeft float64) GoalProjection {
	if weeksLeft < 0 {
		weeksLeft = 0
	}
	projected := int(math.Round(float64(goal.TotalMeters) + float64(goal.CurrentAvgPace)*weeksLeft))
	shortfall := goal.Target - projected
	if shortfall < 0 {
		shortfall = 0
	}
	projectedPct := 0.0
	if goal.Target > 0 {
		projectedPct = math.Round(float64(projected)/float64(goal.Target)*1000) / 10
	}
	return GoalProjection{
		RemainingWeeks:       math.Round(weeksLeft*10) / 10,
		ProjectedTotalMeters: projected,
		ProjectedPct:         projectedPct,
		ShortfallMeters:      shortfall,
	}
}

func ComputeGoalProgress(workouts []models.Workout, cfg config.Config, now time.Time) (GoalProgress, error) {
	target := cfg.Goal.TargetMeters
	if target <= 0 {
		return GoalProgress{}, fmt.Errorf("Goal target must be a positive number of meters.")
	}
	start, err := config.ParseGoalDate(cfg.Goal.StartDate)
	if err != nil {
		return GoalProgress{}, err
	}
	end, err := config.ParseGoalDate(cfg.Goal.EndDate)
	if err != nil {
		return GoalProgress{}, err
	}
	if end.Before(start) {
		return GoalProgress{}, fmt.Errorf("Goal end date must not be before start date.")
	}
	endExclusive := end.AddDate(0, 0, 1)
	today := now
	if today.IsZero() {
		today = time.Now()
	}

	totalMeters := 0
	for _, w := range workouts {
		t := models.ParsedDate(w)
		if !t.Before(start) && t.Before(endExclusive) {
			totalMeters += w.Distance
		}
	}

	progress := float64(totalMeters) / float64(target)
	totalDays := dayNumber(endExclusive) - dayNumber(start)
	totalWeeks := int(math.Ceil(float64(totalDays) / daysPerWeek))

	weeksElapsed := 0
	if today.After(start) {
		weeksElapsed = floorDiv(dayNumber(today)-dayNumber(start), daysPerWeek)
		if weeksElapsed > totalWeeks {
			weeksElapsed = totalWeeks
		}
	}

	remainingMeters := target - totalMeters
	if remainingMeters < 0 {
		remainingMeters = 0
	}
	remainingWeeks := totalWeeks - weeksElapsed
	if remainingWeeks < 1 {
		remainingWeeks = 1
	}
	requiredPace := int(math.Floor(float64(remainingMeters) / float64(remainingWeeks)))

	currentAvgPace := 0
	if weeksElapsed > 0 {
		thisMonday := MondayOf(today)
		windowStart := thisMonday.AddDate(0, 0, -recentPaceWeeks*daysPerWeek)
		if windowStart.Before(start) {
			windowStart = start
		}
		weeksInWindow := math.Round(float64(dayNumber(thisMonday)-dayNumber(windowStart)) / daysPerWeek)
		if weeksInWindow < 1 {
			weeksInWindow = 1
		}
		recentMeters := 0
		for _, w := range workouts {
			t := models.ParsedDate(w)
			if !t.Before(windowStart) && t.Before(thisMonday) {
				recentMeters += w.Distance
			}
		}
		currentAvgPace = int(math.Floor(float64(recentMeters) / weeksInWindow))
	}
	targetWeekly := float64(target) / float64(totalWeeks)

	return GoalProgress{
		Target:          target,
		TotalMeters:     totalMeters,
		Progress:        progress,
		WeeksElapsed:    weeksElapsed,
		TotalWeeks:      totalWeeks,
		RemainingMeters: remainingMeters,
		RemainingWeeks:  remainingWeeks,
		RequiredPace:    requiredPace,
		CurrentAvgPace:  currentAvgPace,
		OnPace:          float64(currentAvgPace) >= targetWeekly,
	}, nil
}

func dayNumber(t time.Time) int {
	y, m, d := t.In(time.Local).Date()
	return int(time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix() / secondsPerDay)
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

func calendarDaysBetween(from, to time.Time) float64 {
	return float64(dayNumber(to)-dayNumber(from)) + dayFraction(to) - dayFraction(from)
}

func dayFraction(t time.Time) float64 {
	local := t.In(time.Local)
	seconds := local.Hour()*60*60 + local.Minute()*60 + local.Second()
	return (float64(seconds) + float64(local.Nanosecond())/float64(time.Second)) / secondsPerDay
}
