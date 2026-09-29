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
	ElapsedFraction float64 `json:"-"`
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
	y, m, d := t.Date()
	midnight := time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	offset := (int(midnight.Weekday()) + 6) % daysPerWeek
	return midnight.AddDate(0, 0, -offset)
}

func WorkoutsInRange(workouts []models.Workout, from, to time.Time) []models.Workout {
	out := make([]models.Workout, 0, len(workouts))
	for _, w := range workouts {
		t := models.ParseInLocation(w.Date, from.Location())
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
		t := models.ParseInLocation(w.Date, now.Location())
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
		ws.Sessions = len(days)

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
	y, m, d := t.Date()
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

func WeekSummariesDataOf(summaries []WeekSummary) []WeekSummaryData {
	out := make([]WeekSummaryData, 0, len(summaries))
	for _, ws := range summaries {
		out = append(out, WeekSummaryDataOf(ws))
	}
	return out
}

func ProjectGoal(goal GoalProgress, start, end, now time.Time) GoalProjection {
	weeksLeft := remainingGoalWeeks(start, end, now)
	projected := goal.TotalMeters + int(math.Floor(float64(goal.CurrentAvgPace)*weeksLeft))
	projectedPct := 0.0
	if goal.Target > 0 {
		projectedPct = math.Round(float64(projected)/float64(goal.Target)*1000) / 10
	}
	return GoalProjection{
		RemainingWeeks:       math.Round(weeksLeft*10) / 10,
		ProjectedTotalMeters: projected,
		ProjectedPct:         projectedPct,
		ShortfallMeters:      max(0, goal.Target-projected),
	}
}

func ComputeGoalProgress(workouts []models.Workout, cfg config.Config, now time.Time) (GoalProgress, error) {
	target := cfg.Goal.TargetMeters
	if target <= 0 {
		return GoalProgress{}, fmt.Errorf("Goal target must be a positive number of meters.")
	}
	loc := cfg.Location
	if loc == nil {
		loc = time.Local
	}
	now = now.In(loc)
	start, err := time.Parse("2006-01-02", cfg.Goal.StartDate)
	if err != nil {
		return GoalProgress{}, err
	}
	end, err := time.Parse("2006-01-02", cfg.Goal.EndDate)
	if err != nil {
		return GoalProgress{}, err
	}
	if end.Before(start) {
		return GoalProgress{}, fmt.Errorf("Goal end date must not be before start date.")
	}
	endExclusive := end.AddDate(0, 0, 1)
	today := now
	if today.IsZero() {
		today = time.Now().In(loc)
	}
	today = time.Date(today.Year(), today.Month(), today.Day(), today.Hour(), today.Minute(), today.Second(), today.Nanosecond(), time.UTC)

	totalMeters := 0
	for _, w := range workouts {
		t := models.ParseInLocation(w.Date, time.UTC)
		if !t.Before(start) && t.Before(endExclusive) {
			totalMeters += w.Distance
		}
	}

	progress := float64(totalMeters) / float64(target)
	totalDays := dayNumber(endExclusive) - dayNumber(start)
	totalWeeks := int(math.Ceil(float64(totalDays) / daysPerWeek))

	weeksElapsed := 0
	if !today.Before(endExclusive) {
		weeksElapsed = totalWeeks
	} else if today.After(start) {
		weeksElapsed = floorDiv(dayNumber(today)-dayNumber(start), daysPerWeek)
	}

	remainingMeters := max(0, target-totalMeters)
	weeksLeft := remainingGoalWeeks(start, endExclusive, today)
	remainingWeeks := int(math.Ceil(weeksLeft))
	requiredPace := 0
	if weeksLeft > 0 {
		requiredPace = int(math.Ceil(float64(remainingMeters) / weeksLeft))
	}

	currentAvgPace := 0
	if weeksElapsed > 0 {
		thisMonday := MondayOf(today)
		windowStart := thisMonday.AddDate(0, 0, -recentPaceWeeks*daysPerWeek)
		if windowStart.Before(start) {
			windowStart = start
		}
		weeksInWindow := max(1, math.Round(float64(dayNumber(thisMonday)-dayNumber(windowStart))/daysPerWeek))
		recentMeters := 0
		for _, w := range workouts {
			t := models.ParseInLocation(w.Date, time.UTC)
			if !t.Before(windowStart) && t.Before(thisMonday) {
				recentMeters += w.Distance
			}
		}
		currentAvgPace = int(math.Floor(float64(recentMeters) / weeksInWindow))
	}
	onPace := remainingMeters == 0 || weeksLeft > 0 && currentAvgPace >= requiredPace
	elapsedFraction := math.Max(0, math.Min(1, 1-weeksLeft*daysPerWeek/float64(totalDays)))

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
		OnPace:          onPace,
		ElapsedFraction: elapsedFraction,
	}, nil
}

func dayNumber(t time.Time) int {
	y, m, d := t.Date()
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

func remainingGoalWeeks(start, end, now time.Time) float64 {
	return math.Max(0, math.Min(calendarDaysBetween(start, end), calendarDaysBetween(now, end))/daysPerWeek)
}

func dayFraction(t time.Time) float64 {
	seconds := t.Hour()*60*60 + t.Minute()*60 + t.Second()
	return (float64(seconds) + float64(t.Nanosecond())/float64(time.Second)) / secondsPerDay
}
