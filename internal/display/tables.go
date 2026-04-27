package display

import (
	"math"
	"strings"
)

const trendThreshold = 0.02

func SparkBar(value int, max int) string {
	if max == 0 {
		return ""
	}
	const barWidth = 20
	filled := int(math.Round((float64(value) / float64(max)) * barWidth))
	if filled < 0 {
		filled = 0
	}
	if filled > barWidth {
		filled = barWidth
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
}

func TrendArrow(prev int, curr int) string {
	return trendArrow(float64(prev), float64(curr))
}

func PaceArrow(prev float64, curr float64) string {
	if prev == 0 {
		return " "
	}
	return trendArrow(curr, prev)
}

func trendArrow(prev float64, curr float64) string {
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
