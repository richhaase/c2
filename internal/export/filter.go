package export

import (
	"time"

	"github.com/richhaase/c2/internal/model"
)

func FilterByDate(workouts []model.Workout, from string, to string) ([]model.Workout, error) {
	if from == "" && to == "" {
		return workouts, nil
	}
	if from != "" {
		if _, err := time.Parse("2006-01-02", from); err != nil {
			return nil, err
		}
	}
	if to != "" {
		if _, err := time.Parse("2006-01-02", to); err != nil {
			return nil, err
		}
	}

	result := make([]model.Workout, 0, len(workouts))
	for _, workout := range workouts {
		date := model.CalendarDay(workout)
		if from != "" && date < from {
			continue
		}
		if to != "" && date > to {
			continue
		}
		result = append(result, workout)
	}
	return result, nil
}
