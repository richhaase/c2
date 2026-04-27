package stats

import (
	"sort"

	"github.com/richhaase/c2/internal/model"
)

type Session struct {
	Date          string
	Workouts      []model.Workout
	TotalDistance int
	TotalTime     int
}

func GroupIntoSessions(workouts []model.Workout) []Session {
	byDay := make(map[string][]model.Workout)
	for _, workout := range workouts {
		day := model.CalendarDay(workout)
		byDay[day] = append(byDay[day], workout)
	}

	sessions := make([]Session, 0, len(byDay))
	for day, dayWorkouts := range byDay {
		sort.Slice(dayWorkouts, func(i, j int) bool {
			return dayWorkouts[i].Date < dayWorkouts[j].Date
		})

		session := Session{
			Date:     day,
			Workouts: dayWorkouts,
		}
		for _, workout := range dayWorkouts {
			session.TotalDistance += workout.Distance
			session.TotalTime += workout.Time
		}
		sessions = append(sessions, session)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Date < sessions[j].Date
	})
	return sessions
}

func SessionCount(workouts []model.Workout) int {
	days := make(map[string]struct{})
	for _, workout := range workouts {
		days[model.CalendarDay(workout)] = struct{}{}
	}
	return len(days)
}
