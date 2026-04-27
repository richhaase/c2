package export

import (
	"strconv"
	"strings"

	"github.com/richhaase/c2/internal/model"
)

var csvHeader = []string{
	"id",
	"date",
	"distance",
	"time_tenths",
	"time_formatted",
	"pace_500m",
	"stroke_rate",
	"stroke_count",
	"calories",
	"drag_factor",
	"hr_avg",
	"hr_min",
	"hr_max",
	"workout_type",
	"rest_time_tenths",
	"rest_distance",
	"machine_type",
	"comments",
}

func CSV(workouts []model.Workout) (string, error) {
	var builder strings.Builder
	writeCSVLine(&builder, csvHeader)
	for _, workout := range workouts {
		writeCSVLine(&builder, csvRow(workout))
	}
	return builder.String(), nil
}

func writeCSVLine(builder *strings.Builder, row []string) {
	for i, field := range row {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(escapeCSV(field))
	}
	builder.WriteByte('\n')
}

func escapeCSV(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

func csvRow(w model.Workout) []string {
	return []string{
		strconv.Itoa(w.ID),
		w.Date,
		strconv.Itoa(w.Distance),
		strconv.Itoa(w.Time),
		w.TimeFormatted,
		model.Pace500m(w),
		optionalInt(w.StrokeRate, true),
		optionalInt(w.StrokeCount, true),
		optionalInt(w.CaloriesTotal, true),
		optionalInt(w.DragFactor, true),
		optionalInt(heartRateAverage(w), false),
		optionalInt(heartRateMin(w), false),
		optionalInt(heartRateMax(w), false),
		w.WorkoutType,
		optionalInt(w.RestTime, true),
		optionalInt(w.RestDistance, true),
		w.Type,
		w.Comments,
	}
}

func optionalInt(value *int, keepZero bool) string {
	if value == nil {
		return ""
	}
	if !keepZero && *value == 0 {
		return ""
	}
	return strconv.Itoa(*value)
}

func heartRateAverage(w model.Workout) *int {
	if w.HeartRate == nil {
		return nil
	}
	return w.HeartRate.Average
}

func heartRateMin(w model.Workout) *int {
	if w.HeartRate == nil {
		return nil
	}
	return w.HeartRate.Min
}

func heartRateMax(w model.Workout) *int {
	if w.HeartRate == nil {
		return nil
	}
	return w.HeartRate.Max
}
