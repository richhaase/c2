package display

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/richhaase/c2/internal/models"
)

const (
	barWidth       = 20
	trendThreshold = 0.02
)

func FormatMeters(m int) string {
	sign := ""
	if m < 0 {
		sign = "-"
	}
	digits := strconv.FormatInt(int64(m), 10)
	digits = strings.TrimPrefix(digits, "-")

	lead := len(digits) % 3
	if lead == 0 {
		lead = 3
	}
	groups := []string{digits[:lead]}
	for i := lead; i < len(digits); i += 3 {
		groups = append(groups, digits[i:i+3])
	}
	return sign + strings.Join(groups, ",")
}

func FormatPercent(ratio float64) string {
	return ToFixed(ratio*100, 1) + "%"
}

func ToFixed(v float64, digits int) string {
	return models.ToFixed(v, digits)
}

func FormatMetersPerWeek(m int) string {
	return FormatMeters(m) + "m/week"
}

func FormatDate(d time.Time, format string) string {
	if format == "%Y-%m-%d" {
		return fmt.Sprintf("%d-%02d-%02d", d.Year(), int(d.Month()), d.Day())
	}
	return fmt.Sprintf("%02d/%02d", int(d.Month()), d.Day())
}

func FormatIntervalTag(w models.Workout) string {
	if !models.IsIntervalWorkout(w) {
		return ""
	}
	rest := models.RestSeconds(w)
	if rest > 0 {
		return fmt.Sprintf("[IVL rest %s]", models.FormatSeconds(rest))
	}
	return "[IVL]"
}

func FormatWorkoutLine(w models.Workout, dateFormat string) string {
	dateStr := FormatDate(models.ParsedDate(w), dateFormat)
	distance := FormatMeters(w.Distance) + "m"
	pace := models.Pace500m(w)

	spm := "-"
	if w.StrokeRate != nil && *w.StrokeRate != 0 {
		spm = fmt.Sprintf("%dspm", *w.StrokeRate)
	}

	hr := "-"
	if w.HeartRate != nil && w.HeartRate.Average != nil && *w.HeartRate.Average > 0 {
		hr = fmt.Sprintf("%dbpm", *w.HeartRate.Average)
	}

	df := "-"
	if w.DragFactor != nil && *w.DragFactor != 0 {
		df = fmt.Sprintf("%ddf", *w.DragFactor)
	}

	tagSuffix := ""
	if tag := FormatIntervalTag(w); tag != "" {
		tagSuffix = "  " + tag
	}

	return fmt.Sprintf("%s  %7s  %8s  %7s/500m  %5s  %6s  %4s%s",
		dateStr, distance, w.TimeFormatted, pace, spm, hr, df, tagSuffix)
}

type WorkoutOutput struct {
	ID              int64    `json:"id"`
	Date            string   `json:"date"`
	Distance        int      `json:"distance"`
	TimeTenths      int      `json:"time_tenths"`
	TimeFormatted   string   `json:"time_formatted"`
	Pace500m        *string  `json:"pace_500m"`
	Pace500mSeconds *float64 `json:"pace_500m_seconds"`
	StrokeRate      *int     `json:"stroke_rate"`
	HRAvg           *int     `json:"hr_avg"`
	DragFactor      *int     `json:"drag_factor"`
	WorkoutType     *string  `json:"workout_type"`
	Interval        bool     `json:"interval"`
	RestSeconds     *float64 `json:"rest_seconds"`
	Comments        *string  `json:"comments"`
	HasStrokeData   bool     `json:"has_stroke_data"`
}

func WorkoutJSON(w models.Workout) WorkoutOutput {
	out := WorkoutOutput{
		ID:            w.ID,
		Date:          w.Date,
		Distance:      w.Distance,
		TimeTenths:    w.Time,
		TimeFormatted: w.TimeFormatted,
		StrokeRate:    w.StrokeRate,
		DragFactor:    w.DragFactor,
		Interval:      models.IsIntervalWorkout(w),
		HasStrokeData: w.StrokeData,
	}

	if paceSecs := models.Pace500mSeconds(w); paceSecs > 0 {
		pace := models.Pace500m(w)
		rounded := math.Round(paceSecs*10) / 10
		out.Pace500m = &pace
		out.Pace500mSeconds = &rounded
	}

	if w.HeartRate != nil {
		out.HRAvg = w.HeartRate.Average
	}

	if w.WorkoutType != "" {
		workoutType := w.WorkoutType
		out.WorkoutType = &workoutType
	}

	if rest := models.RestSeconds(w); rest > 0 {
		out.RestSeconds = &rest
	}

	if w.Comments != "" {
		comments := w.Comments
		out.Comments = &comments
	}

	return out
}

func SparkBar(value, max float64) string {
	if max == 0 {
		return ""
	}
	filled := int(math.Round(value / max * barWidth))
	return strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
}

func TrendArrow(prev, curr float64) string {
	if prev == 0 {
		return " "
	}
	diff := (curr - prev) / prev
	if diff > trendThreshold {
		return "↑"
	}
	if diff < -trendThreshold {
		return "↓"
	}
	return "→"
}

func PaceArrow(prev, curr float64) string {
	if prev == 0 {
		return " "
	}
	return TrendArrow(curr, prev)
}
