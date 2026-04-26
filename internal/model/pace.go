package model

import "fmt"

const (
	tenthsPerSecond = 10.0
	paceDistance    = 500.0
)

func Pace500mSeconds(w Workout) float64 {
	if w.Distance == 0 || w.Time == 0 {
		return 0
	}
	return (float64(w.Time) / tenthsPerSecond) * (paceDistance / float64(w.Distance))
}

func Pace500m(w Workout) string {
	secs := Pace500mSeconds(w)
	if secs == 0 {
		return "-"
	}
	return FormatSeconds(secs)
}

func RestSeconds(w Workout) float64 {
	if w.RestTime == nil {
		return 0
	}
	return float64(*w.RestTime) / tenthsPerSecond
}

func WorkSeconds(w Workout) float64 {
	return float64(w.Time) / tenthsPerSecond
}

func FormatSeconds(totalSeconds float64) string {
	if totalSeconds <= 0 {
		return "0:00.0"
	}
	mins := int(totalSeconds / 60)
	rem := totalSeconds - float64(mins*60)
	return fmt.Sprintf("%d:%04.1f", mins, rem)
}
