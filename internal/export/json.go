package export

import (
	"encoding/json"
	"strings"

	"github.com/richhaase/c2/internal/model"
)

func JSON(workouts []model.Workout) (string, error) {
	if workouts == nil {
		workouts = []model.Workout{}
	}
	data, err := json.MarshalIndent(workouts, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

func JSONL(workouts []model.Workout) (string, error) {
	if len(workouts) == 0 {
		return "", nil
	}

	var builder strings.Builder
	for _, workout := range workouts {
		data, err := json.Marshal(workout)
		if err != nil {
			return "", err
		}
		builder.Write(data)
		builder.WriteByte('\n')
	}
	return builder.String(), nil
}
