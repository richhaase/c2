package stats

import (
	"math"
	"time"

	"github.com/richhaase/c2/internal/model"
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

func MondayOf(day time.Time) time.Time {
	d := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	offset := (int(d.Weekday()) + 6) % 7
	return d.AddDate(0, 0, -offset)
}

func WorkoutsInRange(workouts []model.Workout, start time.Time, end time.Time) []model.Workout {
	result := make([]model.Workout, 0)
	for _, workout := range workouts {
		parsed, err := model.ParsedDate(workout)
		if err != nil {
			continue
		}
		if !parsed.Before(start) && parsed.Before(end) {
			result = append(result, workout)
		}
	}
	return result
}

func BuildWeekSummaries(workouts []model.Workout, weeks int, now time.Time) []WeekSummary {
	if weeks <= 0 {
		return nil
	}

	thisMonday := MondayOf(now)
	cutoff := thisMonday.AddDate(0, 0, -(weeks-1)*7)

	summaries := make([]WeekSummary, 0, weeks)
	for i := 0; i < weeks; i++ {
		summaries = append(summaries, WeekSummary{
			WeekStart: thisMonday.AddDate(0, 0, -(weeks-1-i)*7),
		})
	}

	daysByWeek := make(map[int]map[string]struct{})
	for _, workout := range workouts {
		parsed, err := model.ParsedDate(workout)
		if err != nil {
			continue
		}
		if parsed.Before(cutoff) || parsed.After(now) {
			continue
		}

		monday := MondayOf(parsed)
		diffDays := int(math.Round(monday.Sub(cutoff).Hours() / 24))
		idx := diffDays / 7
		if idx < 0 || idx >= weeks {
			continue
		}

		summary := &summaries[idx]
		summary.Meters += workout.Distance

		if _, ok := daysByWeek[idx]; !ok {
			daysByWeek[idx] = make(map[string]struct{})
		}
		daysByWeek[idx][model.CalendarDay(workout)] = struct{}{}

		pace := model.Pace500mSeconds(workout)
		if pace > 0 {
			summary.PaceSum += pace
			summary.PaceCount++
		}
		if workout.StrokeRate != nil && *workout.StrokeRate > 0 {
			summary.SPMSum += *workout.StrokeRate
			summary.SPMCount++
		}
		if workout.HeartRate != nil && workout.HeartRate.Average != nil && *workout.HeartRate.Average > 0 {
			summary.HRSum += *workout.HeartRate.Average
			summary.HRCount++
		}
	}

	for idx, days := range daysByWeek {
		summaries[idx].Sessions = len(days)
	}
	return summaries
}
