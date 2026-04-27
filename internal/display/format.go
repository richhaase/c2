package display

import (
	"fmt"
	"strconv"
	"time"

	"github.com/richhaase/c2/internal/model"
)

func FormatMeters(n int) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}

	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return sign + s
}

func FormatPercent(v float64) string {
	return fmt.Sprintf("%.1f%%", v*100)
}

func FormatMetersPerWeek(n int) string {
	return fmt.Sprintf("%sm/week", FormatMeters(n))
}

func FormatDate(day time.Time, format string) string {
	switch format {
	case "01/02", "%m/%d":
		return day.Format("01/02")
	case "%Y-%m-%d":
		return day.Format("2006-01-02")
	default:
		return day.Format("01/02")
	}
}

func FormatIntervalTag(w model.Workout) string {
	if !model.IsIntervalWorkout(w) {
		return ""
	}
	rest := model.RestSeconds(w)
	if rest > 0 {
		return fmt.Sprintf("[IVL rest %s]", model.FormatSeconds(rest))
	}
	return "[IVL]"
}

func FormatWorkoutLine(w model.Workout, dateFormat string) string {
	parsed, err := model.ParsedDate(w)
	if err != nil {
		parsed = time.Time{}
	}

	spm := "-"
	if w.StrokeRate != nil && *w.StrokeRate > 0 {
		spm = fmt.Sprintf("%dspm", *w.StrokeRate)
	}

	hr := "-"
	if w.HeartRate != nil && w.HeartRate.Average != nil && *w.HeartRate.Average > 0 {
		hr = fmt.Sprintf("%dbpm", *w.HeartRate.Average)
	}

	drag := "-"
	if w.DragFactor != nil && *w.DragFactor > 0 {
		drag = fmt.Sprintf("%ddf", *w.DragFactor)
	}

	tag := FormatIntervalTag(w)
	tagSuffix := ""
	if tag != "" {
		tagSuffix = "  " + tag
	}

	return fmt.Sprintf(
		"%s  %7s  %8s  %7s/500m  %5s  %6s  %4s%s",
		FormatDate(parsed, dateFormat),
		fmt.Sprintf("%sm", FormatMeters(w.Distance)),
		w.TimeFormatted,
		model.Pace500m(w),
		spm,
		hr,
		drag,
		tagSuffix,
	)
}
